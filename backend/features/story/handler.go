package story

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Moeed-ul-Hassan/chatapp/core/authz"
	"github.com/Moeed-ul-Hassan/chatapp/core/middleware"
	coreRedis "github.com/Moeed-ul-Hassan/chatapp/core/redis"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

const maxStoryUploadSize = 15 << 20 // 15 MB
const storiesFeedCacheKey = "cache:stories:active:global"
const storiesFeedCacheTTL = 30 * time.Second

func resolveRequesterUserID(r *http.Request) (string, bool) {
	return authz.ResolveLocalUserID(r)
}

func respond(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func respondError(w http.ResponseWriter, status int, msg string) {
	respond(w, status, map[string]string{"error": msg})
}

// RegisterRoutes mounts story endpoints under /api/stories.
func RegisterRoutes(r chi.Router) {
	r.Group(func(protected chi.Router) {
		protected.Use(middleware.AuthRequired)
		protected.Get("/", ListActiveStoriesHandler)
		protected.Post("/", CreateStoryHandler)
		protected.Post("/{storyID}/view", MarkStoryViewedHandler)
		protected.Get("/{storyID}/viewers", GetStoryViewersHandler)
	})
}

// ListActiveStoriesHandler returns all non-expired stories.
// @Summary List active stories
// @Tags stories
// @Security ApiKeyAuth
// @Success 200 {array} Story
// @Failure 500 {object} map[string]string
// @Router /api/stories/ [get]
func ListActiveStoriesHandler(w http.ResponseWriter, r *http.Request) {
	var cachedStories []Story
	if hit, err := coreRedis.CacheGetJSON(r.Context(), storiesFeedCacheKey, &cachedStories); err == nil && hit {
		respond(w, http.StatusOK, cachedStories)
		return
	}

	stories, err := GetActiveStoriesByUsers(r.Context(), []string{})
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to fetch stories")
		return
	}
	_ = coreRedis.CacheSetJSON(r.Context(), storiesFeedCacheKey, stories, storiesFeedCacheTTL)
	respond(w, http.StatusOK, stories)
}

// CreateStoryHandler supports either:
// - multipart form upload with "file"
// - JSON body with media_url/media_type
// @Summary Create story
// @Tags stories
// @Security ApiKeyAuth
// @Accept json
// @Accept mpfd
// @Produce json
// @Success 201 {object} Story
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/stories/ [post]
func CreateStoryHandler(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetUserClaims(r)
	if claims == nil || claims.Subject == "" {
		respondError(w, http.StatusUnauthorized, "authentication required")
		return
	}
	requesterUserID, ok := resolveRequesterUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "unable to resolve current user")
		return
	}

	contentType := r.Header.Get("Content-Type")
	var mediaURL string
	var mediaType string

	if strings.Contains(contentType, "multipart/form-data") {
		if err := r.ParseMultipartForm(maxStoryUploadSize); err != nil {
			respondError(w, http.StatusBadRequest, "invalid multipart form")
			return
		}
		file, fileHeader, err := r.FormFile("file")
		if err != nil {
			respondError(w, http.StatusBadRequest, "file is required")
			return
		}
		defer file.Close()

		ext := strings.ToLower(filepath.Ext(fileHeader.Filename))
		if ext == "" {
			ext = ".bin"
		}
		allowed := map[string]string{
			".jpg":  "image",
			".jpeg": "image",
			".png":  "image",
			".webp": "image",
			".gif":  "image",
			".mp4":  "video",
			".webm": "video",
		}
		mt, ok := allowed[ext]
		if !ok {
			respondError(w, http.StatusBadRequest, "unsupported file type")
			return
		}
		mediaType = mt

		storyDir := filepath.Join("uploads", "stories")
		if err := os.MkdirAll(storyDir, 0o755); err != nil {
			respondError(w, http.StatusInternalServerError, "failed to prepare upload directory")
			return
		}

		filename := fmt.Sprintf("%d-%s%s", time.Now().Unix(), uuid.New().String(), ext)
		destPath := filepath.Join(storyDir, filename)
		dest, err := os.Create(destPath)
		if err != nil {
			respondError(w, http.StatusInternalServerError, "failed to save upload")
			return
		}
		defer dest.Close()

		if _, err := dest.ReadFrom(file); err != nil {
			respondError(w, http.StatusInternalServerError, "failed to write upload")
			return
		}

		scheme := "http"
		if r.TLS != nil || strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https") {
			scheme = "https"
		}
		mediaURL = fmt.Sprintf("%s://%s/uploads/stories/%s", scheme, r.Host, filename)
	} else {
		var body struct {
			MediaURL  string `json:"media_url"`
			MediaType string `json:"media_type"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			respondError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		mediaURL = strings.TrimSpace(body.MediaURL)
		mediaType = strings.TrimSpace(strings.ToLower(body.MediaType))
		if mediaURL == "" || (mediaType != "image" && mediaType != "video") {
			respondError(w, http.StatusBadRequest, "media_url and valid media_type are required")
			return
		}
	}

	story, err := CreateStory(r.Context(), requesterUserID, mediaURL, mediaType)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to create story")
		return
	}
	_ = coreRedis.CacheDelete(r.Context(), storiesFeedCacheKey)
	respond(w, http.StatusCreated, story)
}

// @Summary Mark story viewed
// @Tags stories
// @Security ApiKeyAuth
// @Param storyID path string true "Story ID"
// @Success 200 {object} map[string]bool
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/stories/{storyID}/view [post]
func MarkStoryViewedHandler(w http.ResponseWriter, r *http.Request) {
	requesterUserID, ok := resolveRequesterUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "authentication required")
		return
	}
	storyID := chi.URLParam(r, "storyID")
	if storyID == "" {
		respondError(w, http.StatusBadRequest, "story ID is required")
		return
	}
	if err := AddStoryViewer(r.Context(), storyID, requesterUserID); err != nil {
		respondError(w, http.StatusInternalServerError, "failed to mark story as viewed")
		return
	}
	_ = coreRedis.CacheDelete(r.Context(), storiesFeedCacheKey)
	respond(w, http.StatusOK, map[string]bool{"ok": true})
}

// @Summary Get story viewers
// @Tags stories
// @Security ApiKeyAuth
// @Param storyID path string true "Story ID"
// @Success 200 {array} Viewer
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/stories/{storyID}/viewers [get]
func GetStoryViewersHandler(w http.ResponseWriter, r *http.Request) {
	requesterUserID, ok := resolveRequesterUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "authentication required")
		return
	}
	storyID := chi.URLParam(r, "storyID")
	if storyID == "" {
		respondError(w, http.StatusBadRequest, "story ID is required")
		return
	}

	storyDoc, err := GetStoryByID(r.Context(), storyID)
	if err != nil || storyDoc == nil {
		respondError(w, http.StatusNotFound, "story not found")
		return
	}
	if storyDoc.UserID != requesterUserID {
		respondError(w, http.StatusForbidden, "you can only view viewers for your own story")
		return
	}
	viewers, err := GetStoryViewers(r.Context(), storyID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to fetch viewers")
		return
	}
	respond(w, http.StatusOK, viewers)
}
