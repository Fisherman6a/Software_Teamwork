package httpapi

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestAppVersionFreshnessReturnsCurrentThroughGateway(t *testing.T) {
	const latestSHA = "abcdef1234567890"
	var capturedAuthorization string
	github := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedAuthorization = r.Header.Get("Authorization")
		if r.URL.Path != "/" {
			t.Fatalf("github path = %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"sha":"` + latestSHA + `","html_url":"https://github.test/commit/` + latestSHA + `"}`))
	}))
	defer github.Close()

	server := newAppVersionTestServer(t, github.URL, "backend-github-token")
	req := httptest.NewRequest(http.MethodGet, "/api/v1/app-version/freshness?currentSha="+latestSHA, nil)
	req.Header.Set("X-Request-Id", "req_app_version")
	res := httptest.NewRecorder()

	server.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", res.Code, res.Body.String())
	}
	if capturedAuthorization != "Bearer backend-github-token" {
		t.Fatalf("GitHub Authorization header = %q", capturedAuthorization)
	}
	if strings.Contains(res.Body.String(), "backend-github-token") {
		t.Fatalf("backend token leaked into response: %s", res.Body.String())
	}
	var body appVersionEnvelope
	decodeAppVersionJSON(t, res.Body, &body)
	if body.RequestID != "req_app_version" {
		t.Fatalf("requestId = %q", body.RequestID)
	}
	if body.Data.Status != appFreshnessCurrent ||
		body.Data.CurrentSHA != latestSHA ||
		body.Data.LatestSHA != latestSHA ||
		body.Data.LatestURL == "" {
		t.Fatalf("freshness = %+v", body.Data)
	}
}

func TestAppVersionFreshnessFallsBackToUnknownOnGitHubForbidden(t *testing.T) {
	github := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"message":"API rate limit exceeded"}`))
	}))
	defer github.Close()

	server := newAppVersionTestServer(t, github.URL, "")
	req := httptest.NewRequest(http.MethodGet, "/api/v1/app-version/freshness?currentSha=abcdef", nil)
	req.Header.Set("X-Request-Id", "req_app_version_403")
	res := httptest.NewRecorder()

	server.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", res.Code, res.Body.String())
	}
	var body appVersionEnvelope
	decodeAppVersionJSON(t, res.Body, &body)
	if body.Data.Status != appFreshnessUnknown ||
		body.Data.Reason != appFreshnessReasonGitHub403 ||
		body.Data.CurrentSHA != "abcdef" ||
		body.Data.LatestSHA != "" {
		t.Fatalf("freshness = %+v", body.Data)
	}
}

func TestAppVersionFreshnessCachesLatestCommit(t *testing.T) {
	const latestSHA = "abcdef1234567890"
	calls := 0
	github := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"sha":"` + latestSHA + `","html_url":"https://github.test/commit/` + latestSHA + `"}`))
	}))
	defer github.Close()

	server := newAppVersionTestServer(t, github.URL, "")
	for _, currentSHA := range []string{latestSHA, "1111111111111111"} {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/app-version/freshness?currentSha="+currentSHA, nil)
		res := httptest.NewRecorder()
		server.ServeHTTP(res, req)
		if res.Code != http.StatusOK {
			t.Fatalf("status = %d, body = %s", res.Code, res.Body.String())
		}
	}
	if calls != 1 {
		t.Fatalf("GitHub calls = %d, want 1", calls)
	}
}

func TestAppVersionFreshnessRejectsLongCurrentSHA(t *testing.T) {
	server := newAppVersionTestServer(t, "https://github.test/not-called", "")
	req := httptest.NewRequest(http.MethodGet, "/api/v1/app-version/freshness?currentSha="+strings.Repeat("a", 129), nil)
	req.Header.Set("X-Request-Id", "req_app_version_validation")
	res := httptest.NewRecorder()

	server.ServeHTTP(res, req)

	if res.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, body = %s", res.Code, res.Body.String())
	}
	var body struct {
		Error struct {
			Code      string `json:"code"`
			RequestID string `json:"requestId"`
		} `json:"error"`
	}
	decodeAppVersionJSON(t, res.Body, &body)
	if body.Error.Code != "validation_error" || body.Error.RequestID != "req_app_version_validation" {
		t.Fatalf("error = %+v", body.Error)
	}
}

func newAppVersionTestServer(t *testing.T, githubURL string, githubToken string) http.Handler {
	t.Helper()
	checker := newGitHubAppVersionChecker(http.DefaultClient, slog.New(slog.NewTextHandler(io.Discard, nil)), githubToken)
	checker.apiURL = githubURL
	checker.now = func() time.Time {
		return time.Date(2026, 7, 3, 12, 0, 0, 0, time.UTC)
	}
	return NewServer(Config{
		Logger:             slog.New(slog.NewTextHandler(io.Discard, nil)),
		ServiceVersion:     "test",
		Environment:        "test",
		RequestTimeout:     time.Second,
		MaxBodyBytes:       1024,
		CORSAllowedOrigins: []string{"*"},
		AppVersionChecker:  checker,
	})
}

type appVersionEnvelope struct {
	Data      AppVersionFreshness `json:"data"`
	RequestID string              `json:"requestId"`
}

func decodeAppVersionJSON(t *testing.T, r io.Reader, target any) {
	t.Helper()
	if err := json.NewDecoder(r).Decode(target); err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
}
