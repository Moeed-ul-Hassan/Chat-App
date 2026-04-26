package auth

import (
	"fmt"
	"os"

	"github.com/gorilla/sessions"
	"github.com/markbates/goth"
	"github.com/markbates/goth/gothic"
	"github.com/markbates/goth/providers/google"
)

func InitGoth() {
	key := os.Getenv("GOOGLE_KEY")
	secret := os.Getenv("GOOGLE_SECRET")
	callback := "http://localhost:8001/api/auth/google/callback" // This should be synced with frontend URL in prod

	if key == "" || secret == "" {
		fmt.Println("⚠️  Warning: GOOGLE_KEY or GOOGLE_SECRET is not set. Google Login will fail!")
		fmt.Println("   Ensure .env is correctly loaded and keys are present.")
	}

	goth.UseProviders(
		google.New(key, secret, callback, "email", "profile"),
	)

	// Configure Gothic's session store (used for OAuth state)
	sessionSecret := os.Getenv("SESSION_SECRET")
	if sessionSecret == "" {
		sessionSecret = "a-very-secret-key-change-this-in-prod"
	}

	store := sessions.NewCookieStore([]byte(sessionSecret))
	store.MaxAge(86400 * 30) // 30 days
	store.Options.Path = "/"
	store.Options.HttpOnly = true
	store.Options.Secure = os.Getenv("GO_ENV") == "production"

	gothic.Store = store
}
