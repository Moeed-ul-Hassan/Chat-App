package room

import (
	"context"
	"fmt"
	"time"

	"github.com/Moeed-ul-Hassan/chatapp/core/db"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// Room represents a private encrypted vault.
type Room struct {
	ID         string    `json:"id" bson:"_id"`
	RoomID     string    `json:"room_id" bson:"room_id"` // Human-readable ID (e.g. "Echo-1234")
	Name       string    `json:"name" bson:"name"`
	Passkey    string    `json:"-" bson:"passkey"` // Hashed passkey (Argon2id)
	OwnerID    string    `json:"owner_id" bson:"owner_id"`
	InviteOnly bool      `json:"invite_only" bson:"invite_only"`
	Require2FA bool      `json:"require_2fa" bson:"require_2fa"`
	TTLHours   *int      `json:"ttl_hours,omitempty" bson:"ttl_hours,omitempty"`
	AESKey     string    `json:"aes_key" bson:"aes_key"` // E2EE Master Key (encrypted or base64)
	CreatedAt  time.Time `json:"created_at" bson:"created_at"`
}

// CreateRoom inserts a new private room into the database.
func CreateRoom(ctx context.Context, r *Room) (*Room, error) {
	if r.RoomID == "" {
		r.RoomID = uuid.New().String()[:8] // Default short ID
	}

	if r.ID == "" {
		r.ID = uuid.New().String()
	}

	r.CreatedAt = time.Now()

	_, err := db.Database.Collection("rooms").InsertOne(ctx, r)
	if err != nil {
		return nil, fmt.Errorf("failed to create room: %w", err)
	}
	return r, nil
}

// GetRoomByRoomID fetches a room by its human-readable ID (for searching).
func GetRoomByRoomID(ctx context.Context, roomID string) (*Room, error) {
	r := &Room{}
	err := db.Database.Collection("rooms").FindOne(ctx, bson.M{"room_id": roomID}).Decode(r)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("room not found")
		}
		return nil, err
	}
	return r, nil
}

// CheckPasskey matches the provided passkey against the hashed one.
// In Milestone 3, we will use utils.VerifyHash (Argon2id).
func (r *Room) CheckPasskey(provided string) bool {
	// Placeholder for Argon2id check (Milestone 3)
	return r.Passkey == provided
}
