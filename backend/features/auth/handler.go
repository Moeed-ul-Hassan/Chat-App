// Package auth provides HTTP handlers for registration, login, and session management.
package auth

import (
	"encoding/json"
	"net/http"
	"regexp"
	"strings"

	"github.com/Moeed-ul-Hassan/chatapp/core/middleware"
	"github.com/Moeed-ul-Hassan/chatapp/features/user"
	"github.com/go-chi/chi/v5"
)

const (
	OTP_PURPOSE_VERIFY = "verify"
	OTP_PURPOSE_RESET  = "reset"
)

// AuthHandler coordinates authentication logic between the user and auth repositories.
// It depends on interfaces (AuthStore/UserStore) rather than concrete implementations
// so that it matches the 'Go Love' professional standard for testability.
type AuthHandler struct {
	userRepo user.UserStore
}

// NewAuthHandler creates a new instance of AuthHandler with its required dependency.
func NewAuthHandler(userRepository user.UserStore) *AuthHandler {
	return &AuthHandler{
		userRepo: userRepository,
	}
}

// respond writes a JSON successful response.
func (h *AuthHandler) respond(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// respondError writes a JSON error response with a descriptive message.
func (h *AuthHandler) respondError(w http.ResponseWriter, status int, msg string) {
	h.respond(w, status, map[string]string{"error": msg})
}

// RegisterRoutes sets up the routing table for Clerk-based authentication.
func RegisterRoutes(r chi.Router) {
	userRepo := user.NewUserRepository()
	handler := NewAuthHandler(userRepo)

	// Clerk Integration
	r.Group(func(protected chi.Router) {
		protected.Use(middleware.AuthRequired)
		protected.Post("/sync", handler.SyncUser)
	})
}

// SyncUser ensures that a Clerk user has a corresponding record in our MongoDB.
// This is called by the frontend immediately after a successful Clerk sign-in.
func (h *AuthHandler) SyncUser(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserClaims(r)
	if claims == nil {
		h.respondError(w, http.StatusUnauthorized, "No valid Clerk session found")
		return
	}

	// Extract details from Clerk claims/custom payload.
	// In newer Clerk SDK versions, custom fields are held in claims.Custom.
	var email string
	var username string
	if rawMap, ok := claims.Custom.(map[string]any); ok {
		if e, ok := rawMap["email"].(string); ok {
			email = e
		}
		if u, ok := rawMap["username"].(string); ok {
			username = u
		}
	}
	if username == "" {
		// sub usually looks like user_xxx, turn it into a stable default handle.
		re := regexp.MustCompile(`[^a-zA-Z0-9_.]`)
		username = strings.ToLower(re.ReplaceAllString(claims.Subject, ""))
	}
	if username == "" {
		username = "user"
	}
	if email == "" {
		// Keep email unique/non-empty even when Clerk custom claims omit it.
		email = claims.Subject + "@echo.local"
	}

	// Double check if user already exists in our DB
	existingUser, err := h.userRepo.GetUserByClerkID(r.Context(), claims.Subject)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, "Database lookup failure")
		return
	}

	if existingUser != nil {
		// Backfill identity fields for legacy users missing username/email.
		updatedUser, updateErr := h.userRepo.UpdateUserIdentity(r.Context(), existingUser.ID, username, email)
		if updateErr == nil && updatedUser != nil {
			existingUser = updatedUser
		}
		h.userRepo.UpdateLastSeen(r.Context(), existingUser.ID)
		h.respond(w, http.StatusOK, existingUser)
		return
	}

	newUser, err := h.userRepo.CreateUser(r.Context(), claims.Subject, username, email)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, "Failed to create local user profile")
		return
	}

	h.respond(w, http.StatusCreated, newUser)
}
