package middleware

import (
	"net/http"

	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/gateway/internal/response"
)

type InFlightLimiter struct {
	tokens chan struct{}
}

func NewInFlightLimiter(max int) *InFlightLimiter {
	if max <= 0 {
		return nil
	}
	return &InFlightLimiter{tokens: make(chan struct{}, max)}
}

func (l *InFlightLimiter) TryAcquire() (func(), bool) {
	if l == nil {
		return func() {}, true
	}
	select {
	case l.tokens <- struct{}{}:
		return func() { <-l.tokens }, true
	default:
		return nil, false
	}
}

func InFlight(max int) Middleware {
	return InFlightWithLimiter(NewInFlightLimiter(max))
}

func InFlightWithLimiter(limiter *InFlightLimiter) Middleware {
	if limiter == nil {
		return func(next http.Handler) http.Handler { return next }
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			release, ok := limiter.TryAcquire()
			if !ok {
				response.WriteError(w, http.StatusTooManyRequests, response.ErrorDetail{
					Code:      response.CodeRateLimited,
					Message:   "rate limited",
					RequestID: RequestIDFromContext(r.Context()),
				})
				return
			}
			defer release()
			next.ServeHTTP(w, r)
		})
	}
}
