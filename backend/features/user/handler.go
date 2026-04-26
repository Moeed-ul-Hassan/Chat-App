// Package user provides the HTTP handlers for user-related API endpoints.
package user

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/Moeed-ul-Hassan/chatapp/core/middleware"
	coreRedis "github.com/Moeed-ul-Hassan/chatapp/core/redis"
	"github.com/go-chi/chi/v5"
)

// UserHandler contains the dependencies required to process user-related HTTP requests.
// By wrapping our handlers in a struct that depends on an interface (UserStore),
// we can easily swap the real database for a mock during unit testing.
type UserHandler struct {
	repo UserStore
}

const (
	userMeCacheTTL     = 45 * time.Second
	userSearchCacheTTL = 2 * time.Minute
)

// NewUserHandler creates a new instance of UserHandler with the provided repository.
func NewUserHandler(repository UserStore) *UserHandler {
	return &UserHandler{
		repo: repository,
	}
}

// respond sends a JSON success response to the client.
func (h *UserHandler) respond(w http.ResponseWriter, statusCode int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(payload)
}

// respondError sends a JSON error response with a human-readable message.
func (h *UserHandler) respondError(w http.ResponseWriter, statusCode int, errorMessage string) {
	h.respond(w, statusCode, map[string]string{"error": errorMessage})
}

// RegisterRoutes registers all user-related API routes on the provided chi.Router.
// All routes here require the AuthRequired middleware (a valid PASETO token must be present).
func RegisterRoutes(r chi.Router) {
	// Initialize the handler with its repository dependency
	userRepo := NewUserRepository()
	handler := NewUserHandler(userRepo)

	r.Group(func(protectedRouter chi.Router) {
		protectedRouter.Use(middleware.AuthRequired)

		// Get the currently authenticated user's profile (used on app load)
		// GET /api/user/me
		protectedRouter.Get("/me", handler.GetCurrentUserHandler)

		// Search for another user by their unique username handle (used for DM discovery)
		// GET /api/user/search?username=john
		protectedRouter.Get("/search", handler.SearchUserByUsernameHandler)
	})
}

// GetCurrentUserHandler returns the full profile of the currently authenticated user.
// @Summary Get 'Me' Profile
// @Description Returns the full profile of the authenticated user based on their PASETO token.
// @Tags User
// @Produce json
// @Security ApiKeyAuth
// @Success 200 {object} User
// @Router /api/user/me [get]
func (h *UserHandler) GetCurrentUserHandler(w http.ResponseWriter, r *http.Request) {
	// Extract user identity from the validated PASETO claims stored in the request context
	authClaims := middleware.GetUserClaims(r)
	if authClaims == nil {
		h.respondError(w, http.StatusUnauthorized, "You must be logged in to access this profile")
		return
	}

	currentUser, err := h.repo.GetUserByClerkID(r.Context(), authClaims.Subject)
	if err != nil || currentUser == nil {
		h.respondError(w, http.StatusNotFound, "User account not found in our records")
		return
	}

	cacheKey := fmt.Sprintf("cache:user:me:%s", currentUser.ID)
	var foundUser *User
	if hit, err := coreRedis.CacheGetJSON(r.Context(), cacheKey, &foundUser); err == nil && hit && foundUser != nil {
		h.respond(w, http.StatusOK, foundUser)
		return
	}

	// Use already resolved current user.
	foundUser = currentUser

	_ = coreRedis.CacheSetJSON(r.Context(), cacheKey, foundUser, userMeCacheTTL)
	h.respond(w, http.StatusOK, foundUser)
}

// SearchUserByUsernameHandler finds a user by their unique username handle.
// @Summary Search User by Username
// @Description Finds a public profile of another user by their @handle.
// @Tags User
// @Produce json
// @Param username query string true "User handle to search for"
// @Security ApiKeyAuth
// @Success 200 {object} map[string]any
// @Router /api/user/search [get]
func (h *UserHandler) SearchUserByUsernameHandler(w http.ResponseWriter, r *http.Request) {
	authClaims := middleware.GetUserClaims(r)
	if authClaims == nil {
		h.respondError(w, http.StatusUnauthorized, "You must be logged in to search for other users")
		return
	}

	// Read and clean the target username from the query parameters
	targetUsername := strings.TrimSpace(r.URL.Query().Get("username"))
	if targetUsername == "" {
		h.respondError(w, http.StatusBadRequest, "A valid 'username' query parameter is required")
		return
	}

	cacheKey := fmt.Sprintf("cache:user:search:%s", strings.ToLower(targetUsername))
	var foundUser *User
	if hit, err := coreRedis.CacheGetJSON(r.Context(), cacheKey, &foundUser); err == nil && hit && foundUser != nil {
		publicProfile := map[string]any{
			"id":       foundUser.ID,
			"username": foundUser.Username,
			"name":     foundUser.Name,
			"status":   foundUser.Status,
			"country":  foundUser.Country,
		}
		h.respond(w, http.StatusOK, publicProfile)
		return
	}

	// Look up the user via the repository (handles case-insensitive regex matching)
	foundUser, err := h.repo.GetUserByUsername(r.Context(), targetUsername)
	if err != nil || foundUser == nil {
		h.respondError(w, http.StatusNotFound, "No user corresponds to that username")
		return
	}

	// Security: Return only public profile data — never leak password hashes or secrets.
	publicProfile := map[string]any{
		"id":       foundUser.ID,
		"username": foundUser.Username,
		"name":     foundUser.Name,
		"status":   foundUser.Status,
		"country":  foundUser.Country,
	}
	_ = coreRedis.CacheSetJSON(r.Context(), cacheKey, foundUser, userSearchCacheTTL)
	h.respond(w, http.StatusOK, publicProfile)
}
