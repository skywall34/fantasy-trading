package main

import (
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	"github.com/skywall34/fantasy-trading/internal/database"
	"github.com/skywall34/fantasy-trading/internal/handlers"
	"github.com/skywall34/fantasy-trading/internal/middleware"
)

func main() {
	// Load .env file if it exists
	_ = godotenv.Load() // Ignore error if .env doesn't exist

	// Get environment variables
	port := getEnv("PORT", "8080")
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

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}
