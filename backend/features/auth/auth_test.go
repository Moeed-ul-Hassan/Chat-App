// Package auth handles the authentication logic and tests.
// This test shows how to mock 'CheckOTP' and other complex operations.
package auth

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// ── MOCK AUTH STORE ─────────────────────────────────────────────────────────

type MockAuthStore struct {
	mock.Mock
}

func (m *MockAuthStore) CreateSession(ctx context.Context, userID, rawToken, deviceInfo, ipAddress string) error {
	return m.Called(ctx, userID, rawToken, deviceInfo, ipAddress).Error(0)
}
func (m *MockAuthStore) GetSessionByToken(ctx context.Context, rawToken string) (*Session, error) {
	args := m.Called(ctx, rawToken)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*Session), args.Error(1)
}
func (m *MockAuthStore) DeleteSession(ctx context.Context, rawToken string) error {
	return m.Called(ctx, rawToken).Error(0)
}
func (m *MockAuthStore) DeleteAllUserSessions(ctx context.Context, userID string) error {
	return m.Called(ctx, userID).Error(0)
}
func (m *MockAuthStore) ListUserSessions(ctx context.Context, userID string) ([]Session, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).([]Session), args.Error(1)
}
func (m *MockAuthStore) SaveOTP(ctx context.Context, email, code, purpose string) error {
	return m.Called(ctx, email, code, purpose).Error(0)
}
func (m *MockAuthStore) CheckOTP(ctx context.Context, email, code, purpose string) (bool, error) {
	args := m.Called(ctx, email, code, purpose)
	return args.Bool(0), args.Error(1)
}

// ── TESTS ──────────────────────────────────────────────────────────────────

// TestCheckOTP_Valid checks if we return true when the OTP matches.
func TestCheckOTP_Valid(t *testing.T) {
	mockRepo := new(MockAuthStore)
	ctx := context.Background()

	// 1. Arrange: EXPECT the code "123456" to be valid for email "test@echo.com"
	mockRepo.On("CheckOTP", ctx, "test@echo.com", "123456", "verify").Return(true, nil)

	// 2. Act
	isValid, err := mockRepo.CheckOTP(ctx, "test@echo.com", "123456", "verify")

	// 3. Assert
	assert.NoError(t, err)
	assert.True(t, isValid)
	mockRepo.AssertExpectations(t)
}

// TestCheckOTP_Invalid checks if we return false when the code is wrong or expired.
func TestCheckOTP_Invalid(t *testing.T) {
	mockRepo := new(MockAuthStore)
	ctx := context.Background()

	// 1. Arrange: The database says the code is NOT found or EXPIRED (returning false, nil)
	mockRepo.On("CheckOTP", ctx, mock.Anything, mock.Anything, mock.Anything).Return(false, nil)

	// 2. Act
	isValid, _ := mockRepo.CheckOTP(ctx, "someone@else.com", "000000", "verify")

	// 3. Assert
	assert.False(t, isValid)
}

// TestCheckOTP_DatabaseError checks if we pass along the error message.
func TestCheckOTP_DatabaseError(t *testing.T) {
	mockRepo := new(MockAuthStore)
	ctx := context.Background()

	// 1. Arrange: The database connection fails
	mockRepo.On("CheckOTP", ctx, mock.Anything, mock.Anything, mock.Anything).Return(false, errors.New("timeout"))

	// 2. Act
	_, err := mockRepo.CheckOTP(ctx, "test@echo.com", "123456", "verify")

	// 3. Assert
	assert.Error(t, err)
	assert.Equal(t, "timeout", err.Error())
}
