package chat

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

type Channel struct {
	ID          string    `json:"id" bson:"_id"`
	CountryCode string    `json:"country_code" bson:"country_code"`
	Name        string    `json:"name" bson:"name"`
	Description string    `json:"description" bson:"description"`
	CreatedAt   time.Time `json:"created_at" bson:"created_at"`
}

// Conversation represents a persistent 1-on-1 DM between two users.
type Conversation struct {
	ID                 string    `json:"id" bson:"_id"`
	Participants       []string  `json:"participants" bson:"participants"` // exactly 2 userIDs
	Status             string    `json:"status" bson:"status"`             // PENDING, ACCEPTED, REJECTED
	InitiatorID        string    `json:"initiator_id" bson:"initiator_id"` // User who sent the first message
	MsgCountInitiator  int       `json:"msg_count_initiator" bson:"msg_count_initiator"`
	LastMessageSnippet string    `json:"last_message_snippet" bson:"last_message_snippet"`
	CreatedAt          time.Time `json:"created_at" bson:"created_at"`
	UpdatedAt          time.Time `json:"updated_at" bson:"updated_at"`
}

type Message struct {
	ID             string     `json:"id" bson:"_id"`
	ConversationID *string    `json:"conversation_id,omitempty" bson:"conversation_id,omitempty"`
	RoomID         *string    `json:"room_id,omitempty" bson:"room_id,omitempty"`
	ChannelID      *string    `json:"channel_id,omitempty" bson:"channel_id,omitempty"`
	UserID         string     `json:"user_id" bson:"user_id"`
	Content        string     `json:"content" bson:"content"`
	Type           string     `json:"type" bson:"type"` // text, image, voice, reaction
	FileName       *string    `json:"file_name,omitempty" bson:"file_name,omitempty"`
	IsUnsent       bool       `json:"is_unsent" bson:"is_unsent"` // Instagram "Unsend" feature
	DeletedForUser []string   `json:"deleted_for_user" bson:"deleted_for_user"`
	Status         string     `json:"status" bson:"status"` // pending, sent, delivered, seen
	Reactions      []Reaction `json:"reactions,omitempty" bson:"reactions,omitempty"`
	VoiceMeta      *VoiceMeta `json:"voice_meta,omitempty" bson:"voice_meta,omitempty"`
	CreatedAt      time.Time  `json:"created_at" bson:"created_at"`
	SeenAt         *time.Time `json:"seen_at,omitempty" bson:"seen_at,omitempty"`
	EditedAt       *time.Time `json:"edited_at,omitempty" bson:"edited_at,omitempty"`

	// Hydrated fields (not stored in message document)
	Username    string `json:"username,omitempty" bson:"username,omitempty"`
	DisplayName string `json:"display_name,omitempty" bson:"display_name,omitempty"`
}

type Reaction struct {
	UserID string `json:"user_id" bson:"user_id"`
	Type   string `json:"type" bson:"type"` // e.g. "heart"
}

type VoiceMeta struct {
	URL      string  `json:"url" bson:"url"`
	Duration float64 `json:"duration" bson:"duration"` // in seconds
}

// GetOrCreateChannelByCountry fetches the channel for a country code, or creates it.
func GetOrCreateChannelByCountry(ctx context.Context, countryCode string) (*Channel, error) {
	c := &Channel{}
	err := db.Database.Collection("channels").FindOne(ctx, bson.M{"country_code": countryCode}).Decode(c)

	if err == nil {
		return c, nil
	}

	if err != mongo.ErrNoDocuments {
		return nil, err
	}

	// Create new one dynamically
	c.ID = uuid.New().String()
	c.CountryCode = countryCode
	c.Name = countryCode + " General"
	c.Description = "Public world chat for " + countryCode
	c.CreatedAt = time.Now()

	_, err = db.Database.Collection("channels").InsertOne(ctx, c)
	if err != nil {
		return nil, err
	}
	return c, nil
}

// SaveMessage inserts a new message into the database.
func SaveMessage(ctx context.Context, roomID, channelID *string, userID, content, msgType string) (*Message, error) {
	msg := &Message{
		ID:        uuid.New().String(),
		RoomID:    roomID,
		ChannelID: channelID,
		UserID:    userID,
		Content:   content,
		Type:      msgType,
		Status:    "sent",
		CreatedAt: time.Now(),
	}

	_, err := db.Database.Collection("messages").InsertOne(ctx, msg)
	if err != nil {
		return nil, err
	}
	return msg, nil
}

// SaveDMMessage inserts a new direct-message payload into a conversation.
func SaveDMMessage(ctx context.Context, conversationID string, userID, content, msgType string) (*Message, error) {
	msg := &Message{
		ID:             uuid.New().String(),
		ConversationID: &conversationID,
		UserID:         userID,
		Content:        content,
		Type:           msgType,
		Status:         "sent",
		CreatedAt:      time.Now(),
	}

	_, err := db.Database.Collection("messages").InsertOne(ctx, msg)
	if err != nil {
		return nil, err
	}
	return msg, nil
}

