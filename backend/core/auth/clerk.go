package auth

import (
	"fmt"
	"os"

	"github.com/clerk/clerk-sdk-go/v2"
)

// InitClerk initializes the official Clerk SDK client using the CLERK_SECRET_KEY.
func InitClerk() error {
	secretKey := os.Getenv("CLERK_SECRET_KEY")
	if secretKey == "" {
		return fmt.Errorf("CLERK_SECRET_KEY is not set in environment")
	}

	clerk.SetKey(secretKey)
	fmt.Println("✅ Clerk SDK initialized successfully")
	return nil
}
