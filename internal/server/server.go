package server

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
)

// Server wraps the HTTP server with graceful shutdown capabilities
type Server struct {
	httpServer *http.Server
	db         *sql.DB
	router     *gin.Engine
	port       string
}

// Config holds server configuration
type Config struct {
	Port            string
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	ShutdownTimeout time.Duration
}

// New creates a new server instance
func New(router *gin.Engine, db *sql.DB, cfg Config) *Server {
	// Set default timeouts if not provided
	if cfg.ReadTimeout == 0 {
		cfg.ReadTimeout = 10 * time.Second
	}
	if cfg.WriteTimeout == 0 {
		cfg.WriteTimeout = 10 * time.Second
	}
	if cfg.ShutdownTimeout == 0 {
		cfg.ShutdownTimeout = 30 * time.Second
	}

	httpServer := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      router,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
	}

	return &Server{
		httpServer: httpServer,
		db:         db,
		router:     router,
		port:       cfg.Port,
	}
}

// Start starts the server with graceful shutdown handling
func (s *Server) Start() error {
	// Channel to listen for interrupt/terminate signals
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM, syscall.SIGUSR1)

	// Channel to listen for server errors
	serverErrors := make(chan error, 1)

	// Start server in a goroutine
	go func() {
		log.Printf("Starting HTTP server on port %s", s.port)
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			serverErrors <- fmt.Errorf("server failed to start: %w", err)
		}
	}()

	// Block until we receive a signal or server error
	select {
	case err := <-serverErrors:
		return err
	case sig := <-quit:
		log.Printf("Received signal: %s", sig)

		// Handle different signals
		switch sig {
		case syscall.SIGUSR1:
			log.Println("Received SIGUSR1 - initiating graceful restart")
			return s.gracefulRestart()
		default:
			log.Println("Initiating graceful shutdown")
			return s.gracefulShutdown()
		}
	}
}

// gracefulShutdown performs a graceful shutdown of the server
func (s *Server) gracefulShutdown() error {
	log.Println("Starting graceful shutdown...")

	// Create a context with timeout for shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Shutdown the HTTP server
	log.Println("Shutting down HTTP server...")
	if err := s.httpServer.Shutdown(ctx); err != nil {
		log.Printf("HTTP server shutdown error: %v", err)
		// Force close if graceful shutdown fails
		if closeErr := s.httpServer.Close(); closeErr != nil {
			log.Printf("HTTP server force close error: %v", closeErr)
		}
	} else {
		log.Println("HTTP server shutdown completed")
	}

	// Close database connections
	if s.db != nil {
		log.Println("Closing database connections...")
		if err := s.db.Close(); err != nil {
			log.Printf("Database close error: %v", err)
		} else {
			log.Println("Database connections closed")
		}
	}

	log.Println("Graceful shutdown completed")
	return nil
}

// gracefulRestart performs a graceful restart
func (s *Server) gracefulRestart() error {
	log.Println("Starting graceful restart...")

	// Perform graceful shutdown first
	if err := s.gracefulShutdown(); err != nil {
		log.Printf("Error during shutdown phase of restart: %v", err)
	}

	// In a real-world scenario, you might want to:
	// 1. Reload configuration
	// 2. Reconnect to database
	// 3. Restart the server process

	log.Println("Restart signal processed. Application should be restarted by process manager.")
	return nil
}

// Stop stops the server gracefully
func (s *Server) Stop() error {
	return s.gracefulShutdown()
}

// Health returns the health status of the server
func (s *Server) Health() error {
	// Check database connection
	if s.db != nil {
		if err := s.db.Ping(); err != nil {
			return fmt.Errorf("database health check failed: %w", err)
		}
	}
	return nil
}

// AddShutdownHook allows adding custom cleanup functions
func (s *Server) AddShutdownHook(hook func()) {
	// Register the shutdown handler
	s.httpServer.RegisterOnShutdown(hook)
}
