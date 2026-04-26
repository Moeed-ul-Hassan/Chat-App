package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/clerk/clerk-sdk-go/v2"
	"github.com/clerk/clerk-sdk-go/v2/jwt"
)

type contextKey string

const UserKey contextKey = "user"

func AuthRequired(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		tokenStr := ""

		if strings.HasPrefix(authHeader, "Bearer ") {
			tokenStr = strings.TrimPrefix(authHeader, "Bearer ")
		}

		if tokenStr == "" {
			http.Error(w, `{"error":"unauthorized: missing token"}`, http.StatusUnauthorized)
			return
		}

		claims, err := jwt.Verify(r.Context(), &jwt.VerifyParams{
			Token: tokenStr,
		})
		if err != nil {
			http.Error(w, `{"error":"unauthorized: invalid token"}`, http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), UserKey, claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func GetUserClaims(r *http.Request) *clerk.SessionClaims {
	claims, _ := r.Context().Value(UserKey).(*clerk.SessionClaims)
	return claims
}

func RequireAdmin(next http.Handler) http.Handler {
	return AuthRequired(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims := GetUserClaims(r)
		if claims == nil || claims.Subject == "" {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}
		role := ""
		isAdmin := false
		if claims.Custom != nil {
			if customMap, ok := claims.Custom.(map[string]any); ok {
				if roleValue, ok := customMap["role"]; ok {
					role = strings.ToLower(fmt.Sprintf("%v", roleValue))
				}
				if adminValue, ok := customMap["is_admin"]; ok {
					adminRaw := strings.ToLower(fmt.Sprintf("%v", adminValue))
					isAdmin = adminRaw == "true" || adminRaw == "1" || adminRaw == "yes"
				}
			}
		}
		if role != "admin" && !isAdmin {
			http.Error(w, `{"error":"forbidden: admin role required"}`, http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	}))
}
