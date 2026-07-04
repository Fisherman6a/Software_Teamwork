package httpapi

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/gateway/internal/middleware"
	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/gateway/internal/response"
	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/gateway/internal/service"
)

const (
	defaultAdminMetricsDays = 30
	maxAdminMetricsDays     = 90
)

type adminOverviewResponse struct {
	Totals    adminMetricTotals `json:"totals"`
	UpdatedAt time.Time         `json:"updatedAt"`
}

type adminMetricTotals struct {
	UserCount           int64 `json:"userCount"`
	KnowledgeBaseCount  int64 `json:"knowledgeBaseCount"`
	DocumentCount       int64 `json:"documentCount"`
	ChunkCount          int64 `json:"chunkCount"`
	ReportTemplateCount int64 `json:"reportTemplateCount"`
	ReportRecordCount   int64 `json:"reportRecordCount"`
	QACount             int64 `json:"qaCount"`
}

type adminMetricsResponse struct {
	Days        int               `json:"days"`
	Granularity string            `json:"granularity"`
	Series      adminMetricSeries `json:"series"`
	UpdatedAt   time.Time         `json:"updatedAt"`
}

type adminMetricSeries struct {
	UserCount           []adminMetricPoint `json:"userCount"`
	KnowledgeBaseCount  []adminMetricPoint `json:"knowledgeBaseCount"`
	DocumentCount       []adminMetricPoint `json:"documentCount"`
	ChunkCount          []adminMetricPoint `json:"chunkCount"`
	ReportTemplateCount []adminMetricPoint `json:"reportTemplateCount"`
	ReportRecordCount   []adminMetricPoint `json:"reportRecordCount"`
	QACount             []adminMetricPoint `json:"qaCount"`
}

type adminMetricPoint struct {
	Date  time.Time `json:"date"`
	Count int64     `json:"count"`
}

type ownerAdminStatsEnvelope struct {
	Data ownerAdminStatsData `json:"data"`
}

type ownerAdminStatsData struct {
	UserCount           *int64                 `json:"userCount"`
	KnowledgeBaseCount  *int64                 `json:"knowledgeBaseCount"`
	DocumentCount       *int64                 `json:"documentCount"`
	ChunkCount          *int64                 `json:"chunkCount"`
	ReportTemplateCount *int64                 `json:"reportTemplateCount"`
	ReportRecordCount   *int64                 `json:"reportRecordCount"`
	QACount             *int64                 `json:"qaCount"`
	Series              *ownerAdminStatsSeries `json:"series"`
}

type ownerAdminStatsSeries struct {
	UserCount           *[]ownerAdminStatsPoint `json:"userCount"`
	KnowledgeBaseCount  *[]ownerAdminStatsPoint `json:"knowledgeBaseCount"`
	DocumentCount       *[]ownerAdminStatsPoint `json:"documentCount"`
	ChunkCount          *[]ownerAdminStatsPoint `json:"chunkCount"`
	ReportTemplateCount *[]ownerAdminStatsPoint `json:"reportTemplateCount"`
	ReportRecordCount   *[]ownerAdminStatsPoint `json:"reportRecordCount"`
	QACount             *[]ownerAdminStatsPoint `json:"qaCount"`
}

type ownerAdminStatsPoint struct {
	Date  string `json:"date"`
	Count *int64 `json:"count"`
}

type adminStatsSnapshot struct {
	Totals adminMetricTotals
	Series adminMetricSeries
}

type adminStatsDependencyError struct {
	owner string
	err   error
}

func (e adminStatsDependencyError) Error() string {
	return e.owner + " admin statistics dependency failed"
}

func (e adminStatsDependencyError) Unwrap() error {
	return e.err
}

