package main

import (
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/joho/godotenv"
	"github.com/skywall34/fantasy-trading/internal/database"
	"github.com/skywall34/fantasy-trading/internal/handlers"
	"github.com/skywall34/fantasy-trading/internal/middleware"
	"github.com/skywall34/fantasy-trading/internal/sync"
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

	// Start background sync scheduler
	syncInterval := getEnvInt("SYNC_INTERVAL_MINUTES", 5) // Default 5 minutes
	scheduler := sync.NewScheduler(db, syncInterval)
	go scheduler.Start()
	log.Printf("Background sync started (interval: %d minutes)", syncInterval)

	// Create handlers
	loginHandler := handlers.NewAPIKeyLoginHandler(db)
	dashboardHandler := handlers.NewDashboardHandler(db)
	dashboardContentHandler := handlers.NewDashboardContentHandler(db)
	portfolioHistoryHandler := handlers.NewPortfolioHistoryHandler()
	updateProfileHandler := handlers.NewUpdateProfileHandler(db)
	settingsHandler := handlers.NewSettingsHandler(db)

	// Create router
	mux := http.NewServeMux()

	// Public routes
	mux.Handle("/login", loginHandler)

	// Static files
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./static"))))

	// Protected routes
	mux.Handle("/dashboard", middleware.AuthMiddleware(db)(dashboardHandler))
	mux.Handle("/dashboard/content", middleware.AuthMiddleware(db)(dashboardContentHandler))
	mux.Handle("/settings", middleware.AuthMiddleware(db)(settingsHandler))
	mux.Handle("/api/portfolio/history", middleware.AuthMiddleware(db)(portfolioHistoryHandler))
	mux.Handle("/api/profile/update", middleware.AuthMiddleware(db)(updateProfileHandler))
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

func getEnvInt(key string, defaultValue int) int {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	intValue, err := strconv.Atoi(value)
	if err != nil {
		return defaultValue
	}
	return intValue
}
