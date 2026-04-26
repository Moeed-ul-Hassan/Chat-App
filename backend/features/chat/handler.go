package chat

import (
	"encoding/json"
	"net/http"
	"slices"
	"strconv"

	api "github.com/Moeed-ul-Hassan/chatapp/core/api"
	"github.com/Moeed-ul-Hassan/chatapp/core/authz"
	"github.com/Moeed-ul-Hassan/chatapp/core/middleware"
	"github.com/Moeed-ul-Hassan/chatapp/features/user"
	"github.com/go-chi/chi/v5"
)

var _ = api.ErrorResponse{}

// ─────────────────────────────────────────────
// Helpers
// ─────────────────────────────────────────────

// respond sends a JSON success response to the client.
func respond(w http.ResponseWriter, statusCode int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(payload)
}

// respondError sends a JSON error response with a human-readable message.
func respondError(w http.ResponseWriter, statusCode int, errorMessage string) {
	respond(w, statusCode, map[string]string{"error": errorMessage})
}

// resolveRequesterUserID maps Clerk subject -> local Echo user ID.
func resolveRequesterUserID(r *http.Request) (string, bool) {
	return authz.ResolveLocalUserID(r)
}

// ─────────────────────────────────────────────
// Route Registration
// ─────────────────────────────────────────────

// RegisterAPIRoutes attaches all chat-related REST API routes to the main router.
// All routes here require a valid PASETO token (enforced by AuthRequired middleware).
func RegisterAPIRoutes(r chi.Router) {
	r.Group(func(protectedRouter chi.Router) {
		protectedRouter.Use(middleware.AuthRequired)

		// ── Channel (World Chat) ──────────────────────────────────────
		// Fetch message history for a public world-chat channel
		protectedRouter.Get("/channels/{channelID}/messages", GetChannelMessagesHistoryHandler)

		// ── Messages (shared across rooms, channels, DMs) ────────────
		// Edit the text content of an existing message
		protectedRouter.Put("/messages/{id}", EditMessageHandler)
		// Soft-delete a message (replaces content with "🚫 This message was deleted")
		protectedRouter.Delete("/messages/{id}", DeleteMessageHandler)

		// ── Direct Messages (DM Conversations) ───────────────────────
		// List all DM conversations for the logged-in user (inbox)
		protectedRouter.Get("/conversations", ListConversationsHandler)
		// Start a new DM conversation by targeting another user's username
		protectedRouter.Post("/conversations/start", StartConversationHandler)
		// Accept a pending message request from another user
		protectedRouter.Post("/conversations/{conversationID}/accept", AcceptConversationHandler)
		// Fetch the full message history for a specific DM conversation
		protectedRouter.Get("/conversations/{conversationID}/messages", GetDMMessagesHandler)
	})
}

// ─────────────────────────────────────────────
// World Chat Handlers
// ─────────────────────────────────────────────

// GetChannelMessagesHistoryHandler fetches the latest messages for a public world-chat channel.
// Supports a ?limit= query param (default 50, max 200).
// GET /channels/{channelID}/messages
// @Summary Get channel messages
// @Description Returns channel message history ordered oldest-first.
// @Tags chat
// @Security ApiKeyAuth
// @Param channelID path string true "Channel ID"
// @Param limit query int false "Max messages (1-200)"
// @Success 200 {array} Message
// @Failure 400 {object} api.ErrorResponse
// @Failure 401 {object} api.ErrorResponse
// @Failure 500 {object} api.ErrorResponse
// @Router /api/chat/channels/{channelID}/messages [get]
func GetChannelMessagesHistoryHandler(w http.ResponseWriter, r *http.Request) {
	channelID := chi.URLParam(r, "channelID")
	if channelID == "" {
		respondError(w, http.StatusBadRequest, "Channel ID is required")
		return
	}

	// Parse optional limit parameter (defaults to 50 messages)
	requestedLimit := 50
	if rawLimit := r.URL.Query().Get("limit"); rawLimit != "" {
		if parsedLimit, err := strconv.Atoi(rawLimit); err == nil && parsedLimit > 0 && parsedLimit <= 200 {
			requestedLimit = parsedLimit
		}
	}

	channelMessages, err := GetMessagesByChannel(r.Context(), channelID, requestedLimit)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to fetch channel messages")
		return
	}

	// Always return an array, even if empty (avoids null in JSON)
	if channelMessages == nil {
		channelMessages = []Message{}
	}

	respond(w, http.StatusOK, channelMessages)
}

