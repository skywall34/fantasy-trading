package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
	"github.com/skywall34/fantasy-trading/internal/alpaca"
	"github.com/skywall34/fantasy-trading/internal/cache"
	"github.com/skywall34/fantasy-trading/internal/database"
	"github.com/skywall34/fantasy-trading/internal/handlers"
	"github.com/skywall34/fantasy-trading/internal/middleware"
)

// Global cache instance
var alpacaCache *cache.Cache

func main() {
	// Load .env file if it exists
	_ = godotenv.Load() // Ignore error if .env doesn't exist

	// Get environment variables
	port := getEnv("PORT", "8082")
	dbPath := getEnv("DATABASE_PATH", "./data/database.db")

	// Initialize encryption
	if err := database.InitEncryption(); err != nil {
		log.Fatalf("Failed to initialize encryption: %v", err)
	}

	// Initialize database
	db, err := database.New(dbPath)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	log.Println("Database initialized successfully")

	// Initialize cache
	cacheEnabled := getEnv("CACHE_ENABLED", "true") == "true"
	if cacheEnabled {
		accountTTL := getEnvInt("CACHE_ACCOUNT_TTL_SECONDS", 60)
		refreshBuffer := getEnvInt("CACHE_REFRESH_BUFFER_SECONDS", 15)

		alpacaCache = cache.NewCache(
			time.Duration(accountTTL)*time.Second,
			time.Duration(refreshBuffer)*time.Second,
		)
		defer alpacaCache.Stop()

		log.Printf("Cache enabled - TTL: %ds, Refresh Buffer: %ds", accountTTL, refreshBuffer)

		// Warm cache on startup
		go warmCache(db, alpacaCache)
	} else {
		log.Println("Cache disabled")
	}

	// Create handlers
	loginHandler := handlers.NewAPIKeyLoginHandler(db)
	dashboardHandler := handlers.NewDashboardHandler(db)
	dashboardContentHandler := handlers.NewDashboardContentHandler(db)
	portfolioHistoryHandler := handlers.NewPortfolioHistoryHandler()
	updateProfileHandler := handlers.NewUpdateProfileHandler(db)
	settingsHandler := handlers.NewSettingsHandler(db)
	leaderboardHandler := handlers.NewLeaderboardHandler(db)
	activityHandler := handlers.NewActivityHandler(db)
	commentsHandler := handlers.NewCommentsHandler(db)
	commentActionsHandler := handlers.NewCommentActionsHandler(db)
	reactionsHandler := handlers.NewReactionsHandler(db)
	followHandler := handlers.NewFollowHandler(db)
	searchHandler := handlers.NewSearchHandler(db)
	userHandler := handlers.NewUserHandler(db)
	logoutHandler := handlers.NewLogoutHandler(db)

	// Set cache on handlers if enabled
	if alpacaCache != nil {
		leaderboardHandler.SetCache(alpacaCache)
		activityHandler.SetCache(alpacaCache)
		userHandler.SetCache(alpacaCache)
		logoutHandler.SetCache(alpacaCache)
	}

	// Create router
	mux := http.NewServeMux()

	// Public routes
	mux.Handle("/login", loginHandler)
	mux.Handle("/logout", logoutHandler)

	// Static files
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./static"))))

	// Protected routes
	mux.Handle("/dashboard", middleware.AuthMiddleware(db)(dashboardHandler))
	mux.Handle("/dashboard/content", middleware.AuthMiddleware(db)(dashboardContentHandler))
	mux.Handle("/leaderboard", middleware.AuthMiddleware(db)(leaderboardHandler))
	mux.Handle("/activity", middleware.AuthMiddleware(db)(activityHandler))
	mux.Handle("/search", middleware.AuthMiddleware(db)(searchHandler))
	mux.Handle("/user/", middleware.AuthMiddleware(db)(userHandler))
	mux.Handle("/settings", middleware.AuthMiddleware(db)(settingsHandler))
	mux.Handle("/api/portfolio/history", middleware.AuthMiddleware(db)(portfolioHistoryHandler))
	mux.Handle("/api/profile/update", middleware.AuthMiddleware(db)(updateProfileHandler))
	mux.Handle("/api/activities/", middleware.AuthMiddleware(db)(http.StripPrefix("/api/activities/", commentsHandler)))
	mux.Handle("/api/comments/", middleware.AuthMiddleware(db)(http.StripPrefix("/api/comments/", commentActionsHandler)))
	mux.Handle("/api/reactions/", middleware.AuthMiddleware(db)(http.StripPrefix("/api/reactions/", reactionsHandler)))
	mux.Handle("/api/follow", middleware.AuthMiddleware(db)(followHandler))
	mux.Handle("/", http.RedirectHandler("/dashboard", http.StatusTemporaryRedirect))

	// Cache stats endpoint (admin/monitoring)
	if alpacaCache != nil {
		mux.HandleFunc("/admin/cache/stats", func(w http.ResponseWriter, r *http.Request) {
			stats := alpacaCache.GetStats()
			total := stats.Hits + stats.Misses
			hitRate := 0.0
			if total > 0 {
				hitRate = float64(stats.Hits) / float64(total) * 100
			}

			w.Header().Set("Content-Type", "text/plain")
			fmt.Fprintf(w, "Cache Statistics\n")
			fmt.Fprintf(w, "================\n\n")
			fmt.Fprintf(w, "Total Requests: %d\n", total)
			fmt.Fprintf(w, "Cache Hits:     %d\n", stats.Hits)
			fmt.Fprintf(w, "Cache Misses:   %d\n", stats.Misses)
			fmt.Fprintf(w, "Hit Rate:       %.2f%%\n", hitRate)
			fmt.Fprintf(w, "Refreshes:      %d\n", stats.Refreshes)
		})
	}

	// Wrap with middleware
	handler := middleware.LoggingMiddleware(
		middleware.CSPMiddleware(
			mux,
		),
	)

	// Start server
	addr := ":" + port
	log.Printf("Server starting on %s", addr)
	if err := http.ListenAndServe(addr, handler); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}

