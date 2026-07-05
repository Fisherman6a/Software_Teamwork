package attachmentclient

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/qa/internal/service"
)

type RuntimeParserConfig struct {
	BaseURL      string
	ServiceToken string
	TokenHeader  string
	Timeout      time.Duration
	MaxReadBytes int64
}

type RuntimeParserClient struct {
	baseURL      *url.URL
	serviceToken string
	tokenHeader  string
	httpClient   *http.Client
	maxReadBytes int64
}

func NewRuntimeParserClient(cfg RuntimeParserConfig) (*RuntimeParserClient, error) {
	baseURL, err := parseBaseURL(cfg.BaseURL)
	if err != nil {
		return nil, err
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 60 * time.Second
	}
	if cfg.MaxReadBytes <= 0 {
		return nil, errors.New("max read bytes must be positive")
	}
	tokenHeader := strings.TrimSpace(cfg.TokenHeader)
	if tokenHeader == "" {
		tokenHeader = "X-Service-Token"
	}
	if strings.ContainsAny(tokenHeader, "\r\n:") {
		return nil, errors.New("runtime token header is invalid")
	}
	return &RuntimeParserClient{
		baseURL:      baseURL,
		serviceToken: strings.TrimSpace(cfg.ServiceToken),
		tokenHeader:  tokenHeader,
		httpClient:   &http.Client{Timeout: cfg.Timeout},
		maxReadBytes: cfg.MaxReadBytes,
	}, nil
}

func (c *RuntimeParserClient) Parse(ctx context.Context, filename, contentType string, data []byte) (service.ParsedAttachment, error) {
	if len(data) == 0 {
		return service.ParsedAttachment{}, errors.New("document is empty")
	}
	if int64(len(data)) > c.maxReadBytes {
		return service.ParsedAttachment{}, fmt.Errorf("document exceeds configured attachment limit of %d bytes", c.maxReadBytes)
	}
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	header := make(textproto.MIMEHeader)
	header.Set("Content-Disposition", mime.FormatMediaType("form-data", map[string]string{
		"name":     "file",
		"filename": filename,
	}))
	contentType = strings.TrimSpace(contentType)
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	header.Set("Content-Type", contentType)
	part, err := writer.CreatePart(header)
	if err != nil {
		return service.ParsedAttachment{}, err
	}
	if _, err := part.Write(data); err != nil {
		return service.ParsedAttachment{}, err
	}
	if err := writer.Close(); err != nil {
		return service.ParsedAttachment{}, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint("/api/v1/internal/attachments/parse"), &body)
	if err != nil {
		return service.ParsedAttachment{}, err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	applyInternalHeaders(req, "", "", service.RequestIDFromContext(ctx))
	if c.serviceToken != "" {
		req.Header.Set(c.tokenHeader, c.serviceToken)
	}

	res, err := c.httpClient.Do(req)
	if err != nil {
		return service.ParsedAttachment{}, err
	}
	defer res.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(res.Body, 16<<20))
	if err != nil {
		return service.ParsedAttachment{}, err
	}
	if res.StatusCode >= http.StatusBadRequest {
		return service.ParsedAttachment{}, fmt.Errorf("knowledge runtime parser returned %d", res.StatusCode)
	}
	var envelope struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    struct {
			PageCount int `json:"pageCount"`
			Chunks    []struct {
				PageNumber  int    `json:"pageNumber"`
				SectionPath string `json:"sectionPath"`
				Content     string `json:"content"`
			} `json:"chunks"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return service.ParsedAttachment{}, err
	}
	if envelope.Code != 0 {
		msg := strings.TrimSpace(envelope.Message)
		if msg == "" {
			msg = "knowledge runtime parser failed"
		}
		return service.ParsedAttachment{}, errors.New(msg)
	}
	chunks := make([]service.ParsedAttachmentChunk, 0, len(envelope.Data.Chunks))
	for _, chunk := range envelope.Data.Chunks {
		content := strings.TrimSpace(chunk.Content)
		if content == "" {
			continue
		}
		pageNumber := chunk.PageNumber
		if pageNumber <= 0 {
			pageNumber = 1
		}
		chunks = append(chunks, service.ParsedAttachmentChunk{
			PageNumber:  pageNumber,
			SectionPath: strings.TrimSpace(chunk.SectionPath),
			Content:     content,
		})
	}
	if len(chunks) == 0 {
		return service.ParsedAttachment{}, errors.New("knowledge runtime parser returned no content")
	}
	pageCount := envelope.Data.PageCount
	if pageCount <= 0 {
		pageCount = 1
	}
	return service.ParsedAttachment{PageCount: pageCount, Chunks: chunks}, nil
}

func (c *RuntimeParserClient) endpoint(suffix string) string {
	clone := *c.baseURL
	clone.Path = path.Join(c.baseURL.Path, suffix)
	return clone.String()
}