// ─────────────────────────────────────────────
// Message Action Handlers
// ─────────────────────────────────────────────

// EditMessageHandler updates the text content of an existing message.
// Only the original author can edit their own messages.
// PUT /messages/{id}
// @Summary Edit a message
// @Tags chat
// @Security ApiKeyAuth
// @Param id path string true "Message ID"
// @Param body body map[string]string true "content + room"
// @Success 200 {object} Message
// @Failure 400 {object} api.ErrorResponse
// @Failure 401 {object} api.ErrorResponse
// @Failure 403 {object} api.ErrorResponse
// @Router /api/chat/messages/{id} [put]
func EditMessageHandler(w http.ResponseWriter, r *http.Request) {
	requesterUserID, ok := resolveRequesterUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "You must be logged in to edit messages")
		return
	}

	messageID := chi.URLParam(r, "id")

	// Parse request body
	var requestBody struct {
		NewContent string `json:"content"` // The updated message text
		RoomID     string `json:"room"`    // Room or channel this message belongs to (for WS broadcast)
	}
	if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	if requestBody.NewContent == "" {
		respondError(w, http.StatusBadRequest, "Message content cannot be empty")
		return
	}

	// Apply the edit (only allowed if requesterUserID matches message.UserID)
	updatedMessage, err := UpdateMessage(r.Context(), messageID, requesterUserID, requestBody.NewContent)
	if err != nil {
		respondError(w, http.StatusForbidden, "Cannot edit this message — it may not exist or belong to another user")
		return
	}

	// Determine the broadcast target (room or channel ID)
	broadcastRoomID := requestBody.RoomID
	if broadcastRoomID == "" && updatedMessage.ChannelID != nil {
		broadcastRoomID = *updatedMessage.ChannelID
	} else if broadcastRoomID == "" && updatedMessage.RoomID != nil {
		broadcastRoomID = *updatedMessage.RoomID
	}

	// Push a real-time "message edited" event to all connected clients in this room
	if broadcastRoomID != "" && updatedMessage.EditedAt != nil {
		go func() {
			BroadcastServerMsg(MetaMessage{
				Action:    "msg_edited",
				MessageID: updatedMessage.ID,
				RoomID:    broadcastRoomID,
				Content:   updatedMessage.Content,
				Timestamp: *updatedMessage.EditedAt,
			})
		}()
	}

	respond(w, http.StatusOK, updatedMessage)
}

// DeleteMessageHandler soft-deletes a message, replacing its content with a placeholder.
// Only the original author can delete their own messages.
// DELETE /messages/{id}
// @Summary Delete a message
// @Tags chat
// @Security ApiKeyAuth
// @Param id path string true "Message ID"
// @Param room query string false "Room/channel ID for realtime broadcast"
// @Success 200 {object} Message
// @Failure 401 {object} api.ErrorResponse
// @Failure 403 {object} api.ErrorResponse
// @Router /api/chat/messages/{id} [delete]
func DeleteMessageHandler(w http.ResponseWriter, r *http.Request) {
	requesterUserID, ok := resolveRequesterUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "You must be logged in to delete messages")
		return
	}

	messageID := chi.URLParam(r, "id")
	// Optional: which room to broadcast the deletion event to
	broadcastRoomID := r.URL.Query().Get("room")

	deletedMessage, err := DeleteMessage(r.Context(), messageID, requesterUserID)
	if err != nil {
		respondError(w, http.StatusForbidden, "Cannot delete this message — it may not exist or belong to another user")
		return
	}

	// Fallback: determine the broadcast room from the message itself
	if broadcastRoomID == "" && deletedMessage.ChannelID != nil {
		broadcastRoomID = *deletedMessage.ChannelID
	} else if broadcastRoomID == "" && deletedMessage.RoomID != nil {
		broadcastRoomID = *deletedMessage.RoomID
	}

	// Notify all room members in real-time that this message was deleted
	if broadcastRoomID != "" {
		go func() {
			BroadcastServerMsg(MetaMessage{
				Action:    "msg_deleted",
				MessageID: deletedMessage.ID,
				RoomID:    broadcastRoomID,
				Content:   "🚫 This message was deleted",
			})
		}()
	}

	respond(w, http.StatusOK, deletedMessage)
}

// ─────────────────────────────────────────────
// DM Conversation Handlers
// ─────────────────────────────────────────────

