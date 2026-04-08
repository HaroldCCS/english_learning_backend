package http

import (
	"context"
	"net/http"
	"strings"

	"github.com/haroldcamargo/english/backend/internal/config"
	"github.com/haroldcamargo/english/backend/internal/security"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ctxKey string

const userIDKey ctxKey = "userId"

type AuthMiddleware struct {
	cfg config.Config
}

func NewAuthMiddleware(cfg config.Config) AuthMiddleware {
	return AuthMiddleware{cfg: cfg}
}

func (m AuthMiddleware) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth == "" {
			writeError(w, http.StatusUnauthorized, "missing authorization")
			return
		}
		parts := strings.SplitN(auth, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			writeError(w, http.StatusUnauthorized, "invalid authorization")
			return
		}
		userID, err := security.ParseToken(m.cfg.JWTSecret, parts[1])
		if err != nil {
			writeError(w, http.StatusUnauthorized, "invalid token")
			return
		}
		ctx := context.WithValue(r.Context(), userIDKey, userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func UserIDFromContext(ctx context.Context) (primitive.ObjectID, bool) {
	v := ctx.Value(userIDKey)
	id, ok := v.(primitive.ObjectID)
	return id, ok
}
