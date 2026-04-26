// Package user handles the core user data model and database operations.
// We use the Repository Pattern here to make the code testable and easy to maintain.
package user

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/Moeed-ul-Hassan/chatapp/core/db"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// UserStore defines the blueprint for database operations on the User entity.
// By defining an interface, we can swap the real MongoDB implementation with a 'Mock'
// during unit testing, allowing us to test our logic without needing a live database.
type UserStore interface {
	CreateUser(ctx context.Context, clerkID, username, email string) (*User, error)
	UpdateUserIdentity(ctx context.Context, userID, preferredUsername, email string) (*User, error)
	GetUserByEmail(ctx context.Context, email string) (*User, error)
	GetUserByID(ctx context.Context, id string) (*User, error)
	GetUsersByIDs(ctx context.Context, ids []string) (map[string]*User, error)
	GetUserByClerkID(ctx context.Context, clerkID string) (*User, error)
	GetUserByUsername(ctx context.Context, username string) (*User, error)
	VerifyUser(ctx context.Context, email string) error
	UpdateLastSeen(ctx context.Context, userID string) error
}

// User represents a human individual in the Echo system.
type User struct {
	ID            string     `json:"id" bson:"_id"`                  // Unique UUID string
	ClerkID       string     `json:"clerk_id" bson:"clerk_id"`       // Link to Clerk user
	Username      string     `json:"username" bson:"username"`       // Human-readable handle
	UsernameLower string     `json:"-" bson:"username_lower"`        // Normalized username for case-insensitive uniqueness
	Email         string     `json:"email" bson:"email"`             // Verified email address
	Name          string     `json:"name" bson:"name"`               // Full display name
	Bio           string     `json:"bio" bson:"bio"`                 // Short bio or status
	Country       string     `json:"country" bson:"country"`         // Country detected via IP
	IsVerified    bool       `json:"is_verified" bson:"is_verified"` // Has user verified email?
	CreatedAt     time.Time  `json:"created_at" bson:"created_at"`   // Creation timestamp
	LastSeen      *time.Time `json:"last_seen" bson:"last_seen"`     // Last interaction timestamp
	Status        string     `json:"status" bson:"status"`           // Current status message
	IsAdmin       bool       `json:"is_admin" bson:"is_admin"`       // Admin flag
}

// UserRepository is the concrete implementation of UserStore using MongoDB.
type UserRepository struct {
	collection *mongo.Collection
}

// NewUserRepository returns a new instance of the repository.
func NewUserRepository() UserStore {
	return &UserRepository{
		collection: db.Database.Collection("users"),
	}
}

// CreateUser generates a new user record linked to a Clerk identity.
func (repo *UserRepository) CreateUser(ctx context.Context, clerkID, username, email string) (*User, error) {
	uniqueUsername, err := repo.GenerateUniqueUsername(ctx, username, clerkID, email)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(email) == "" {
		email = clerkID + "@echo.local"
	}

	newUser := &User{
		ID:            uuid.New().String(),
		ClerkID:       clerkID,
		Username:      uniqueUsername,
		UsernameLower: strings.ToLower(uniqueUsername),
		Email:         email,
		IsVerified:    true, // Clerk users are typically verified by Clerk
		CreatedAt:     time.Now(),
		IsAdmin:       false,
	}

	_, err = repo.collection.InsertOne(ctx, newUser)
	if err != nil {
		return nil, err
	}
	return newUser, nil
}

// UpdateUserIdentity ensures username/email are populated and normalized.
func (repo *UserRepository) UpdateUserIdentity(ctx context.Context, userID, preferredUsername, email string) (*User, error) {
	foundUser, err := repo.GetUserByID(ctx, userID)
	if err != nil || foundUser == nil {
		return nil, err
	}

	needsUsername := strings.TrimSpace(foundUser.Username) == ""
	needsEmail := strings.TrimSpace(foundUser.Email) == ""
	if !needsUsername && !needsEmail {
		return foundUser, nil
	}

	update := bson.M{}
	if needsUsername {
		uniqueUsername, genErr := repo.GenerateUniqueUsername(ctx, preferredUsername, foundUser.ClerkID, email)
		if genErr != nil {
			return nil, genErr
		}
		update["username"] = uniqueUsername
		update["username_lower"] = strings.ToLower(uniqueUsername)
	}
	if needsEmail {
		nextEmail := strings.TrimSpace(email)
		if nextEmail == "" {
			nextEmail = foundUser.ClerkID + "@echo.local"
		}
		update["email"] = nextEmail
	}

	_, err = repo.collection.UpdateOne(ctx, bson.M{"_id": userID}, bson.M{"$set": update})
	if err != nil {
		return nil, err
	}
	return repo.GetUserByID(ctx, userID)
}

