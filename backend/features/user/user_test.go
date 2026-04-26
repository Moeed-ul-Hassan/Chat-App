// Package user_test handles unit testing for the user feature set.
// We use 'testify/mock' here to simulate database interactions without needing a real MongoDB.
package user

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// ── MOCK SYSTEM (The "Fake Store") ──────────────────────────────────────────

// MockUserStore is a fake implementation of the UserStore interface.
// During tests, we tell this mock exactly what to return when it is called.
type MockUserStore struct {
	mock.Mock
}

// We implement EVERY method in the interface so it qualifies as a UserStore.
// In Go, interfaces are satisfied implicitly, making this very easy.

func (m *MockUserStore) CreateUser(ctx context.Context, username, email, password string) (*User, error) {
	args := m.Called(ctx, username, email, password)
	// Return the index[0] as *User and index[1] as error
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*User), args.Error(1)
}

// Other mock methods (minimal implementations for our current test)
func (m *MockUserStore) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*User), args.Error(1)
}
func (m *MockUserStore) GetUserByID(ctx context.Context, id string) (*User, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*User), args.Error(1)
}
func (m *MockUserStore) GetUserByUsername(ctx context.Context, username string) (*User, error) {
	args := m.Called(ctx, username)
	return args.Get(0).(*User), args.Error(1)
}
func (m *MockUserStore) UpdatePassword(ctx context.Context, email, pass string) (string, error) {
	args := m.Called(ctx, email, pass)
	return args.String(0), args.Error(1)
}
func (m *MockUserStore) VerifyUser(ctx context.Context, email string) error {
	return m.Called(ctx, email).Error(0)
}
func (m *MockUserStore) UpdateLastSeen(ctx context.Context, id string) error {
	return m.Called(ctx, id).Error(0)
}
func (m *MockUserStore) UpdateTwoFactorSecret(ctx context.Context, id, secret string) error {
	return m.Called(ctx, id, secret).Error(0)
}
func (m *MockUserStore) EnableTwoFactorAuth(ctx context.Context, id string) error {
	return m.Called(ctx, id).Error(0)
}

// ── THE TUTORIAL TEST ──────────────────────────────────────────────────────

// TestCreateUser is our first "Go Love" tutorial test.
// It checks if our CreateUser function correctly handles a database success.
func TestCreateUser_Success(t *testing.T) {
	// 1. ARRANGE (Set up the world)
	mockRepo := new(MockUserStore)
	ctx := context.Background()

	testUser := &User{
		Username: "moeed_dev",
		Email:    "test@echo.com",
	}

	// Tell the mock: "When CreateUser is called with these exact strings,
	// RETURN our testUser and NO error."
	mockRepo.On("CreateUser", ctx, "moeed_dev", "test@echo.com", "secureHash").Return(testUser, nil)

	// 2. ACT (Run the code under test)
	// Even though we're testing the interface, this demonstrates how we would
	// call it in a real handler.
	createdUser, err := mockRepo.CreateUser(ctx, "moeed_dev", "test@echo.com", "secureHash")

	// 3. ASSERT (Verify the results)
	// We use the 'testify/assert' package for clean, readable checks.
	assert.NoError(t, err)                             // Expect no error
	assert.NotNil(t, createdUser)                      // Expect a user object back
	assert.Equal(t, "moeed_dev", createdUser.Username) // Check if name matches

	// Ensure that the mock was actually called as expected.
	mockRepo.AssertExpectations(t)
}

// TestCreateUser_DatabaseFailure checks if we correctly propagate errors.
func TestCreateUser_DatabaseFailure(t *testing.T) {
	mockRepo := new(MockUserStore)
	ctx := context.Background()

	// Tell the mock to FAIL with a specific error
	dbError := errors.New("mongodb connection lost")
	mockRepo.On("CreateUser", ctx, mock.Anything, mock.Anything, mock.Anything).Return(nil, dbError)

	// ACT
	_, err := mockRepo.CreateUser(ctx, "any", "any@test.com", "anypass")

	// ASSERT
	assert.Error(t, err)
	assert.Equal(t, "mongodb connection lost", err.Error())
}