// ListConversationsHandler returns all DM conversations for the authenticated user.
// This populates the main chat list (inbox) on the frontend.
// GET /conversations
// @Summary List conversations
// @Tags chat
// @Security ApiKeyAuth
// @Success 200 {array} map[string]interface{}
// @Failure 401 {object} api.ErrorResponse
// @Failure 500 {object} api.ErrorResponse
// @Router /api/chat/conversations [get]
func ListConversationsHandler(w http.ResponseWriter, r *http.Request) {
	requesterUserID, ok := resolveRequesterUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	allConversations, err := ListUserConversations(r.Context(), requesterUserID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to load conversations")
		return
	}

	// Always return an array so the frontend can safely iterate
	if allConversations == nil {
		allConversations = []Conversation{}
	}

	// Enrich each conversation with "other_user" profile so frontend can render receiver/sender info.
	userRepo := user.NewUserRepository()
	type conversationWithOtherUser struct {
		Conversation
		OtherUser map[string]any `json:"other_user,omitempty"`
	}
	otherIDs := make([]string, 0, len(allConversations))
	for _, conv := range allConversations {
		for _, participantID := range conv.Participants {
			if participantID != requesterUserID {
				otherIDs = append(otherIDs, participantID)
				break
			}
		}
	}
	otherUsersByID := map[string]*user.User{}
	if len(otherIDs) > 0 {
		if usersByID, batchErr := userRepo.GetUsersByIDs(r.Context(), otherIDs); batchErr == nil {
			otherUsersByID = usersByID
		}
	}
	response := make([]conversationWithOtherUser, 0, len(allConversations))
	for _, conv := range allConversations {
		item := conversationWithOtherUser{Conversation: conv}
		for _, participantID := range conv.Participants {
			if participantID == requesterUserID {
				continue
			}
			other := otherUsersByID[participantID]
			if other != nil {
				item.OtherUser = map[string]any{
					"id":       other.ID,
					"username": other.Username,
					"name":     other.Name,
					"status":   other.Status,
				}
			}
			break
		}
		response = append(response, item)
	}

	respond(w, http.StatusOK, response)
}

// StartConversationHandler finds a user by their unique username and starts (or retrieves) a DM.
// If a conversation already exists between these two users, it returns the existing one.
// If not, it creates a new PENDING conversation (Instagram-style message request).
// POST /conversations/start
// Body: { "target_username": "john" }
// @Summary Start conversation
// @Tags chat
// @Security ApiKeyAuth
// @Param body body map[string]string true "target_username"
// @Success 200 {object} map[string]interface{}
// @Success 201 {object} map[string]interface{}
// @Failure 400 {object} api.ErrorResponse
// @Failure 401 {object} api.ErrorResponse
// @Failure 404 {object} api.ErrorResponse
// @Failure 500 {object} api.ErrorResponse
// @Router /api/chat/conversations/start [post]
func StartConversationHandler(w http.ResponseWriter, r *http.Request) {
	requesterUserID, ok := resolveRequesterUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	var requestBody struct {
		TargetUsername string `json:"target_username"` // The unique username handle of the person to message
	}
	if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
		respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	if requestBody.TargetUsername == "" {
		respondError(w, http.StatusBadRequest, "target_username is required")
		return
	}

	// Look up the target user by their username handle
	userRepo := user.NewUserRepository()
	targetUser, err := userRepo.GetUserByUsername(r.Context(), requestBody.TargetUsername)
	if err != nil || targetUser == nil {
		respondError(w, http.StatusNotFound, "No user found with that username")
		return
	}

	// Prevent messaging yourself
	if targetUser.ID == requesterUserID {
		respondError(w, http.StatusBadRequest, "You cannot start a conversation with yourself")
		return
	}

	// Check if a conversation between these two users already exists
	existingConversation, err := GetConversationByParticipants(r.Context(), requesterUserID, targetUser.ID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Error checking existing conversations")
		return
	}

	// Return the existing conversation if found (don't create duplicates)
	if existingConversation != nil {
		respond(w, http.StatusOK, map[string]any{
			"conversation": existingConversation,
			"other_user":   targetUser,
			"is_new":       false,
		})
		return
	}

	// Create a new PENDING DM conversation (the other user must accept before full messaging)
	newConversation, err := CreateConversation(r.Context(), requesterUserID, targetUser.ID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to create conversation")
		return
	}

	respond(w, http.StatusCreated, map[string]any{
		"conversation": newConversation,
		"other_user":   targetUser,
		"is_new":       true,
	})
}