func (s *Server) handleAdminOverview(w http.ResponseWriter, r *http.Request) {
	authContext, _, ok := s.authenticateRequest(w, r)
	if !ok {
		return
	}
	if !hasAdminRouteAccess(authContext, dashboardAdminPermissions, nil, true) {
		response.WriteError(w, http.StatusForbidden, response.ErrorDetail{
			Code:      response.CodeForbidden,
			Message:   "forbidden",
			RequestID: middleware.RequestIDFromContext(r.Context()),
		})
		return
	}
	snapshot, err := s.collectAdminStats(r, authContext, defaultAdminMetricsDays, "daily", false)
	if err != nil {
		s.writeAdminStatsDependencyError(w, r, err)
		return
	}
	response.WriteJSON(w, http.StatusOK, adminOverviewResponse{
		Totals:    snapshot.Totals,
		UpdatedAt: time.Now().UTC(),
	}, middleware.RequestIDFromContext(r.Context()))
}

func (s *Server) handleAdminMetrics(w http.ResponseWriter, r *http.Request) {
	authContext, _, ok := s.authenticateRequest(w, r)
	if !ok {
		return
	}
	if !hasAdminRouteAccess(authContext, dashboardAdminPermissions, nil, true) {
		response.WriteError(w, http.StatusForbidden, response.ErrorDetail{
			Code:      response.CodeForbidden,
			Message:   "forbidden",
			RequestID: middleware.RequestIDFromContext(r.Context()),
		})
		return
	}
	days, granularity, err := parseAdminMetricsQuery(r)
	if err != nil {
		response.WriteError(w, http.StatusBadRequest, response.ErrorDetail{
			Code:      response.CodeValidation,
			Message:   "request validation failed",
			RequestID: middleware.RequestIDFromContext(r.Context()),
			Fields:    err,
		})
		return
	}
	snapshot, depErr := s.collectAdminStats(r, authContext, days, granularity, true)
	if depErr != nil {
		s.writeAdminStatsDependencyError(w, r, depErr)
		return
	}
	response.WriteJSON(w, http.StatusOK, adminMetricsResponse{
		Days:        days,
		Granularity: granularity,
		Series:      snapshot.Series,
		UpdatedAt:   time.Now().UTC(),
	}, middleware.RequestIDFromContext(r.Context()))
}

func parseAdminMetricsQuery(r *http.Request) (int, string, map[string]string) {
	fields := map[string]string{}
	days := defaultAdminMetricsDays
	if raw := strings.TrimSpace(r.URL.Query().Get("days")); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value < 1 || value > maxAdminMetricsDays {
			fields["days"] = "must be between 1 and 90"
		} else {
			days = value
		}
	}
	granularity := strings.TrimSpace(r.URL.Query().Get("granularity"))
	if granularity == "" {
		granularity = "daily"
	}
	if granularity != "daily" && granularity != "hourly" {
		fields["granularity"] = "must be daily or hourly"
	}
	if len(fields) > 0 {
		return 0, "", fields
	}
	return days, granularity, nil
}

func (s *Server) collectAdminStats(r *http.Request, authContext service.SessionCacheEntry, days int, granularity string, requireSeries bool) (adminStatsSnapshot, error) {
	snapshot := adminStatsSnapshot{Series: emptyAdminMetricSeries()}
	if err := s.mergeAuthAdminStats(r, authContext, days, granularity, requireSeries, &snapshot); err != nil {
		return adminStatsSnapshot{}, adminStatsDependencyError{owner: "auth", err: err}
	}
	if err := s.mergeKnowledgeAdminStats(r, authContext, days, granularity, requireSeries, &snapshot); err != nil {
		return adminStatsSnapshot{}, adminStatsDependencyError{owner: "knowledge", err: err}
	}
	if err := s.mergeDocumentAdminStats(r, authContext, days, granularity, requireSeries, &snapshot); err != nil {
		return adminStatsSnapshot{}, adminStatsDependencyError{owner: "document", err: err}
	}
	if err := s.mergeQAAdminStats(r, authContext, days, granularity, requireSeries, &snapshot); err != nil {
		return adminStatsSnapshot{}, adminStatsDependencyError{owner: "qa", err: err}
	}
	return snapshot, nil
}