// GetUserByEmail locates a user by their verified email address.
func (repo *UserRepository) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	foundUser := &User{}
	err := repo.collection.FindOne(ctx, bson.M{"email": email}).Decode(foundUser)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, err
	}
	return foundUser, nil
}

// GetUserByID locates a user by their unique UUID string.
func (repo *UserRepository) GetUserByID(ctx context.Context, id string) (*User, error) {
	foundUser := &User{}
	err := repo.collection.FindOne(ctx, bson.M{"_id": id}).Decode(foundUser)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, err
	}
	return foundUser, nil
}

// GetUsersByIDs fetches multiple users in a single query and returns map[id]user.
func (repo *UserRepository) GetUsersByIDs(ctx context.Context, ids []string) (map[string]*User, error) {
	result := map[string]*User{}
	if len(ids) == 0 {
		return result, nil
	}
	unique := make(map[string]struct{}, len(ids))
	normalizedIDs := make([]string, 0, len(ids))
	for _, id := range ids {
		if id == "" {
			continue
		}
		if _, exists := unique[id]; exists {
			continue
		}
		unique[id] = struct{}{}
		normalizedIDs = append(normalizedIDs, id)
	}
	cursor, err := repo.collection.Find(ctx, bson.M{"_id": bson.M{"$in": normalizedIDs}})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	var users []*User
	if err := cursor.All(ctx, &users); err != nil {
		return nil, err
	}
	for _, u := range users {
		if u != nil {
			result[u.ID] = u
		}
	}
	return result, nil
}

// GetUserByUsername locates a user by their unique handle using case-insensitive comparison.
func (repo *UserRepository) GetUserByUsername(ctx context.Context, username string) (*User, error) {
	foundUser := &User{}
	normalized := NormalizeUsername(username)
	filter := bson.M{"username_lower": strings.ToLower(normalized)}
	err := repo.collection.FindOne(ctx, filter).Decode(foundUser)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			// Fallback for legacy records created before username_lower existed.
			legacyFilter := bson.M{
				"username": bson.M{"$regex": "^" + normalized + "$", "$options": "i"},
			}
			legacyErr := repo.collection.FindOne(ctx, legacyFilter).Decode(foundUser)
			if legacyErr == mongo.ErrNoDocuments {
				return nil, nil
			}
			if legacyErr != nil {
				return nil, legacyErr
			}
			return foundUser, nil
		}
		return nil, err
	}
	return foundUser, nil
}

// GetUserByClerkID locates a user by their external Clerk identifier.
func (repo *UserRepository) GetUserByClerkID(ctx context.Context, clerkID string) (*User, error) {
	foundUser := &User{}
	err := repo.collection.FindOne(ctx, bson.M{"clerk_id": clerkID}).Decode(foundUser)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, err
	}
	return foundUser, nil
}

// VerifyUser marks an account as verified by email.
func (repo *UserRepository) VerifyUser(ctx context.Context, email string) error {
	_, err := repo.collection.UpdateOne(
		ctx,
		bson.M{"email": email},
		bson.M{"$set": bson.M{"is_verified": true}},
	)
	return err
}

// UpdateLastSeen stamps the current time onto the user profile.
func (repo *UserRepository) UpdateLastSeen(ctx context.Context, userID string) error {
	now := time.Now()
	_, err := repo.collection.UpdateOne(
		ctx,
		bson.M{"_id": userID},
		bson.M{"$set": bson.M{"last_seen": &now}},
	)
	return err
}

// NormalizeUsername converts arbitrary user input into a safe username handle.
func NormalizeUsername(input string) string {
	cleaned := strings.TrimSpace(strings.ToLower(input))
	cleaned = strings.TrimPrefix(cleaned, "@")
	allowed := regexp.MustCompile(`[^a-z0-9_.]`)
	cleaned = allowed.ReplaceAllString(cleaned, "")
	if cleaned == "" {
		cleaned = "user"
	}
	if len(cleaned) > 20 {
		cleaned = cleaned[:20]
	}
	return cleaned
}

// GenerateUniqueUsername returns a collision-free username.
func (repo *UserRepository) GenerateUniqueUsername(ctx context.Context, preferred, clerkID, email string) (string, error) {
	base := NormalizeUsername(preferred)
	if base == "user" {
		if email != "" {
			base = NormalizeUsername(strings.Split(email, "@")[0])
		} else if clerkID != "" {
			base = NormalizeUsername(clerkID)
		}
	}
	if base == "" {
		base = "user"
	}

	// Try base first, then suffixes.
	for i := 0; i < 5000; i++ {
		candidate := base
		if i > 0 {
			candidate = fmt.Sprintf("%s%d", base, i)
		}
		if len(candidate) > 20 {
			candidate = candidate[:20]
		}
		existing, err := repo.GetUserByUsername(ctx, candidate)
		if err != nil {
			return "", err
		}
		if existing == nil {
			return candidate, nil
		}
	}

	return "", fmt.Errorf("unable to generate unique username")
}