// AcceptConversationHandler transitions a PENDING message request to ACCEPTED.
// Only the receiver (non-initiator) can accept a request.
// POST /conversations/{conversationID}/accept
// @Summary Accept conversation request
// @Tags chat
// @Security ApiKeyAuth
// @Param conversationID path string true "Conversation ID"
// @Success 200 {object} map[string]string
// @Failure 400 {object} api.ErrorResponse
// @Failure 401 {object} api.ErrorResponse
// @Failure 403 {object} api.ErrorResponse
// @Failure 404 {object} api.ErrorResponse
// @Failure 409 {object} api.ErrorResponse
// @Failure 500 {object} api.ErrorResponse
// @Router /api/chat/conversations/{conversationID}/accept [post]
func AcceptConversationHandler(w http.ResponseWriter, r *http.Request) {
	requesterUserID, ok := resolveRequesterUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	conversationID := chi.URLParam(r, "conversationID")
	if conversationID == "" {
		respondError(w, http.StatusBadRequest, "Conversation ID is required")
		return
	}

	conversation, err := GetConversationByID(r.Context(), conversationID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to load conversation")
		return
	}
	if conversation == nil {
		respondError(w, http.StatusNotFound, "Conversation not found")
		return
	}
	if !slices.Contains(conversation.Participants, requesterUserID) {
		respondError(w, http.StatusForbidden, "You are not part of this conversation")
		return
	}
	// Only the non-initiator can accept a pending request.
	if conversation.InitiatorID == requesterUserID {
		respondError(w, http.StatusForbidden, "Initiator cannot accept their own request")
		return
	}
	if conversation.Status != "PENDING" {
		respondError(w, http.StatusConflict, "Conversation is not pending")
		return
	}

	// Mark the conversation as fully accepted in the database.
	if err := AcceptConversationRequest(r.Context(), conversationID); err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to accept the conversation request")
		return
	}

	// Notify the initiating user in real-time that their request was accepted
	go func() {
		BroadcastServerMsg(MetaMessage{
			Action:         "request_accepted",
			ConversationID: conversationID,
			SenderID:       requesterUserID,
		})
	}()

	respond(w, http.StatusOK, map[string]string{
		"message": "Conversation request accepted. You can now message freely.",
	})
}

// GetDMMessagesHandler fetches the message history for a specific 1-on-1 DM conversation.
// GET /conversations/{conversationID}/messages
// @Summary Get DM messages
// @Tags chat
// @Security ApiKeyAuth
// @Param conversationID path string true "Conversation ID"
// @Param limit query int false "Max messages (1-200)"
// @Success 200 {array} Message
// @Failure 400 {object} api.ErrorResponse
// @Failure 401 {object} api.ErrorResponse
// @Failure 403 {object} api.ErrorResponse
// @Failure 404 {object} api.ErrorResponse
// @Failure 500 {object} api.ErrorResponse
// @Router /api/chat/conversations/{conversationID}/messages [get]
func GetDMMessagesHandler(w http.ResponseWriter, r *http.Request) {
	requesterUserID, ok := resolveRequesterUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "Authentication required")
		return
	}

	conversationID := chi.URLParam(r, "conversationID")
	if conversationID == "" {
		respondError(w, http.StatusBadRequest, "Conversation ID is required")
		return
	}

	conversation, err := GetConversationByID(r.Context(), conversationID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to load conversation")
		return
	}
	if conversation == nil {
		respondError(w, http.StatusNotFound, "Conversation not found")
		return
	}
	if !slices.Contains(conversation.Participants, requesterUserID) {
		respondError(w, http.StatusForbidden, "You are not allowed to access this conversation")
		return
	}

	// Parse optional limit for pagination
	messageLimit := 50
	if rawLimit := r.URL.Query().Get("limit"); rawLimit != "" {
		if parsedLimit, err := strconv.Atoi(rawLimit); err == nil && parsedLimit > 0 && parsedLimit <= 200 {
			messageLimit = parsedLimit
		}
	}

	conversationMessages, err := GetMessagesByConversation(r.Context(), conversationID, messageLimit)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to fetch conversation messages")
		return
	}

	if conversationMessages == nil {
		conversationMessages = []Message{}
	}

	respond(w, http.StatusOK, conversationMessages)
}