func (s *Server) mergeAuthAdminStats(r *http.Request, authContext service.SessionCacheEntry, days int, granularity string, requireSeries bool, snapshot *adminStatsSnapshot) error {
	data, err := s.fetchOwnerAdminStats(r, authContext, "auth", "/internal/v1/admin/statistics", s.authAdminServiceToken, days, granularity, requireSeries)
	if err != nil {
		return err
	}
	if snapshot.Totals.UserCount, err = requiredAdminCount(data.UserCount, "userCount"); err != nil {
		return err
	}
	if requireSeries {
		if err := requireOwnerSeries(data.Series); err != nil {
			return err
		}
		snapshot.Series.UserCount, err = requiredAdminSeries(data.Series.UserCount, "userCount")
	}
	return err
}

func (s *Server) mergeKnowledgeAdminStats(r *http.Request, authContext service.SessionCacheEntry, days int, granularity string, requireSeries bool, snapshot *adminStatsSnapshot) error {
	data, err := s.fetchOwnerAdminStats(r, authContext, "knowledge", "/internal/v1/knowledge-statistics", s.internalServiceToken, days, granularity, requireSeries)
	if err != nil {
		return err
	}
	if snapshot.Totals.KnowledgeBaseCount, err = requiredAdminCount(data.KnowledgeBaseCount, "knowledgeBaseCount"); err != nil {
		return err
	}
	if snapshot.Totals.DocumentCount, err = requiredAdminCount(data.DocumentCount, "documentCount"); err != nil {
		return err
	}
	if snapshot.Totals.ChunkCount, err = requiredAdminCount(data.ChunkCount, "chunkCount"); err != nil {
		return err
	}
	if requireSeries {
		if err := requireOwnerSeries(data.Series); err != nil {
			return err
		}
		if snapshot.Series.KnowledgeBaseCount, err = requiredAdminSeries(data.Series.KnowledgeBaseCount, "knowledgeBaseCount"); err != nil {
			return err
		}
		if snapshot.Series.DocumentCount, err = requiredAdminSeries(data.Series.DocumentCount, "documentCount"); err != nil {
			return err
		}
		snapshot.Series.ChunkCount, err = requiredAdminSeries(data.Series.ChunkCount, "chunkCount")
	}
	return err
}

func (s *Server) mergeDocumentAdminStats(r *http.Request, authContext service.SessionCacheEntry, days int, granularity string, requireSeries bool, snapshot *adminStatsSnapshot) error {
	data, err := s.fetchOwnerAdminStats(r, authContext, "document", "/admin/statistics", s.internalServiceToken, days, granularity, requireSeries)
	if err != nil {
		return err
	}
	if snapshot.Totals.ReportTemplateCount, err = requiredAdminCount(data.ReportTemplateCount, "reportTemplateCount"); err != nil {
		return err
	}
	if snapshot.Totals.ReportRecordCount, err = requiredAdminCount(data.ReportRecordCount, "reportRecordCount"); err != nil {
		return err
	}
	if requireSeries {
		if err := requireOwnerSeries(data.Series); err != nil {
			return err
		}
		if snapshot.Series.ReportTemplateCount, err = requiredAdminSeries(data.Series.ReportTemplateCount, "reportTemplateCount"); err != nil {
			return err
		}
		snapshot.Series.ReportRecordCount, err = requiredAdminSeries(data.Series.ReportRecordCount, "reportRecordCount")
	}
	return err
}

func (s *Server) mergeQAAdminStats(r *http.Request, authContext service.SessionCacheEntry, days int, granularity string, requireSeries bool, snapshot *adminStatsSnapshot) error {
	data, err := s.fetchOwnerAdminStats(r, authContext, "qa", "/internal/v1/admin/statistics", s.internalServiceToken, days, granularity, requireSeries)
	if err != nil {
		return err
	}
	if snapshot.Totals.QACount, err = requiredAdminCount(data.QACount, "qaCount"); err != nil {
		return err
	}
	if requireSeries {
		if err := requireOwnerSeries(data.Series); err != nil {
			return err
		}
		snapshot.Series.QACount, err = requiredAdminSeries(data.Series.QACount, "qaCount")
	}
	return err
}

