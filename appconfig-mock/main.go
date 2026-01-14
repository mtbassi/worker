package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	defaultPort       = "2772"
	defaultConfigsDir = "/configs"
)

type Server struct {
	configsDir string
	logger     *slog.Logger
}

func NewServer(configsDir string, logger *slog.Logger) *Server {
	return &Server{
		configsDir: configsDir,
		logger:     logger,
	}
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	// Log request
	s.logger.Info("Incoming request",
		"method", r.Method,
		"path", r.URL.Path,
		"remote", r.RemoteAddr,
	)

	// Health check endpoint
	if r.URL.Path == "/health" {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"healthy"}`))
		s.logger.Info("Health check",
			"status", 200,
			"duration", time.Since(start),
		)
		return
	}

	// Only allow GET requests for config files
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		s.logger.Warn("Method not allowed",
			"method", r.Method,
			"path", r.URL.Path,
		)
		return
	}

	// Remove leading slash and get filename
	filename := strings.TrimPrefix(r.URL.Path, "/")
	if filename == "" {
		w.WriteHeader(http.StatusNotFound)
		s.logger.Warn("Empty filename", "path", r.URL.Path)
		return
	}

	// Security: prevent directory traversal
	if strings.Contains(filename, "..") {
		w.WriteHeader(http.StatusBadRequest)
		s.logger.Warn("Invalid filename - directory traversal attempt",
			"filename", filename,
		)
		return
	}

	// Construct full file path
	filePath := filepath.Join(s.configsDir, filename)

	// Read file
	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			w.WriteHeader(http.StatusNotFound)
			s.logger.Warn("Config file not found",
				"filename", filename,
				"path", filePath,
			)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
			s.logger.Error("Failed to read config file",
				"filename", filename,
				"error", err,
			)
		}
		return
	}

	// Determine content type based on file extension
	contentType := "application/octet-stream"
	if strings.HasSuffix(filename, ".yaml") || strings.HasSuffix(filename, ".yml") {
		contentType = "application/x-yaml"
	}

	// Send response
	w.Header().Set("Content-Type", contentType)
	w.WriteHeader(http.StatusOK)
	w.Write(data)

	s.logger.Info("Config file served",
		"filename", filename,
		"size", len(data),
		"duration", time.Since(start),
	)
}

func main() {
	// Setup logger
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	// Get configuration from environment
	port := os.Getenv("PORT")
	if port == "" {
		port = defaultPort
	}

	configsDir := os.Getenv("CONFIGS_DIR")
	if configsDir == "" {
		configsDir = defaultConfigsDir
	}

	// Verify configs directory exists
	if _, err := os.Stat(configsDir); os.IsNotExist(err) {
		logger.Error("Configs directory does not exist",
			"path", configsDir,
			"error", err,
		)
		os.Exit(1)
	}

	// Create server
	server := NewServer(configsDir, logger)

	// Start HTTP server
	addr := fmt.Sprintf(":%s", port)
	logger.Info("Starting AppConfig Mock Server",
		"port", port,
		"configs_dir", configsDir,
	)

	if err := http.ListenAndServe(addr, server); err != nil {
		logger.Error("Server failed", "error", err)
		os.Exit(1)
	}
}
