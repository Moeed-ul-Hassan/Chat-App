package authz

import (
	"net/http"
	"sync"
	"time"

	"github.com/Moeed-ul-Hassan/chatapp/core/middleware"
	"github.com/Moeed-ul-Hassan/chatapp/features/user"
)

type cachedIdentity struct {
	userID    string
	expiresAt time.Time
}

var (
	identityCacheTTL = 2 * time.Minute
	identityCache    sync.Map // map[clerkSubject]cachedIdentity
)

// ResolveLocalUserID maps authenticated Clerk subject -> local Echo user ID with a short TTL cache.
func ResolveLocalUserID(r *http.Request) (string, bool) {
	claims := middleware.GetUserClaims(r)
	if claims == nil || claims.Subject == "" {
		return "", false
	}

	if raw, ok := identityCache.Load(claims.Subject); ok {
		if item, ok := raw.(cachedIdentity); ok && time.Now().Before(item.expiresAt) {
			return item.userID, true
		}
		identityCache.Delete(claims.Subject)
	}

	userRepo := user.NewUserRepository()
	currentUser, err := userRepo.GetUserByClerkID(r.Context(), claims.Subject)
	if err != nil || currentUser == nil || currentUser.ID == "" {
		return "", false
	}
	identityCache.Store(claims.Subject, cachedIdentity{
		userID:    currentUser.ID,
		expiresAt: time.Now().Add(identityCacheTTL),
	})
	return currentUser.ID, true
}
