package ditto

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/digital-egiz/backend/internal/config"
	"github.com/digital-egiz/backend/internal/utils"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

// WebSocketClient provides a WebSocket connection to Eclipse Ditto for real-time events
type WebSocketClient struct {
	config      *config.DittoConfig
	logger      *utils.Logger
	conn        *websocket.Conn
	mu          sync.Mutex
	isConnected bool
	handlers    map[string]EventHandler
	ctx         context.Context
	cancel      context.CancelFunc
	backoff     time.Duration
	maxBackoff  time.Duration
}

// EventHandler is a function that processes Ditto events
type EventHandler func(event *DittoEvent)

// DittoEvent represents an event from the Ditto WebSocket
type DittoEvent struct {
	Topic     string                 `json:"topic"`
	Path      string                 `json:"path"`
	Value     interface{}            `json:"value,omitempty"`
	Revision  int64                  `json:"revision,omitempty"`
	Timestamp string                 `json:"timestamp,omitempty"`
	Metadata  map[string]interface{} `json:"_metadata,omitempty"`
	Headers   map[string]interface{} `json:"headers,omitempty"`
	Extra     map[string]interface{} `json:"extra,omitempty"`
	Status    int                    `json:"status,omitempty"`
	ThingID   string                 `json:"-"` // Parsed from topic
	Action    string                 `json:"-"` // Parsed from topic
	FeatureID string                 `json:"-"` // Parsed from path if applicable
}

// NewWebSocketClient creates a new WebSocket client for Ditto
func NewWebSocketClient(cfg *config.DittoConfig, logger *utils.Logger) *WebSocketClient {
	ctx, cancel := context.WithCancel(context.Background())
	return &WebSocketClient{
		config:     cfg,
		logger:     logger.Named("ditto_ws"),
		handlers:   make(map[string]EventHandler),
		ctx:        ctx,
		cancel:     cancel,
		backoff:    1 * time.Second,
		maxBackoff: 60 * time.Second,
	}
}

// Connect establishes a WebSocket connection to Ditto
func (c *WebSocketClient) Connect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.isConnected {
		return nil
	}

	// Parse the WebSocket URL (using /ws endpoint)
	wsURL := c.config.URL
	// Replace http(s) with ws(s)
	if wsURL[:5] == "https" {
		wsURL = "wss" + wsURL[5:]
	} else if wsURL[:4] == "http" {
		wsURL = "ws" + wsURL[4:]
	}
	wsURL = wsURL + "/ws/2"

	// Build request headers with authentication
	header := http.Header{}
	if c.config.APIToken != "" {
		header.Add("Authorization", "Bearer "+c.config.APIToken)
	} else if c.config.Username != "" && c.config.Password != "" {
		auth := c.config.Username + ":" + c.config.Password
		encoded := base64.StdEncoding.EncodeToString([]byte(auth))
		header.Add("Authorization", "Basic "+encoded)
	}

	c.logger.Info("Connecting to Ditto WebSocket", zap.String("url", wsURL))

	// Connect to the WebSocket
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, header)
	if err != nil {
		return fmt.Errorf("failed to connect to WebSocket: %w", err)
	}

	c.conn = conn
	c.isConnected = true
	c.backoff = 1 * time.Second // Reset backoff timer on successful connection

	// Start the message handler in a goroutine
	go c.handleMessages()

	return nil
}

// Disconnect closes the WebSocket connection
func (c *WebSocketClient) Disconnect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.isConnected {
		return nil
	}

	c.cancel() // Cancel the context to stop the handling goroutine

	// Close the connection
	err := c.conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	if err != nil {
		c.logger.Warn("Error while sending close message", zap.Error(err))
	}

	err = c.conn.Close()
	c.isConnected = false
	c.conn = nil

	return err
}

// IsConnected returns whether the client is connected
func (c *WebSocketClient) IsConnected() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.isConnected
}

// SubscribeToThings subscribes to thing change events
func (c *WebSocketClient) SubscribeToThings(filter string) error {
	// Build subscription command
	subscription := map[string]interface{}{
		"topic":  "/_/things/twin/events",
		"filter": filter,
		"namespaces": []string{
			"org.eclipse.ditto", // Replace with your namespaces as needed
		},
	}

	return c.sendCommand("START-SEND-EVENTS", subscription)
}

// SubscribeToThing subscribes to events for a specific thing
func (c *WebSocketClient) SubscribeToThing(thingID string) error {
	// Build subscription command
	subscription := map[string]interface{}{
		"topic": fmt.Sprintf("/%s/things/twin/events", thingID),
	}

	return c.sendCommand("START-SEND-EVENTS", subscription)
}

// SubscribeToFeature subscribes to events for a specific feature of a thing
func (c *WebSocketClient) SubscribeToFeature(thingID, featureID string) error {
	// Build subscription command
	subscription := map[string]interface{}{
		"topic": fmt.Sprintf("/%s/things/twin/events?extraFields=features/%s", thingID, featureID),
	}

	return c.sendCommand("START-SEND-EVENTS", subscription)
}

// Unsubscribe cancels all subscriptions
func (c *WebSocketClient) Unsubscribe() error {
	return c.sendCommand("STOP-SEND-EVENTS", nil)
}