// warmCache pre-populates cache with all public users' data
func warmCache(db *database.DB, cache *cache.Cache) {
	log.Println("Warming cache for all public users...")

	users, err := db.GetAllPublicUsers()
	if err != nil {
		log.Printf("Failed to warm cache: %v", err)
		return
	}

	successCount := 0
	for i, user := range users {
		session, err := db.GetLatestSession(user.ID)
		if err != nil {
			continue
		}

		apiKey, apiSecret, err := database.DecryptAPIKeys(session.APIKey, session.APISecret)
		if err != nil {
			continue
		}

		client := alpaca.NewClient(apiKey, apiSecret)
		ctx := context.Background()

		// Cache account data with refresh function
		accountCacheKey := fmt.Sprintf("account:%d", user.ID)
		accountRefreshFunc := func(ctx context.Context) (any, error) {
			return client.GetAccount(ctx)
		}

		if account, err := client.GetAccount(ctx); err == nil {
			activitiesTTL := getEnvInt("CACHE_ACTIVITIES_TTL_SECONDS", 30)
			cache.SetWithRefresh(accountCacheKey, account, 60*time.Second, accountRefreshFunc)

			// Cache activities with refresh function
			activitiesCacheKey := fmt.Sprintf("activities:%d", user.ID)
			activitiesRefreshFunc := func(ctx context.Context) (any, error) {
				return client.GetActivities(ctx)
			}

			if activities, err := client.GetActivities(ctx); err == nil {
				cache.SetWithRefresh(activitiesCacheKey, activities, time.Duration(activitiesTTL)*time.Second, activitiesRefreshFunc)
			}

			successCount++
		}

		// Rate limit protection - spread requests over time
		if i < len(users)-1 {
			time.Sleep(100 * time.Millisecond)
		}
	}

	log.Printf("Cache warming complete for %d/%d users", successCount, len(users))
}

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

func getEnvInt(key string, defaultValue int) int {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	intVal, err := strconv.Atoi(value)
	if err != nil {
		return defaultValue
	}
	return intVal
}
