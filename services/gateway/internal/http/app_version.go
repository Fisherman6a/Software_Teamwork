package httpapi

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/gateway/internal/middleware"
	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/gateway/internal/response"
)

const (
	githubDevelopCommitAPI = "https://api.github.com/repos/Sakayori-Iroha-168/Software_Teamwork/commits/develop"
	appVersionCacheTTL     = 5 * time.Minute

	appFreshnessCurrent   = "current"
	appFreshnessDifferent = "different"
	appFreshnessUnknown   = "unknown"

	appFreshnessReasonMissingCurrentSHA = "missing_current_sha"
	appFreshnessReasonGitHub403         = "github_403"
	appFreshnessReasonGitHub404         = "github_404"
	appFreshnessReasonGitHub429         = "github_429"
	appFreshnessReasonGitHubStatus      = "github_status"
	appFreshnessReasonNetworkError      = "network_error"
	appFreshnessReasonInvalidResponse   = "invalid_response"
)

type AppVersionChecker interface {
	CheckFreshness(ctx context.Context, currentSHA string) AppVersionFreshness
}

type AppVersionFreshness struct {
	Status     string    `json:"status"`
	CurrentSHA string    `json:"currentSha,omitempty"`
	LatestSHA  string    `json:"latestSha,omitempty"`
	LatestURL  string    `json:"latestUrl,omitempty"`
	CheckedAt  time.Time `json:"checkedAt"`
	Reason     string    `json:"reason,omitempty"`
}

type gitHubAppVersionChecker struct {
	apiURL    string
	token     string
	client    *http.Client
	logger    *slog.Logger
	cacheTTL  time.Duration
	now       func() time.Time
	cacheLock sync.Mutex
	cache     gitHubLatestCommitCache
}

type gitHubLatestCommitCache struct {
	latest    gitHubLatestCommit
	expiresAt time.Time
}

type gitHubLatestCommit struct {
	sha       string
	url       string
	checkedAt time.Time
	reason    string
}

type gitHubCommitResponse struct {
	SHA     string `json:"sha"`
	HTMLURL string `json:"html_url"`
}

func newGitHubAppVersionChecker(client *http.Client, logger *slog.Logger, token string) *gitHubAppVersionChecker {
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &gitHubAppVersionChecker{
		apiURL:   githubDevelopCommitAPI,
		token:    strings.TrimSpace(token),
		client:   client,
		logger:   logger,
		cacheTTL: appVersionCacheTTL,
		now:      time.Now,
	}
}

func (s *Server) handleAppVersionFreshness(w http.ResponseWriter, r *http.Request) {
	currentSHA := strings.TrimSpace(r.URL.Query().Get("currentSha"))
	if len(currentSHA) > 128 {
		response.WriteError(w, http.StatusBadRequest, response.ErrorDetail{
			Code:      response.CodeValidation,
			Message:   "currentSha is too long",
			RequestID: middleware.RequestIDFromContext(r.Context()),
		})
		return
	}

	checker := s.appVersionChecker
	if checker == nil {
		checker = newGitHubAppVersionChecker(s.httpClient, s.logger, "")
	}
	freshness := checker.CheckFreshness(r.Context(), currentSHA)
	response.WriteJSON(w, http.StatusOK, freshness, middleware.RequestIDFromContext(r.Context()))
}

func (c *gitHubAppVersionChecker) CheckFreshness(ctx context.Context, currentSHA string) AppVersionFreshness {
	currentSHA = normalizeCommitSHA(currentSHA)
	checkedAt := c.now().UTC()
	if currentSHA == "" {
		return unknownAppVersionFreshness(currentSHA, checkedAt, appFreshnessReasonMissingCurrentSHA)
	}

	latest := c.latestDevelopCommit(ctx)
	if latest.sha == "" {
		return unknownAppVersionFreshness(currentSHA, latest.checkedAt, latest.reason)
	}

	status := appFreshnessDifferent
	if currentSHA == latest.sha {
		status = appFreshnessCurrent
	}
	return AppVersionFreshness{
		Status:     status,
		CurrentSHA: currentSHA,
		LatestSHA:  latest.sha,
		LatestURL:  latest.url,
		CheckedAt:  latest.checkedAt,
	}
}

