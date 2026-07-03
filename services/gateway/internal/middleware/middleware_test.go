package middleware_test

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/gateway/internal/middleware"
	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/gateway/internal/response"
)

func TestRecoverConvertsPanicToErrorEnvelope(t *testing.T) {
	handler := middleware.Chain(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			panic("boom")
		}),
		middleware.RequestID(),
		middleware.Recover(slog.New(slog.NewTextHandler(io.Discard, nil))),
	)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Request-Id", "req_panic")
	res := httptest.NewRecorder()

	handler.ServeHTTP(res, req)

	if res.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, body = %s", res.Code, res.Body.String())
	}
	var body errorBody
	decodeJSON(t, res.Body, &body)
	if body.Error.Code != response.CodeInternal || body.Error.RequestID != "req_panic" {
		t.Fatalf("error body = %+v", body.Error)
	}
}

func TestTimeoutAddsRequestDeadline(t *testing.T) {
	handler := middleware.Chain(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, ok := r.Context().Deadline()
			response.WriteJSON(w, http.StatusOK, map[string]bool{"hasDeadline": ok}, middleware.RequestIDFromContext(r.Context()))
		}),
		middleware.RequestID(),
		middleware.Timeout(time.Second),
	)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	res := httptest.NewRecorder()

	handler.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", res.Code, res.Body.String())
	}
	var body deadlineBody
	decodeJSON(t, res.Body, &body)
	if !body.Data.HasDeadline {
		t.Fatal("request context has no deadline")
	}
}

func TestInFlightReturnsRateLimitedWhenSaturated(t *testing.T) {
	entered := make(chan struct{})
	releaseHandler := make(chan struct{})
	firstDone := make(chan struct{})
	var once sync.Once

	handler := middleware.Chain(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			once.Do(func() { close(entered) })
			<-releaseHandler
			response.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"}, middleware.RequestIDFromContext(r.Context()))
		}),
		middleware.RequestID(),
		middleware.InFlight(1),
	)

	firstReq := httptest.NewRequest(http.MethodGet, "/", nil)
	firstReq.Header.Set("X-Request-Id", "req_first")
	firstRes := httptest.NewRecorder()
	go func() {
		defer close(firstDone)
		handler.ServeHTTP(firstRes, firstReq)
	}()

	select {
	case <-entered:
	case <-time.After(time.Second):
		t.Fatal("first request did not enter handler")
	}

	secondReq := httptest.NewRequest(http.MethodGet, "/", nil)
	secondReq.Header.Set("X-Request-Id", "req_second")
	secondRes := httptest.NewRecorder()
	handler.ServeHTTP(secondRes, secondReq)

	if secondRes.Code != http.StatusTooManyRequests {
		t.Fatalf("status = %d, body = %s", secondRes.Code, secondRes.Body.String())
	}
	if got := secondRes.Header().Get("X-Request-Id"); got != "req_second" {
		t.Fatalf("X-Request-Id = %q", got)
	}
	var body errorBody
	decodeJSON(t, secondRes.Body, &body)
	if body.Error.Code != response.CodeRateLimited || body.Error.RequestID != "req_second" {
		t.Fatalf("error body = %+v", body.Error)
	}

	close(releaseHandler)
	select {
	case <-firstDone:
	case <-time.After(time.Second):
		t.Fatal("first request did not finish")
	}
	if firstRes.Code != http.StatusOK {
		t.Fatalf("first status = %d, body = %s", firstRes.Code, firstRes.Body.String())
	}
}

func decodeJSON(t *testing.T, reader io.Reader, target any) {
	t.Helper()
	if err := json.NewDecoder(reader).Decode(target); err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
}

type errorBody struct {
	Error struct {
		Code      response.Code `json:"code"`
		Message   string        `json:"message"`
		RequestID string        `json:"requestId"`
	} `json:"error"`
}

type deadlineBody struct {
	Data struct {
		HasDeadline bool `json:"hasDeadline"`
	} `json:"data"`
	RequestID string `json:"requestId"`
}
