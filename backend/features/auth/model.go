// Package auth handles user sessions, OTP codes, and authentication logic.
// We use the Repository Pattern here to make the authentication flow testable.
package auth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"time"

	"github.com/Moeed-ul-Hassan/chatapp/core/db"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// AuthStore defines the required methods for session and OTP management.
// This interface allows us to mock the database layer for reliable unit testing.
type AuthStore interface {
	CreateSession(ctx context.Context, userID, rawToken, deviceInfo, ipAddress string) error
	GetSessionByToken(ctx context.Context, rawToken string) (*Session, error)
	DeleteSession(ctx context.Context, rawToken string) error
	DeleteAllUserSessions(ctx context.Context, userID string) error
	ListUserSessions(ctx context.Context, userID string) ([]Session, error)
	SaveOTP(ctx context.Context, email, code, purpose string) error
	CheckOTP(ctx context.Context, email, code, purpose string) (bool, error)
}

// Session represents an active login instance for a user.
type Session struct {
	ID         string    `json:"id" bson:"_id"`
	UserID     string    `json:"user_id" bson:"user_id"`
	TokenHash  string    `json:"-" bson:"token_hash"`
	DeviceInfo string    `json:"device_info" bson:"device_info"`
	IPAddress  string    `json:"ip_address" bson:"ip_address"`
	CreatedAt  time.Time `json:"created_at" bson:"created_at"`
	ExpiresAt  time.Time `json:"expires_at" bson:"expires_at"`
}

// OTPCode represents a short-lived security code for email verification or password reset.
type OTPCode struct {
	ID        string    `json:"id" bson:"_id"`
	Email     string    `json:"email" bson:"email"`
	Code      string    `json:"code" bson:"code"`
	Purpose   string    `json:"purpose" bson:"purpose"`
	IsUsed    bool      `json:"is_used" bson:"is_used"`
	CreatedAt time.Time `json:"created_at" bson:"created_at"`
	ExpiresAt time.Time `json:"expires_at" bson:"expires_at"`
}

// AuthRepository is the real-world implementation of AuthStore using MongoDB.
type AuthRepository struct {
	sessions *mongo.Collection
	otpCodes *mongo.Collection
}

// NewAuthRepository returns a new concrete repository instance.
func NewAuthRepository() AuthStore {
	return &AuthRepository{
		sessions: db.Database.Collection("sessions"),
		otpCodes: db.Database.Collection("otp_codes"),
	}
}

// HashToken generates a SHA-256 hash of a refresh token for secure database storage.
func HashToken(rawToken string) string {
	hasher := sha256.Sum256([]byte(rawToken))
	return hex.EncodeToString(hasher[:])
}

// CreateSession generates and persists a new session.
func (repo *AuthRepository) CreateSession(ctx context.Context, userID, rawToken, deviceInfo, ipAddress string) error {
	newSession := Session{
		ID:         uuid.New().String(),
		UserID:     userID,
		TokenHash:  HashToken(rawToken),
		DeviceInfo: deviceInfo,
		IPAddress:  ipAddress,
		CreatedAt:  time.Now(),
		ExpiresAt:  time.Now().AddDate(0, 0, 30),
	}
	_, err := repo.sessions.InsertOne(ctx, newSession)
	return err
}

// GetSessionByToken retrieves a session using a hashed version of the raw token.
func (repo *AuthRepository) GetSessionByToken(ctx context.Context, rawToken string) (*Session, error) {
	sessionRecord := &Session{}
	filter := bson.M{
		"token_hash": HashToken(rawToken),
		"expires_at": bson.M{"$gt": time.Now()},
	}
	err := repo.sessions.FindOne(ctx, filter).Decode(sessionRecord)
	if err != nil {
		return nil, err
	}
	return sessionRecord, nil
}

// DeleteSession removes a specific session (Logout).
func (repo *AuthRepository) DeleteSession(ctx context.Context, rawToken string) error {
	filter := bson.M{"token_hash": HashToken(rawToken)}
	_, err := repo.sessions.DeleteOne(ctx, filter)
	return err
}

// DeleteAllUserSessions removes all active sessions for a user (Security Wipe).
func (repo *AuthRepository) DeleteAllUserSessions(ctx context.Context, userID string) error {
	filter := bson.M{"user_id": userID}
	_, err := repo.sessions.DeleteMany(ctx, filter)
	return err
}

// ListUserSessions returns all valid (non-expired) sessions for a user ID.
func (repo *AuthRepository) ListUserSessions(ctx context.Context, userID string) ([]Session, error) {
	filter := bson.M{
		"user_id":    userID,
		"expires_at": bson.M{"$gt": time.Now()},
	}
	findOptions := options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}})

	cursor, err := repo.sessions.Find(ctx, filter, findOptions)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var sessions []Session
	if err := cursor.All(ctx, &sessions); err != nil {
		return nil, err
	}
	return sessions, nil
}

// SaveOTP stores a newly generated security code.
func (repo *AuthRepository) SaveOTP(ctx context.Context, email, code, purpose string) error {
	newOTP := OTPCode{
		ID:        uuid.New().String(),
		Email:     email,
		Code:      code,
		Purpose:   purpose,
		IsUsed:    false,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(10 * time.Minute),
	}
	_, err := repo.otpCodes.InsertOne(ctx, newOTP)
	return err
}

// CheckOTP verifies the provided code and marks it as used atomically.
func (repo *AuthRepository) CheckOTP(ctx context.Context, email, code, purpose string) (bool, error) {
	filter := bson.M{
		"email":      email,
		"code":       code,
		"purpose":    purpose,
		"is_used":    false,
		"expires_at": bson.M{"$gt": time.Now()},
	}
	update := bson.M{"$set": bson.M{"is_used": true}}

	result := repo.otpCodes.FindOneAndUpdate(ctx, filter, update)
	if result.Err() != nil {
		if result.Err() == mongo.ErrNoDocuments {
			return false, nil
		}
		return false, result.Err()
	}
	return true, nil
}