func (c *gitHubAppVersionChecker) latestDevelopCommit(ctx context.Context) gitHubLatestCommit {
	now := c.now().UTC()

	c.cacheLock.Lock()
	if !c.cache.expiresAt.IsZero() && now.Before(c.cache.expiresAt) {
		latest := c.cache.latest
		c.cacheLock.Unlock()
		return latest
	}
	c.cacheLock.Unlock()

	latest := c.fetchLatestDevelopCommit(ctx, now)

	c.cacheLock.Lock()
	c.cache.latest = latest
	c.cache.expiresAt = now.Add(c.cacheTTL)
	c.cacheLock.Unlock()

	return latest
}

func (c *gitHubAppVersionChecker) fetchLatestDevelopCommit(ctx context.Context, checkedAt time.Time) gitHubLatestCommit {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.apiURL, nil)
	if err != nil {
		c.warnGitHubFallback(ctx, checkedAt, appFreshnessReasonInvalidResponse, 0)
		return gitHubLatestCommit{checkedAt: checkedAt, reason: appFreshnessReasonInvalidResponse}
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "Software-Teamwork-Gateway")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	res, err := c.client.Do(req)
	if err != nil {
		c.warnGitHubFallback(ctx, checkedAt, appFreshnessReasonNetworkError, 0)
		return gitHubLatestCommit{checkedAt: checkedAt, reason: appFreshnessReasonNetworkError}
	}
	defer res.Body.Close()

	if res.StatusCode < http.StatusOK || res.StatusCode >= http.StatusMultipleChoices {
		_, _ = io.Copy(io.Discard, io.LimitReader(res.Body, 1<<20))
		reason := gitHubStatusReason(res.StatusCode)
		c.warnGitHubFallback(ctx, checkedAt, reason, res.StatusCode)
		return gitHubLatestCommit{checkedAt: checkedAt, reason: reason}
	}

	var body gitHubCommitResponse
	if err := json.NewDecoder(io.LimitReader(res.Body, 1<<20)).Decode(&body); err != nil {
		c.warnGitHubFallback(ctx, checkedAt, appFreshnessReasonInvalidResponse, 0)
		return gitHubLatestCommit{checkedAt: checkedAt, reason: appFreshnessReasonInvalidResponse}
	}

	latestSHA := normalizeCommitSHA(body.SHA)
	if latestSHA == "" {
		c.warnGitHubFallback(ctx, checkedAt, appFreshnessReasonInvalidResponse, 0)
		return gitHubLatestCommit{checkedAt: checkedAt, reason: appFreshnessReasonInvalidResponse}
	}
	return gitHubLatestCommit{
		sha:       latestSHA,
		url:       strings.TrimSpace(body.HTMLURL),
		checkedAt: checkedAt,
	}
}

func (c *gitHubAppVersionChecker) warnGitHubFallback(ctx context.Context, checkedAt time.Time, reason string, statusCode int) {
	args := []any{
		"service", "gateway",
		"operation", "app_version_freshness",
		"dependency", "github",
		"status", "unknown",
		"reason", reason,
		"checked_at", checkedAt.Format(time.RFC3339),
	}
	if statusCode > 0 {
		args = append(args, "status_code", statusCode)
	}
	c.logger.WarnContext(ctx, "github app version freshness check fell back to unknown", args...)
}

func unknownAppVersionFreshness(currentSHA string, checkedAt time.Time, reason string) AppVersionFreshness {
	return AppVersionFreshness{
		Status:     appFreshnessUnknown,
		CurrentSHA: normalizeCommitSHA(currentSHA),
		CheckedAt:  checkedAt,
		Reason:     reason,
	}
}

func normalizeCommitSHA(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func gitHubStatusReason(statusCode int) string {
	switch statusCode {
	case http.StatusForbidden:
		return appFreshnessReasonGitHub403
	case http.StatusNotFound:
		return appFreshnessReasonGitHub404
	case http.StatusTooManyRequests:
		return appFreshnessReasonGitHub429
	default:
		return appFreshnessReasonGitHubStatus
	}
}
