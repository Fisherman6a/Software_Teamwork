package attachmentclient

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/qa/internal/service"
)

func TestRuntimeParserClientParseMapsRuntimeChunks(t *testing.T) {
	var sawToken, sawRequestID bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/internal/attachments/parse" {
			t.Fatalf("request = %s %s", r.Method, r.URL.Path)
		}
		if r.Header.Get("X-Runtime-Token") == "runtime-token" {
			sawToken = true
		}
		if r.Header.Get("X-Request-Id") == "req-runtime-parser" {
			sawRequestID = true
		}
		if err := r.ParseMultipartForm(1 << 20); err != nil {
			t.Fatalf("ParseMultipartForm: %v", err)
		}
		file, header, err := r.FormFile("file")
		if err != nil {
			t.Fatalf("FormFile: %v", err)
		}
		defer file.Close()
		data, err := io.ReadAll(file)
		if err != nil {
			t.Fatalf("ReadAll: %v", err)
		}
		if header.Filename != "manual.pdf" || string(data) != "%PDF" {
			t.Fatalf("uploaded file = %q %q", header.Filename, data)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"code": 0,
			"data": map[string]any{
				"pageCount": 2,
				"chunks": []map[string]any{
					{"pageNumber": 1, "sectionPath": "intro", "content": " first chunk "},
					{"pageNumber": 2, "content": "second chunk"},
					{"pageNumber": 3, "content": "   "},
				},
			},
		})
	}))
	defer server.Close()

	client, err := NewRuntimeParserClient(RuntimeParserConfig{
		BaseURL:      server.URL,
		ServiceToken: "runtime-token",
		TokenHeader:  "X-Runtime-Token",
		Timeout:      time.Second,
		MaxReadBytes: 16,
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx := service.WithRequestID(context.Background(), "req-runtime-parser")
	parsed, err := client.Parse(ctx, "manual.pdf", "application/pdf", []byte("%PDF"))
	if err != nil {
		t.Fatal(err)
	}
	if !sawToken || !sawRequestID {
		t.Fatalf("headers sawToken=%v sawRequestID=%v", sawToken, sawRequestID)
	}
	if parsed.PageCount != 2 || len(parsed.Chunks) != 2 {
		t.Fatalf("parsed = %+v", parsed)
	}
	if parsed.Chunks[0].Content != "first chunk" || parsed.Chunks[0].SectionPath != "intro" || parsed.Chunks[1].PageNumber != 2 {
		t.Fatalf("chunks = %+v", parsed.Chunks)
	}
}

func TestRuntimeParserClientParseRejectsRuntimeErrors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte(`{"code":500,"message":"downstream secret should not matter"}`))
	}))
	defer server.Close()

	client, err := NewRuntimeParserClient(RuntimeParserConfig{BaseURL: server.URL, MaxReadBytes: 16})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := client.Parse(context.Background(), "manual.pdf", "application/pdf", []byte("%PDF")); err == nil ||
		!strings.Contains(err.Error(), "knowledge runtime parser returned 502") {
		t.Fatalf("Parse() error = %v", err)
	}
}

func TestNewRuntimeParserClientValidatesConfiguration(t *testing.T) {
	if _, err := NewRuntimeParserClient(RuntimeParserConfig{BaseURL: "http://runtime:9380", TokenHeader: "bad:header", MaxReadBytes: 1}); err == nil {
		t.Fatal("expected invalid token header to fail")
	}
	if _, err := NewRuntimeParserClient(RuntimeParserConfig{BaseURL: "http://runtime:9380"}); err == nil {
		t.Fatal("expected missing max read bytes to fail")
	}
}
