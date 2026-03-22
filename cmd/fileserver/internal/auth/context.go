package auth

import (
	"context"
	"errors"
	"net/http"

	"github.com/ricochhet/fileserver/cmd/fileserver/internal/configutil"
	"github.com/ricochhet/fileserver/cmd/fileserver/internal/db"
	"github.com/ricochhet/fileserver/pkg/logutil"
)

type ctxKey string

const (
	ctxKeyUsername    ctxKey = "fs_username"
	ctxKeyDisplayName ctxKey = "fs_displayName"
	ctxKeyIsAdmin     ctxKey = "fs_isAdmin"
)

// UsernameFromCtx returns the authenticated username from the request context.
func UsernameFromCtx(r *http.Request) string {
	v, _ := r.Context().Value(ctxKeyUsername).(string)
	return v
}

// DisplayNameFromCtx returns the display name from the request context.
func DisplayNameFromCtx(r *http.Request) string {
	v, _ := r.Context().Value(ctxKeyDisplayName).(string)
	return v
}

// IsAdminFromCtx reports whether the authenticated user has the admin flag.
func IsAdminFromCtx(r *http.Request) bool {
	v, _ := r.Context().Value(ctxKeyIsAdmin).(bool)
	return v
}

// WithIdentity injects username, displayName, and isAdmin into the context.
func WithIdentity(ctx context.Context, username, displayName string, isAdmin bool) context.Context {
	ctx = context.WithValue(ctx, ctxKeyUsername, username)
	ctx = context.WithValue(ctx, ctxKeyDisplayName, displayName)
	ctx = context.WithValue(ctx, ctxKeyIsAdmin, isAdmin)

	return ctx
}

// ResolveDisplayName returns the best display name for username: DB > config > auto-generated.
func ResolveDisplayName(
	ctx context.Context,
	users []configutil.FormAuthUser,
	username string,
	database *db.DB,
) string {
	if database != nil {
		u, err := database.GetUser(ctx, username)
		if err != nil && !errors.Is(err, db.ErrUserNotFound) {
			logutil.Errorf(logutil.Get(), "ResolveDisplayName: db.GetUser %q: %v\n", username, err)
		}

		if u != nil && u.DisplayName != "" {
			return u.DisplayName
		}
	}

	for _, u := range users {
		if u.Username == username && u.DisplayName != "" {
			return u.DisplayName
		}
	}

	return GenerateDisplayName(username)
}

// ResolveIsAdmin reports whether username has the admin flag set. The DB takes
// precedence over the config list so runtime changes are respected immediately.
func ResolveIsAdmin(
	ctx context.Context,
	users []configutil.FormAuthUser,
	username string,
	database *db.DB,
) bool {
	if database != nil {
		u, err := database.GetUser(ctx, username)
		if err == nil && u != nil {
			return u.IsAdmin
		}
	}

	for _, u := range users {
		if u.Username == username {
			return u.Admin
		}
	}

	return false
}
