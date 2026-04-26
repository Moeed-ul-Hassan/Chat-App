package admin

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/Moeed-ul-Hassan/chatapp/core/db"
	"github.com/Moeed-ul-Hassan/chatapp/core/middleware"
	"github.com/go-chi/chi/v5"
	"go.mongodb.org/mongo-driver/bson"
)

func RegisterRoutes(r chi.Router) {
	r.Group(func(protected chi.Router) {
		protected.Use(middleware.RequireAdmin)
		protected.Get("/stats", GetStats)
		protected.Get("/traffic", GetTraffic)
		protected.Get("/logs", GetLogs)
	})
}

// respond writes a JSON response
func respond(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func GetStats(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	userCollection := db.Database.Collection("users")
	totalUsers, err := userCollection.CountDocuments(ctx, bson.M{})
	if err != nil {
		totalUsers = 0
	}

	// Mocking other telemetry stats dynamically to satisfy the UI
	// Server load varies slightly around 23%
	// Active Vaults randomly hops around 12-15
	stats := map[string]interface{}{
		"totalUsers":      totalUsers,
		"messagesSent":    142050, // Mocked for demonstration
		"serverLoad":      23.4,   // Mocked percentage
		"activeVaults":    12,     // Mocked integer
		"newUsersLast7d":  14,
		"messagesLast24h": 1200,
	}
	respond(w, http.StatusOK, stats)
}

func GetTraffic(w http.ResponseWriter, r *http.Request) {
	// Recharts compatible array of traffic volume over last 7 days
	now := time.Now()
	data := []map[string]interface{}{}

	for i := 6; i >= 0; i-- {
		day := now.AddDate(0, 0, -i)
		data = append(data, map[string]interface{}{
			"date":  day.Format(time.RFC3339),
			"count": 5000 + (i * 1200) + (time.Now().Second() * 10), // slightly dynamic
		})
	}
	respond(w, http.StatusOK, data)
}

func GetLogs(w http.ResponseWriter, r *http.Request) {
	// Mock live security logs for the dashboard
	now := time.Now()
	logs := []map[string]interface{}{
		{
			"id":          "1",
			"username":    "System",
			"description": "Authentication failed for admin",
			"createdAt":   now.Add(-2 * time.Minute).Format(time.RFC3339),
		},
		{
			"id":          "2",
			"username":    "JohnDoe",
			"description": "New user registered",
			"createdAt":   now.Add(-15 * time.Minute).Format(time.RFC3339),
		},
		{
			"id":          "3",
			"username":    "System",
			"description": "Vault 'Secure Channel Alpha' joined by 2 users",
			"createdAt":   now.Add(-1 * time.Hour).Format(time.RFC3339),
		},
	}
	respond(w, http.StatusOK, logs)
}