func (s *Server) fetchOwnerAdminStats(r *http.Request, authContext service.SessionCacheEntry, owner string, path string, serviceToken string, days int, granularity string, includeQuery bool) (ownerAdminStatsData, error) {
	baseURL := s.ownerBaseURLs[owner]
	if baseURL == nil {
		return ownerAdminStatsData{}, fmt.Errorf("owner base URL is not configured")
	}
	targetURL := *baseURL
	targetURL.Path = joinProxyPath(baseURL.Path, path)
	if includeQuery {
		query := targetURL.Query()
		query.Set("days", strconv.Itoa(days))
		query.Set("granularity", granularity)
		targetURL.RawQuery = query.Encode()
	}
	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, targetURL.String(), nil)
	if err != nil {
		return ownerAdminStatsData{}, fmt.Errorf("owner request could not be created")
	}
	req.Header = cloneProxyHeaders(r.Header)
	req.Header.Set("Accept", "application/json")
	applyGatewayHeaders(req, r, authContext, serviceToken)

	res, err := s.httpClient.Do(req)
	if err != nil {
		return ownerAdminStatsData{}, fmt.Errorf("owner request failed")
	}
	defer res.Body.Close()
	if res.StatusCode < http.StatusOK || res.StatusCode >= http.StatusMultipleChoices {
		io.Copy(io.Discard, res.Body)
		return ownerAdminStatsData{}, fmt.Errorf("owner returned status %d", res.StatusCode)
	}
	var envelope ownerAdminStatsEnvelope
	if err := json.NewDecoder(io.LimitReader(res.Body, 1<<20)).Decode(&envelope); err != nil {
		return ownerAdminStatsData{}, fmt.Errorf("owner returned invalid JSON")
	}
	return envelope.Data, nil
}

func requiredAdminCount(value *int64, field string) (int64, error) {
	if value == nil {
		return 0, fmt.Errorf("%s is missing", field)
	}
	if *value < 0 {
		return 0, fmt.Errorf("%s is negative", field)
	}
	return *value, nil
}

func requireOwnerSeries(series *ownerAdminStatsSeries) error {
	if series == nil {
		return fmt.Errorf("series is missing")
	}
	return nil
}

func requiredAdminSeries(points *[]ownerAdminStatsPoint, field string) ([]adminMetricPoint, error) {
	if points == nil {
		return nil, fmt.Errorf("%s series is missing", field)
	}
	out := make([]adminMetricPoint, 0, len(*points))
	for _, point := range *points {
		if point.Count == nil {
			return nil, fmt.Errorf("%s series count is missing", field)
		}
		if *point.Count < 0 {
			return nil, fmt.Errorf("%s series count is negative", field)
		}
		date, err := parseAdminMetricDate(point.Date)
		if err != nil {
			return nil, fmt.Errorf("%s series date is invalid", field)
		}
		out = append(out, adminMetricPoint{Date: date, Count: *point.Count})
	}
	return out, nil
}

func parseAdminMetricDate(raw string) (time.Time, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Time{}, fmt.Errorf("date is required")
	}
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339, "2006-01-02"} {
		parsed, err := time.Parse(layout, raw)
		if err == nil {
			return parsed.UTC(), nil
		}
	}
	return time.Time{}, fmt.Errorf("date is invalid")
}

func emptyAdminMetricSeries() adminMetricSeries {
	return adminMetricSeries{
		UserCount:           []adminMetricPoint{},
		KnowledgeBaseCount:  []adminMetricPoint{},
		DocumentCount:       []adminMetricPoint{},
		ChunkCount:          []adminMetricPoint{},
		ReportTemplateCount: []adminMetricPoint{},
		ReportRecordCount:   []adminMetricPoint{},
		QACount:             []adminMetricPoint{},
	}
}

func (s *Server) writeAdminStatsDependencyError(w http.ResponseWriter, r *http.Request, err error) {
	if depErr, ok := err.(adminStatsDependencyError); ok {
		s.logger.WarnContext(r.Context(), "admin statistics dependency failed",
			"service", "gateway",
			"request_id", middleware.RequestIDFromContext(r.Context()),
			"operation", "admin_statistics",
			"dependency", depErr.owner,
			"status", "failed",
		)
	}
	s.writeDependencyError(w, r, "admin statistics dependency is unavailable")
}
