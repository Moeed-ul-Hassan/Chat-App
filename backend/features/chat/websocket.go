package chat

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/Moeed-ul-Hassan/chatapp/features/user"
	"github.com/clerk/clerk-sdk-go/v2/jwt"
	"github.com/go-chi/chi/v5"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		allowedRaw := os.Getenv("WS_ALLOWED_ORIGINS")
		if strings.TrimSpace(allowedRaw) == "" {
			return true // developer-friendly fallback for local setup
		}
		origin := strings.TrimSpace(r.Header.Get("Origin"))
		for _, item := range strings.Split(allowedRaw, ",") {
			if strings.TrimSpace(item) == origin {
				return true
			}
		}
		return false
	},
}

var (
	hubMutex        sync.RWMutex
	roomConnections = make(map[string]map[*websocket.Conn]string)
	userConnections = make(map[string][]*websocket.Conn)
)

func writeJSONAsync(conn *websocket.Conn, event MetaMessage) <-chan error {
	done := make(chan error, 1)
	go func() {
		_ = conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
		done <- conn.WriteJSON(event)
	}()
	return done
}

func removeConnFromRoomLocked(targetRoom string, conn *websocket.Conn) {
	members, ok := roomConnections[targetRoom]
	if !ok {
		return
	}
	delete(members, conn)
	if len(members) == 0 {
		delete(roomConnections, targetRoom)
	}
}

func removeConnFromUserLocked(targetUserID string, conn *websocket.Conn) {
	existingConns, ok := userConnections[targetUserID]
	if !ok {
		return
	}
	for i, c := range existingConns {
		if c == conn {
			userConnections[targetUserID] = append(existingConns[:i], existingConns[i+1:]...)
			break
		}
	}
	if len(userConnections[targetUserID]) == 0 {
		delete(userConnections, targetUserID)
	}
}

type MetaMessage struct {
	Action         string    `json:"action"`
	MessageID      string    `json:"messageId,omitempty"`
	ConversationID string    `json:"conversationId,omitempty"`
	RoomID         string    `json:"roomId,omitempty"`
	SenderID       string    `json:"senderId,omitempty"`
	SenderName     string    `json:"senderName,omitempty"`
	TargetUserID   string    `json:"targetUserId,omitempty"`
	Content        string    `json:"content,omitempty"`
	MediaType      string    `json:"mediaType,omitempty"`
	VoiceURL       string    `json:"voiceUrl,omitempty"`
	IsOnline       bool      `json:"isOnline,omitempty"`
	Timestamp      time.Time `json:"timestamp"`
}

func RegisterRoutes(r chi.Router) {
	r.Get("/ws", HandleWebSocket)
}

func HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	authHeader := r.Header.Get("Authorization")
	tokenStr := ""
	if strings.HasPrefix(authHeader, "Bearer ") {
		tokenStr = strings.TrimPrefix(authHeader, "Bearer ")
	}
	allowQueryToken := strings.EqualFold(os.Getenv("ALLOW_WS_QUERY_TOKEN"), "true")
	if tokenStr == "" && allowQueryToken {
		tokenStr = strings.TrimSpace(r.URL.Query().Get("token"))
	}
	if tokenStr == "" {
		http.Error(w, "unauthorized: missing token", http.StatusUnauthorized)
		return
	}
	claims, err := jwt.Verify(r.Context(), &jwt.VerifyParams{Token: tokenStr})
	if err != nil || claims == nil || claims.Subject == "" {
		http.Error(w, "unauthorized: invalid token", http.StatusUnauthorized)
		return
	}
	userRepo := user.NewUserRepository()
	currentUser, err := userRepo.GetUserByClerkID(r.Context(), claims.Subject)
	if err != nil || currentUser == nil {
		http.Error(w, "unauthorized: user not found", http.StatusUnauthorized)
		return
	}

	wsConn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Println("WS upgrade error:", err)
		return
	}
	defer wsConn.Close()
	wsConn.SetReadLimit(1 << 20)
	_ = wsConn.SetReadDeadline(time.Now().Add(60 * time.Second))
	wsConn.SetPongHandler(func(string) error {
		return wsConn.SetReadDeadline(time.Now().Add(60 * time.Second))
	})

	roomID := r.URL.Query().Get("room")
	username := r.URL.Query().Get("username")
	userID := r.URL.Query().Get("userId")

	if roomID == "" || username == "" {
		wsConn.WriteJSON(MetaMessage{Action: "error", Content: "room and username are required"})
		return
	}
	if userID != "" && userID != currentUser.ID {
		wsConn.WriteJSON(MetaMessage{Action: "error", Content: "userId does not match authenticated token"})
		return
	}
	if userID == "" {
		userID = currentUser.ID
	}

	hubMutex.Lock()
	if _, exists := roomConnections[roomID]; !exists {
		roomConnections[roomID] = make(map[*websocket.Conn]string)
	}
	roomConnections[roomID][wsConn] = username

	if userID != "" {
		userConnections[userID] = append(userConnections[userID], wsConn)
	}
	hubMutex.Unlock()

	fmt.Printf("👤 %s (%s) joined room %s\n", username, userID, roomID)

	defer func() {
		hubMutex.Lock()
		removeConnFromRoomLocked(roomID, wsConn)
		if userID != "" {
			removeConnFromUserLocked(userID, wsConn)
		}
		hubMutex.Unlock()
		fmt.Printf("👋 %s left room %s\n", username, roomID)
	}()

	pingTicker := time.NewTicker(30 * time.Second)
	defer pingTicker.Stop()
	go func() {
		for range pingTicker.C {
			_ = wsConn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := wsConn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}()

	for {
		_, rawPayload, readErr := wsConn.ReadMessage()
		if readErr != nil {
			break
		}

		var event MetaMessage
		if err := json.Unmarshal(rawPayload, &event); err != nil {
			continue
		}

		switch event.Action {
		case "chat":
			if userID != "" {
				channelID := &event.RoomID
				savedMsg, saveErr := SaveMessage(r.Context(), nil, channelID, userID, event.Content, "text")
				if saveErr == nil {
					event.MessageID = savedMsg.ID
				} else {
					fmt.Println("❌ Failed to save message:", saveErr)
				}
			}
			event.Timestamp = time.Now()
			broadcastToRoom(event, wsConn, true)

		case "dm":
			if event.TargetUserID != "" && event.ConversationID != "" {
				conv, convErr := GetConversationByID(r.Context(), event.ConversationID)
				if convErr != nil || conv == nil {
					continue
				}
				isSenderParticipant := false
				isTargetParticipant := false
				for _, participant := range conv.Participants {
					if participant == userID {
						isSenderParticipant = true
					}
					if participant == event.TargetUserID {
						isTargetParticipant = true
					}
				}
				if !isSenderParticipant || !isTargetParticipant {
					continue
				}
				savedDM, saveErr := SaveDMMessage(r.Context(), event.ConversationID, userID, event.Content, "text")
				if saveErr == nil {
					event.MessageID = savedDM.ID
					_ = UpdateConversationActivity(r.Context(), event.ConversationID, event.Content)
				}
				event.SenderID = userID
				event.Timestamp = time.Now()
				sendToUser(event.TargetUserID, event)
				sendToUser(userID, event)
			}

		case "typing":
			if event.TargetUserID != "" {
				sendToUser(event.TargetUserID, event)
			}

		case "seen":
			if event.MessageID != "" {
				_ = MarkMessageAsSeen(r.Context(), event.MessageID)
				if event.TargetUserID != "" {
					sendToUser(event.TargetUserID, event)
				}
			}

		case "reaction":
			if event.TargetUserID != "" {
				sendToUser(event.TargetUserID, event)
			}
		}
	}
}

func BroadcastServerMsg(event MetaMessage) {
	broadcastToRoom(event, nil, true)
}

func broadcastToRoom(event MetaMessage, sender *websocket.Conn, echoToSender bool) {
	room := event.RoomID
	if room == "" {
		room = event.ConversationID
	}

	hubMutex.RLock()
	members, ok := roomConnections[room]
	if !ok {
		hubMutex.RUnlock()
		return
	}
	targetConns := make([]*websocket.Conn, 0, len(members))
	for conn := range members {
		if conn != sender || echoToSender {
			targetConns = append(targetConns, conn)
		}
	}
	hubMutex.RUnlock()

	for _, conn := range targetConns {
		select {
		case writeErr := <-writeJSONAsync(conn, event):
			if writeErr == nil {
				continue
			}
			_ = conn.Close()
			hubMutex.Lock()
			removeConnFromRoomLocked(room, conn)
			hubMutex.Unlock()
		case <-time.After(11 * time.Second):
			_ = conn.Close()
			hubMutex.Lock()
			removeConnFromRoomLocked(room, conn)
			hubMutex.Unlock()
		}
	}
}

func sendToUser(uid string, event MetaMessage) {
	hubMutex.RLock()
	targetConns := append([]*websocket.Conn(nil), userConnections[uid]...)
	hubMutex.RUnlock()

	for _, conn := range targetConns {
		select {
		case writeErr := <-writeJSONAsync(conn, event):
			if writeErr == nil {
				continue
			}
			_ = conn.Close()
			hubMutex.Lock()
			removeConnFromUserLocked(uid, conn)
			hubMutex.Unlock()
		case <-time.After(11 * time.Second):
			_ = conn.Close()
			hubMutex.Lock()
			removeConnFromUserLocked(uid, conn)
			hubMutex.Unlock()
		}
	}
}