// GetMessagesByChannel fetches the latest messages for a given channel, joining user data.
func GetMessagesByChannel(ctx context.Context, channelID string, limit int) ([]Message, error) {
	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.D{{Key: "channel_id", Value: channelID}}}},
		{{Key: "$sort", Value: bson.D{{Key: "created_at", Value: -1}}}},
		{{Key: "$limit", Value: limit}},
		{{Key: "$lookup", Value: bson.D{
			{Key: "from", Value: "users"},
			{Key: "localField", Value: "user_id"},
			{Key: "foreignField", Value: "_id"},
			{Key: "as", Value: "user_info"},
		}}},
		{{Key: "$unwind", Value: bson.D{{Key: "path", Value: "$user_info"}, {Key: "preserveNullAndEmptyArrays", Value: true}}}},
		{{Key: "$project", Value: bson.M{
			"_id": 1, "room_id": 1, "channel_id": 1, "user_id": 1, "content": 1,
			"type": 1, "file_name": 1, "edited": 1, "deleted": 1, "status": 1,
			"created_at": 1, "edited_at": 1,
			"username":     "$user_info.username",
			"display_name": "$user_info.name",
		}}},
	}

	cursor, err := db.Database.Collection("messages").Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var messages []Message
	if err := cursor.All(ctx, &messages); err != nil {
		return nil, err
	}

	// Reverse to chronological order
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}

	return messages, nil
}

// GetMessagesByConversation fetches the latest messages for a 1-on-1 DM conversation.
// Results are returned in chronological order (oldest first), joining user data for display names.
func GetMessagesByConversation(ctx context.Context, conversationID string, limit int) ([]Message, error) {
	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.D{{Key: "conversation_id", Value: conversationID}}}},
		{{Key: "$sort", Value: bson.D{{Key: "created_at", Value: -1}}}},
		{{Key: "$limit", Value: limit}},
		{{Key: "$lookup", Value: bson.D{
			{Key: "from", Value: "users"},
			{Key: "localField", Value: "user_id"},
			{Key: "foreignField", Value: "_id"},
			{Key: "as", Value: "sender_info"},
		}}},
		{{Key: "$unwind", Value: bson.D{{Key: "path", Value: "$sender_info"}, {Key: "preserveNullAndEmptyArrays", Value: true}}}},
		{{Key: "$project", Value: bson.M{
			"_id": 1, "conversation_id": 1, "user_id": 1, "content": 1,
			"type": 1, "is_unsent": 1, "status": 1, "reactions": 1,
			"created_at": 1, "seen_at": 1, "edited_at": 1,
			"username":     "$sender_info.username",
			"display_name": "$sender_info.name",
		}}},
	}

	cursor, err := db.Database.Collection("messages").Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var conversationMessages []Message
	if err := cursor.All(ctx, &conversationMessages); err != nil {
		return nil, err
	}

	// Reverse so messages render oldest-first in the chat view
	for i, j := 0, len(conversationMessages)-1; i < j; i, j = i+1, j-1 {
		conversationMessages[i], conversationMessages[j] = conversationMessages[j], conversationMessages[i]
	}

	return conversationMessages, nil
}

// UpdateMessageStatus updates the delivery/read status of a message.
func UpdateMessageStatus(ctx context.Context, messageID, status string) error {
	filter := bson.M{"_id": messageID}
	update := bson.M{"$set": bson.M{"status": status}}
	_, err := db.Database.Collection("messages").UpdateOne(ctx, filter, update)
	return err
}

// UpdateMessage marks a message as edited and updates its content.
func UpdateMessage(ctx context.Context, messageID, userID, content string) (*Message, error) {
	filter := bson.M{
		"_id":     messageID,
		"user_id": userID,
		"deleted": false,
	}
	update := bson.M{
		"$set": bson.M{
			"content":   content,
			"edited":    true,
			"edited_at": time.Now(),
		},
	}
	opts := options.FindOneAndUpdate().SetReturnDocument(options.After)

	msg := &Message{}
	err := db.Database.Collection("messages").FindOneAndUpdate(ctx, filter, update, opts).Decode(msg)
	if err != nil {
		return nil, err
	}
	return msg, nil
}

// DeleteMessage soft-deletes a message.
func DeleteMessage(ctx context.Context, messageID, userID string) (*Message, error) {
	filter := bson.M{
		"_id":     messageID,
		"user_id": userID,
	}
	update := bson.M{
		"$set": bson.M{
			"deleted": true,
			"content": "🚫 This message was deleted",
		},
	}
	opts := options.FindOneAndUpdate().SetReturnDocument(options.After)

	msg := &Message{}
	err := db.Database.Collection("messages").FindOneAndUpdate(ctx, filter, update, opts).Decode(msg)
	if err != nil {
		return nil, err
	}
	return msg, nil
}

