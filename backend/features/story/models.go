package story

import (
	"context"
	"errors"
	"time"

	"github.com/Moeed-ul-Hassan/chatapp/core/db"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Story represents an ephemeral photo/video post that expires after 24 hours.
type Story struct {
	ID        string    `json:"id" bson:"_id"`
	UserID    string    `json:"user_id" bson:"user_id"`
	MediaURL  string    `json:"media_url" bson:"media_url"`
	MediaType string    `json:"media_type" bson:"media_type"` // image, video
	Viewers   []Viewer  `json:"viewers" bson:"viewers"`       // Users who swiped up/viewed
	CreatedAt time.Time `json:"created_at" bson:"created_at"`
	ExpiresAt time.Time `json:"expires_at" bson:"expires_at"` // Set to CreatedAt + 24h
}

// Viewer tracks who saw a story and at what time.
type Viewer struct {
	UserID   string    `json:"user_id" bson:"user_id"`
	ViewedAt time.Time `json:"viewed_at" bson:"viewed_at"`
}

// CreateStory persists a new ephemeral post.
// It automatically calculated the 24-hour expiration for the MongoDB TTL index.
func CreateStory(ctx context.Context, userID, mediaURL, mediaType string) (*Story, error) {
	now := time.Now()
	story := &Story{
		ID:        uuid.New().String(),
		UserID:    userID,
		MediaURL:  mediaURL,
		MediaType: mediaType,
		Viewers:   []Viewer{},
		CreatedAt: now,
		ExpiresAt: now.Add(24 * time.Hour), // Enforced by TTL index in mongo.go
	}

	_, err := db.Database.Collection("stories").InsertOne(ctx, story)
	if err != nil {
		return nil, err
	}
	return story, nil
}

// GetActiveStoriesByUsers fetches non-expired stories for a list of user IDs.
// This is used to populate the Story Bar for the user's circle.
func GetActiveStoriesByUsers(ctx context.Context, userIDs []string) ([]Story, error) {
	filter := bson.M{
		"expires_at": bson.M{"$gt": time.Now()}, // Double check expiration
	}
	if len(userIDs) > 0 {
		filter["user_id"] = bson.M{"$in": userIDs}
	}
	options := options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}})

	cursor, err := db.Database.Collection("stories").Find(ctx, filter, options)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var stories []Story
	if err := cursor.All(ctx, &stories); err != nil {
		return nil, err
	}
	return stories, nil
}

// AddStoryViewer records a view for a story.
// Privacy Rule: The caller must ensure the viewer has a conversation with the story owner.
func AddStoryViewer(ctx context.Context, storyID, viewerID string) error {
	filter := bson.M{
		"_id":             storyID,
		"viewers.user_id": bson.M{"$ne": viewerID}, // Only add if not already viewed
	}

	update := bson.M{
		"$push": bson.M{
			"viewers": Viewer{
				UserID:   viewerID,
				ViewedAt: time.Now(),
			},
		},
	}

	result, err := db.Database.Collection("stories").UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return errors.New("story not found or already viewed")
	}
	return nil
}

// GetStoryViewers returns the list of people who viewed a story.
// Only the story owner should be able to call this.
func GetStoryViewers(ctx context.Context, storyID string) ([]Viewer, error) {
	story := &Story{}
	err := db.Database.Collection("stories").FindOne(ctx, bson.M{"_id": storyID}).Decode(story)
	if err != nil {
		return nil, err
	}
	return story.Viewers, nil
}

// GetStoryByID fetches a single story document by ID.
func GetStoryByID(ctx context.Context, storyID string) (*Story, error) {
	story := &Story{}
	err := db.Database.Collection("stories").FindOne(ctx, bson.M{"_id": storyID}).Decode(story)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, err
	}
	return story, nil
}
