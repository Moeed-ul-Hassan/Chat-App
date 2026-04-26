package main

import (
	//Libraries
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	//Modules
	coreAuth "github.com/Moeed-ul-Hassan/chatapp/core/auth"
	"github.com/Moeed-ul-Hassan/chatapp/core/db"
	appMiddleware "github.com/Moeed-ul-Hassan/chatapp/core/middleware"
	coreRedis "github.com/Moeed-ul-Hassan/chatapp/core/redis"
	"github.com/Moeed-ul-Hassan/chatapp/core/utils"

	//features Modules
	"github.com/Moeed-ul-Hassan/chatapp/features/admin"
	"github.com/Moeed-ul-Hassan/chatapp/features/auth"
	"github.com/Moeed-ul-Hassan/chatapp/features/chat"
	"github.com/Moeed-ul-Hassan/chatapp/features/room"
	"github.com/Moeed-ul-Hassan/chatapp/features/story"
	"github.com/Moeed-ul-Hassan/chatapp/features/user"

	//Routers and Middlewares
	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/rs/cors"
	"go.mongodb.org/mongo-driver/mongo/readpref"

	_ "github.com/Moeed-ul-Hassan/chatapp/docs"
	httpSwagger "github.com/swaggo/http-swagger"
)

// @title Echo Messenger API
// @version 1.0
// @description This is the professional backend API for the Echo messaging platform.
// @termsOfService http://swagger.io/terms/

// @contact.name Moeed-ul-Hassan
// @contact.url http://www.moeed.me
// @contact.email moeed@example.com

// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html

// @host localhost:8001
// @BasePath /
// @securityDefinitions.apikey ApiKeyAuth
// @in header
// @name Authorization

func main() {
	portFlag := flag.String("port", "", "Port to run the server on (default: 8001)")
	flag.Parse()

	if err := utils.LoadDotEnv(".env"); err != nil {
		if err2 := utils.LoadDotEnv("../../.env"); err2 != nil {
			fmt.Println("warning: .env not found, using system environment")
		}
	}

	requiredVars := []string{"CLERK_SECRET_KEY", "MONGODB_URI"}
	for _, key := range requiredVars {
		if utils.GetEnv(key, "") == "" {
			fmt.Printf("required env var %s is missing\n", key)
			os.Exit(1)
		}
	}

	if err := coreAuth.InitClerk(); err != nil {
		fmt.Printf("clerk init failed: %v\n", err)
		os.Exit(1)
	}

	if err := db.Connect(); err != nil {
		fmt.Printf("db connect failed: %v\n", err)
		os.Exit(1)
	}

	if err := db.CreateIndexes(); err != nil {
		fmt.Printf("db index setup failed: %v\n", err)
		os.Exit(1)
	}

	if err := coreRedis.Connect(); err != nil {
		fmt.Printf("redis disabled: %v\n", err)
	}

	r := chi.NewRouter()

	r.Use(chiMiddleware.RequestID)
	r.Use(appMiddleware.AccessLogger)
	r.Use(chiMiddleware.Recoverer)
	r.Use(chiMiddleware.RealIP)
	r.Use(chiMiddleware.Timeout(60 * time.Second))
	r.Use(appMiddleware.SecurityHeaders)

	origin := utils.GetEnv("FRONTEND_URL", "http://localhost:3001")
	c := cors.New(cors.Options{
		AllowedOrigins:   []string{origin, "http://localhost:3000", "http://localhost:5173", "http://localhost:5174"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Authorization", "Content-Type", "X-Requested-With"},
		AllowCredentials: true,
		Debug:            false,
	})
	r.Use(c.Handler)

	r.With(appMiddleware.IPRateLimitPreferRedis(60)).Route("/api/auth", auth.RegisterRoutes)
	r.Route("/api/user", user.RegisterRoutes)
	r.Route("/api/admin", admin.RegisterRoutes)
	r.With(appMiddleware.IPRateLimitPreferRedis(30)).Route("/api/rooms", room.RegisterRoutes)
	r.Route("/api/stories", story.RegisterRoutes)

	chat.RegisterRoutes(r)
	r.With(appMiddleware.IPRateLimitPreferRedis(90)).Route("/api/chat", chat.RegisterAPIRoutes)
	r.Handle("/uploads/*", http.StripPrefix("/uploads/", http.FileServer(http.Dir("uploads"))))

	r.Get("/swagger/*", httpSwagger.Handler(
		httpSwagger.URL("/swagger/doc.json"),
	))

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"status":"online","service":"echo-backend","timestamp":"%s"}`, time.Now().Format(time.RFC3339))
	})
	r.Get("/metrics", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(appMiddleware.MetricsSnapshot())
	})
	r.Get("/ready", func(w http.ResponseWriter, r *http.Request) {
		type componentStatus struct {
			Name   string `json:"name"`
			Ready  bool   `json:"ready"`
			Reason string `json:"reason,omitempty"`
		}
		report := struct {
			Status     string            `json:"status"`
			Service    string            `json:"service"`
			Timestamp  string            `json:"timestamp"`
			Components []componentStatus `json:"components"`
		}{
			Status:    "ready",
			Service:   "echo-backend",
			Timestamp: time.Now().Format(time.RFC3339),
		}

		mongoReady := false
		mongoReason := ""
		if db.Client != nil {
			ctx, cancel := context.WithTimeout(r.Context(), 1200*time.Millisecond)
			defer cancel()
			if err := db.Client.Ping(ctx, readpref.Primary()); err == nil {
				mongoReady = true
			} else {
				mongoReason = err.Error()
			}
		} else {
			mongoReason = "client not initialized"
		}
		report.Components = append(report.Components, componentStatus{Name: "mongo", Ready: mongoReady, Reason: mongoReason})

		redisReady := false
		redisReason := ""
		if coreRedis.IsReady() && coreRedis.Client != nil {
			ctx, cancel := context.WithTimeout(r.Context(), 1200*time.Millisecond)
			defer cancel()
			if err := coreRedis.Client.Ping(ctx).Err(); err == nil {
				redisReady = true
			} else {
				redisReason = err.Error()
			}
		} else {
			redisReason = "optional component not connected"
		}
		report.Components = append(report.Components, componentStatus{Name: "redis", Ready: redisReady, Reason: redisReason})

		if !mongoReady {
			report.Status = "not_ready"
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusServiceUnavailable)
			_ = json.NewEncoder(w).Encode(report)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(report)
	})

	port := *portFlag
	if port == "" {
		port = utils.GetEnv("PORT", "8001")
	}

	fmt.Printf("server listening on :%s\n", port)
	fmt.Printf("swagger at http://localhost:%s/swagger/index.html\n", port)

	srv := &http.Server{
		Addr:         "0.0.0.0:" + port,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	srvErr := make(chan error, 1)
	go func() {
		srvErr <- srv.ListenAndServe()
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-stop:
		fmt.Printf("received signal %s, shutting down\n", sig.String())
	case err := <-srvErr:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			fmt.Printf("server crash: %v\n", err)
			os.Exit(1)
		}
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		fmt.Printf("shutdown failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("server stopped")
}