// GetConversationByParticipants searches for an existing 1-on-1 chat between two specific users.
func GetConversationByParticipants(ctx context.Context, participantA, participantB string) (*Conversation, error) {
	conversation := &Conversation{}

	// We sort the IDs to ensure the participants slice is always consistent (canonical form)
	sortedParticipants := []string{participantA, participantB}
	if participantA > participantB {
		sortedParticipants = []string{participantB, participantA}
	}

	filter := bson.M{"participants": sortedParticipants}
	err := db.Database.Collection("conversations").FindOne(ctx, filter).Decode(conversation)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil // No existing chat found
		}
		return nil, err
	}
	return conversation, nil
}

// GetConversationByID fetches one conversation document by ID.
func GetConversationByID(ctx context.Context, conversationID string) (*Conversation, error) {
	conv := &Conversation{}
	err := db.Database.Collection("conversations").FindOne(ctx, bson.M{"_id": conversationID}).Decode(conv)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, err
	}
	return conv, nil
}

// CreateConversation initializes a new 1-on-1 talk session.
// This is the 'Invite to Talk' phase.
func CreateConversation(ctx context.Context, initiatorID, receiverID string) (*Conversation, error) {
	// Canonical sorting for unique participant pairs
	sortedParticipants := []string{initiatorID, receiverID}
	if initiatorID > receiverID {
		sortedParticipants = []string{receiverID, initiatorID}
	}

	conversation := &Conversation{
		ID:                uuid.New().String(),
		Participants:      sortedParticipants,
		Status:            "PENDING", // Wait for receiver to accept
		InitiatorID:       initiatorID,
		MsgCountInitiator: 0,
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}

	_, err := db.Database.Collection("conversations").InsertOne(ctx, conversation)
	if err != nil {
		return nil, err
	}
	return conversation, nil
}

// ListUserConversations fetches all active DMs for a user, sorted by the most recent activity.
// This populates the 'Recent Chat' list similar to Instagram's DM tab.
func ListUserConversations(ctx context.Context, userID string) ([]Conversation, error) {
	filter := bson.M{"participants": userID}
	options := options.Find().SetSort(bson.D{{Key: "updated_at", Value: -1}})

	cursor, err := db.Database.Collection("conversations").Find(ctx, filter, options)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var conversations []Conversation
	if err := cursor.All(ctx, &conversations); err != nil {
		return nil, err
	}
	return conversations, nil
}

// UpdateConversationActivity updates the snippet and timestamp for a conversation.
// This moves the chat to the top of the 'Recent' list.
func UpdateConversationActivity(ctx context.Context, conversationID, lastSnippet string) error {
	filter := bson.M{"_id": conversationID}
	update := bson.M{
		"$set": bson.M{
			"last_message_snippet": lastSnippet,
			"updated_at":           time.Now(),
		},
	}
	result, err := db.Database.Collection("conversations").UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return errors.New("conversation not found")
	}
	return nil
}

// IncrementInitiatorMsgCount tracks the 2-message limit for pending requests.
func IncrementInitiatorMsgCount(ctx context.Context, conversationID string) error {
	filter := bson.M{"_id": conversationID}
	update := bson.M{"$inc": bson.M{"msg_count_initiator": 1}}
	result, err := db.Database.Collection("conversations").UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return errors.New("conversation not found")
	}
	return nil
}

// AcceptConversationRequest transition a chat from PENDING to ACCEPTED.
func AcceptConversationRequest(ctx context.Context, conversationID string) error {
	filter := bson.M{"_id": conversationID}
	update := bson.M{"$set": bson.M{"status": "ACCEPTED"}}
	result, err := db.Database.Collection("conversations").UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return errors.New("conversation not found")
	}
	return nil
}

// UnsendMessage implements the Instagram 'Unsend' behavior.
// It hides the message for everyone and replaces its content.
func UnsendMessage(ctx context.Context, messageID, userID string) error {
	filter := bson.M{
		"_id":     messageID,
		"user_id": userID,
	}
	update := bson.M{
		"$set": bson.M{
			"content":   "🚫 This message was un-sent",
			"is_unsent": true,
		},
	}
	_, err := db.Database.Collection("messages").UpdateOne(ctx, filter, update)
	return err
}

// MarkMessageAsSeen updates the 'seen_at' timestamp for read receipts.
func MarkMessageAsSeen(ctx context.Context, messageID string) error {
	filter := bson.M{
		"_id":     messageID,
		"seen_at": nil, // Only mark if not already seen
	}
	update := bson.M{
		"$set": bson.M{
			"seen_at": time.Now(),
			"status":  "seen",
		},
	}
	_, err := db.Database.Collection("messages").UpdateOne(ctx, filter, update)
	return err
}
