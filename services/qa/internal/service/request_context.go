package service

import (
	"context"
	"strings"

	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/qa/internal/platform/contextutil"
)

type userRolesContextKey struct{}
type userPermissionsContextKey struct{}

func WithRequestID(ctx context.Context, requestID string) context.Context {
	return contextutil.WithRequestID(ctx, requestID)
}

func RequestIDFromContext(ctx context.Context) string {
	return contextutil.RequestIDFromContext(ctx)
}

func WithUserID(ctx context.Context, userID string) context.Context {
	return contextutil.WithUserID(ctx, userID)
}

func UserIDFromContext(ctx context.Context) string {
	return contextutil.UserIDFromContext(ctx)
}

func WithUserRoles(ctx context.Context, roles string) context.Context {
	return context.WithValue(ctx, userRolesContextKey{}, strings.TrimSpace(roles))
}

func UserRolesFromContext(ctx context.Context) string {
	value, _ := ctx.Value(userRolesContextKey{}).(string)
	return value
}

func WithUserPermissions(ctx context.Context, permissions string) context.Context {
	return context.WithValue(ctx, userPermissionsContextKey{}, strings.TrimSpace(permissions))
}

func UserPermissionsFromContext(ctx context.Context) string {
	value, _ := ctx.Value(userPermissionsContextKey{}).(string)
	return value
}