// RegisterHandler registers a handler for a specific event type
// eventType should be "thing", "feature", or a specific action like "thing.created"
func (c *WebSocketClient) RegisterHandler(eventType string, handler EventHandler) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.handlers[eventType] = handler
}

// handleMessages processes WebSocket messages
func (c *WebSocketClient) handleMessages() {
	defer func() {
		if r := recover(); r != nil {
			c.logger.Error("Panic in WebSocket message handler", zap.Any("recover", r))
		}
	}()

	c.logger.Info("Starting WebSocket message handler")

	// Keep reading messages until context is canceled
	for {
		select {
		case <-c.ctx.Done():
			c.logger.Info("WebSocket handler stopped due to context cancellation")
			return
		default:
			// Continue processing
		}

		if !c.IsConnected() {
			// Try to reconnect with exponential backoff
			c.logger.Info("WebSocket disconnected, attempting to reconnect",
				zap.Duration("backoff", c.backoff))

			time.Sleep(c.backoff)

			// Increase backoff for next attempt with a cap
			c.backoff = time.Duration(float64(c.backoff) * 1.5)
			if c.backoff > c.maxBackoff {
				c.backoff = c.maxBackoff
			}

			err := c.Connect()
			if err != nil {
				c.logger.Error("Failed to reconnect WebSocket", zap.Error(err))
				continue
			}
		}

		// Read the next message
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			c.mu.Lock()
			c.isConnected = false
			c.mu.Unlock()

			if websocket.IsUnexpectedCloseError(err,
				websocket.CloseGoingAway,
				websocket.CloseNormalClosure) {
				c.logger.Error("WebSocket read error", zap.Error(err))
			} else {
				c.logger.Warn("WebSocket closed", zap.Error(err))
			}
			continue
		}

		// Process the message
		go c.processMessage(message)
	}
}

// processMessage handles a single WebSocket message
func (c *WebSocketClient) processMessage(message []byte) {
	var event DittoEvent
	if err := json.Unmarshal(message, &event); err != nil {
		c.logger.Error("Failed to unmarshal WebSocket message",
			zap.Error(err),
			zap.String("message", string(message)))
		return
	}

	// Parse topic to extract thingId and action
	// Topics are in the format: <namespace>/<entityId>/things/twin/events/<action>
	parts := splitAndStripEmpty(event.Topic, "/")
	if len(parts) >= 6 {
		event.ThingID = parts[1]
		event.Action = parts[5]
	}

	// Parse path to extract featureId if applicable
	if event.Path != "" && len(event.Path) > 10 && event.Path[:10] == "/features/" {
		featurePath := event.Path[10:]
		featureParts := splitAndStripEmpty(featurePath, "/")
		if len(featureParts) > 0 {
			event.FeatureID = featureParts[0]
		}
	}

	c.mu.Lock()
	handlers := c.handlers
	c.mu.Unlock()

	// Dispatch to handlers
	if event.FeatureID != "" {
		// Check for specific feature handler
		if handler, ok := handlers["feature."+event.FeatureID]; ok {
			handler(&event)
			return
		}

		// Check for general feature handler
		if handler, ok := handlers["feature"]; ok {
			handler(&event)
			return
		}
	}

	// Check for specific action handler
	specificHandler := "thing." + event.Action
	if handler, ok := handlers[specificHandler]; ok {
		handler(&event)
		return
	}

	// Default to general thing handler
	if handler, ok := handlers["thing"]; ok {
		handler(&event)
		return
	}

	c.logger.Debug("No handler for event",
		zap.String("topic", event.Topic),
		zap.String("thingId", event.ThingID),
		zap.String("action", event.Action))
}

// sendCommand sends a command to the Ditto WebSocket
func (c *WebSocketClient) sendCommand(command string, payload interface{}) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.isConnected {
		return fmt.Errorf("not connected to WebSocket")
	}

	// Build command
	cmd := map[string]interface{}{
		"type": command,
	}
	if payload != nil {
		cmd["payload"] = payload
	}

	// Marshal command to JSON
	data, err := json.Marshal(cmd)
	if err != nil {
		return fmt.Errorf("failed to marshal command: %w", err)
	}

	// Send command
	err = c.conn.WriteMessage(websocket.TextMessage, data)
	if err != nil {
		c.isConnected = false
		return fmt.Errorf("failed to send command: %w", err)
	}

	return nil
}

// splitAndStripEmpty splits a string and removes empty parts
func splitAndStripEmpty(s, sep string) []string {
	parts := []string{}
	for _, part := range splitN(s, sep, -1) {
		if part != "" {
			parts = append(parts, part)
		}
	}
	return parts
}

// splitN is a helper that splits a string
func splitN(s, sep string, n int) []string {
	return splitFunc(s, func(c rune) bool {
		return string(c) == sep
	}, n)
}

// splitFunc is a helper that splits a string
func splitFunc(s string, f func(rune) bool, n int) []string {
	if n == 0 {
		return nil
	}
	if s == "" {
		return []string{""}
	}

	parts := []string{}
	start := 0
	for i, c := range s {
		if f(c) {
			if n > 0 && len(parts) == n-1 {
				break
			}
			parts = append(parts, s[start:i])
			start = i + 1
		}
	}

	// Add the last part
	if start < len(s) {
		parts = append(parts, s[start:])
	} else if len(s) > 0 && f(rune(s[len(s)-1])) {
		parts = append(parts, "")
	}

	return parts
}
