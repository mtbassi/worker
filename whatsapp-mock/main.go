package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/google/uuid"
)

const (
	defaultPort = "8081"
)

type Server struct {
	logger *slog.Logger
}

type WhatsAppMessageRequest struct {
	MessagingProduct string                 `json:"messaging_product"`
	RecipientType    string                 `json:"recipient_type,omitempty"`
	To               string                 `json:"to"`
	Type             string                 `json:"type"`
	Text             map[string]interface{} `json:"text,omitempty"`
	Template         map[string]interface{} `json:"template,omitempty"`
}

type WhatsAppMessageResponse struct {
	MessagingProduct string `json:"messaging_product"`
	Contacts         []struct {
		Input string `json:"input"`
		WaID  string `json:"wa_id"`
	} `json:"contacts"`
	Messages []struct {
		ID string `json:"id"`
	} `json:"messages"`
}

func NewServer(logger *slog.Logger) *Server {
	return &Server{
		logger: logger,
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

	// Only allow POST requests for message endpoint
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		s.logger.Warn("Method not allowed",
			"method", r.Method,
			"path", r.URL.Path,
		)
		return
	}

	// Read request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		s.logger.Error("Failed to read request body", "error", err)
		return
	}
	defer r.Body.Close()

	// Parse message request
	var msgReq WhatsAppMessageRequest
	if err := json.Unmarshal(body, &msgReq); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		s.logger.Error("Failed to parse request JSON", "error", err)
		return
	}

	// Log the message details with pretty formatting
	s.logger.Info("📱 MOCK WhatsApp Message",
		"to", msgReq.To,
		"type", msgReq.Type,
		"messaging_product", msgReq.MessagingProduct,
	)

	// Log message content
	if msgReq.Text != nil {
		if body, ok := msgReq.Text["body"].(string); ok {
			s.logger.Info("💬 Message Content",
				"body", body,
			)
		}
	}

	if msgReq.Template != nil {
		templateName := ""
		if name, ok := msgReq.Template["name"].(string); ok {
			templateName = name
		}
		s.logger.Info("📋 Template Message",
			"template", templateName,
			"components", msgReq.Template,
		)
	}

	// Pretty print full request for debugging
	prettyJSON, err := json.MarshalIndent(msgReq, "", "  ")
	if err == nil {
		s.logger.Info("📨 Full Message Request",
			"json", string(prettyJSON),
		)
	}

	// Create mock response
	messageID := fmt.Sprintf("wamid.mock-%s", uuid.New().String())
	response := WhatsAppMessageResponse{
		MessagingProduct: "whatsapp",
		Contacts: []struct {
			Input string `json:"input"`
			WaID  string `json:"wa_id"`
		}{
			{
				Input: msgReq.To,
				WaID:  msgReq.To,
			},
		},
		Messages: []struct {
			ID string `json:"id"`
		}{
			{
				ID: messageID,
			},
		},
	}

	// Send response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)

	s.logger.Info("✅ Message sent successfully",
		"message_id", messageID,
		"to", msgReq.To,
		"duration", time.Since(start),
	)
}

func main() {
	// Setup logger
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	// Get configuration from environment
	port := os.Getenv("PORT")
	if port == "" {
		port = defaultPort
	}

	// Create server
	server := NewServer(logger)

	// Start HTTP server
	addr := fmt.Sprintf(":%s", port)
	logger.Info("🚀 Starting WhatsApp Mock Server",
		"port", port,
		"endpoint", fmt.Sprintf("http://localhost:%s/{phone_number_id}/messages", port),
	)

	if err := http.ListenAndServe(addr, server); err != nil {
		logger.Error("Server failed", "error", err)
		os.Exit(1)
	}
}
