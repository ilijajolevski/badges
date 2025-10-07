package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/finki/badges/internal/apikey"
	"github.com/finki/badges/internal/auth"
	"github.com/finki/badges/internal/badge"
	"github.com/finki/badges/internal/cache"
	"github.com/finki/badges/internal/certificate"
	"github.com/finki/badges/internal/config"
	"github.com/finki/badges/internal/database"
	"github.com/finki/badges/internal/details"
	"github.com/finki/badges/internal/home"
	"github.com/finki/badges/internal/list"
	"github.com/finki/badges/internal/middleware"
	"go.uber.org/zap"
)

func main() {
	// Initialize configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize logger
	logger, err := initLogger(cfg.LogLevel)
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	defer logger.Sync()

	// Replace the global logger
	zap.ReplaceGlobals(logger)

	// Initialize database
	db, err := database.New(cfg.DatabasePath, logger)
	if err != nil {
		logger.Fatal("Failed to initialize database", zap.Error(err))
	}
	defer db.Close()

	// Initialize cache
	imageCache := cache.New()

	// Initialize middleware
	errorHandler, err := middleware.NewErrorHandler(logger)
	if err != nil {
		logger.Fatal("Failed to initialize error handler", zap.Error(err))
	}

	sanitizer := middleware.NewSanitizer(logger)
	rateLimiter := middleware.NewRateLimiter(logger, 100, time.Minute) // 100 requests per minute
	requestLogger := middleware.NewRequestLogger(logger)

	// Initialize handlers
	badgeHandler := badge.NewHandler(db, logger, imageCache)
	certificateHandler := certificate.NewHandler(db, logger, imageCache)

	detailsHandler, err := details.NewHandler(db, logger, imageCache)
	if err != nil {
		logger.Fatal("Failed to initialize details handler", zap.Error(err))
	}

	listHandler, err := list.NewHandler(db, logger, imageCache)
	if err != nil {
		logger.Fatal("Failed to initialize list handler", zap.Error(err))
	}

	// Initialize home handler
	homeHandler, err := home.NewHandler(db, logger, imageCache)
	if err != nil {
		logger.Fatal("Failed to initialize home handler", zap.Error(err))
	}
	
	// Initialize API key handler
	apiKeyHandler := apikey.NewHandler(db, logger)
	
	// Initialize auth handler
	authHandler := auth.NewHandler(db, logger)

	// Initialize HTTP server
	mux := http.NewServeMux()

	// Register routes
	registerRoutes(mux, badgeHandler, certificateHandler, detailsHandler, listHandler, homeHandler, apiKeyHandler, authHandler, errorHandler, sanitizer, rateLimiter, requestLogger)

	// Create HTTP server
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Port),
		Handler:      mux,
		ReadTimeout:  time.Second * 15,
		WriteTimeout: time.Second * 15,
		IdleTimeout:  time.Second * 60,
	}

	// List available badges
	badges, err := db.ListBadges()
	if err != nil {
		logger.Warn("Failed to list badges", zap.Error(err))
	} else {
		logger.Info("Available badges for serving", zap.Int("count", len(badges)))
		for _, badge := range badges {
			logger.Info("Badge", 
				zap.String("commit_id", badge.CommitID), 
				zap.String("type", badge.Type), 
				zap.String("status", badge.Status),
				zap.String("software", badge.SoftwareName + " " + badge.SoftwareVersion))
		}
	}

	// Start HTTP server in a goroutine
	go func() {
		logger.Info("Starting server", zap.Int("port", cfg.Port))
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Failed to start server", zap.Error(err))
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")

	// Create a deadline to wait for
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*15)
	defer cancel()

	// Doesn't block if no connections, but will otherwise wait
	// until the timeout deadline
	if err := server.Shutdown(ctx); err != nil {
		logger.Fatal("Server forced to shutdown", zap.Error(err))
	}

	logger.Info("Server exited properly")
}

func initLogger(level string) (*zap.Logger, error) {
	var cfg zap.Config

	if level == "production" {
		cfg = zap.NewProductionConfig()
	} else {
		cfg = zap.NewDevelopmentConfig()
	}

	return cfg.Build()
}

func registerRoutes(
	mux *http.ServeMux,
	badgeHandler *badge.Handler,
	certificateHandler *certificate.Handler,
	detailsHandler *details.Handler,
	listHandler *list.Handler,
	homeHandler *home.Handler,
	apiKeyHandler *apikey.Handler,
	authHandler *auth.Handler,
	errorHandler *middleware.ErrorHandler,
	sanitizer *middleware.Sanitizer,
	rateLimiter *middleware.RateLimiter,
	requestLogger *middleware.RequestLogger,
) {
	// Apply middleware to handlers
	badgeHandlerWithMiddleware := requestLogger.Middleware(
		errorHandler.Middleware(
			rateLimiter.Middleware(
				sanitizer.Middleware(badgeHandler),
			),
		),
	)

	certificateHandlerWithMiddleware := requestLogger.Middleware(
		errorHandler.Middleware(
			rateLimiter.Middleware(
				sanitizer.Middleware(certificateHandler),
			),
		),
	)

	detailsHandlerWithMiddleware := requestLogger.Middleware(
		errorHandler.Middleware(
			rateLimiter.Middleware(
				sanitizer.Middleware(detailsHandler),
			),
		),
	)

	listHandlerWithMiddleware := requestLogger.Middleware(
		errorHandler.Middleware(
			rateLimiter.Middleware(
				sanitizer.Middleware(listHandler),
			),
		),
	)

	homeHandlerWithMiddleware := requestLogger.Middleware(
		errorHandler.Middleware(
			rateLimiter.Middleware(
				sanitizer.Middleware(homeHandler),
			),
		),
	)

	// Create a handler function for the ListAPIKeys endpoint
	listAPIKeysHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Only handle GET requests
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		
		// Call the ListAPIKeys method
		apiKeyHandler.ListAPIKeys(w, r)
	})
	
	// Apply middleware to the ListAPIKeys handler
	listAPIKeysHandlerWithMiddleware := requestLogger.Middleware(
		errorHandler.Middleware(
			rateLimiter.Middleware(
				sanitizer.Middleware(
					auth.JWTAuthMiddleware(listAPIKeysHandler),
				),
			),
		),
	)

	// Create a handler function for the login endpoint
	loginHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHandler.Login(w, r)
	})
	
	// Apply middleware to the login handler
	loginHandlerWithMiddleware := requestLogger.Middleware(
		errorHandler.Middleware(
			rateLimiter.Middleware(
				sanitizer.Middleware(loginHandler),
			),
		),
	)

	// Register routes
	mux.Handle("/badge/", badgeHandlerWithMiddleware)
	mux.Handle("/certificate/", certificateHandlerWithMiddleware)
	mux.Handle("/details/", detailsHandlerWithMiddleware)
	mux.Handle("/badges", listHandlerWithMiddleware)
	mux.Handle("/api/keys", listAPIKeysHandlerWithMiddleware)
	mux.Handle("/api/auth/login", loginHandlerWithMiddleware)
	mux.Handle("/", homeHandlerWithMiddleware)

	// Serve static files
	fs := http.FileServer(http.Dir("./static"))
	mux.Handle("/static/", http.StripPrefix("/static/", fs))
}
