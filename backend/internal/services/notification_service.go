package services

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/digital-egiz/backend/internal/utils"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

// Client represents a websocket client connection
type Client struct {
	conn      *websocket.Conn
	userID    uint
	projectID uint
	send      chan []byte
	topics    map[string]bool
}

// NotificationType defines types of notification messages
type NotificationType string

const (
	// NotificationTypeTwinUpdate for digital twin updates
	NotificationTypeTwinUpdate NotificationType = "twin_update"
	// NotificationTypeAlert for real-time alerts
	NotificationTypeAlert NotificationType = "alert"
	// NotificationTypeMLPrediction for ML prediction results
	NotificationTypeMLPrediction NotificationType = "ml_prediction"
	// NotificationTypeSystemEvent for system-wide events
	NotificationTypeSystemEvent NotificationType = "system_event"
)

// NotificationMessage represents a message sent to clients
type NotificationMessage struct {
	Type      NotificationType `json:"type"`
	Timestamp time.Time        `json:"timestamp"`
	Topic     string           `json:"topic"`
	Payload   interface{}      `json:"payload"`
}

// NotificationService manages websocket connections and notifications
type NotificationService struct {
	logger       *utils.Logger
	clients      map[*Client]bool
	register     chan *Client
	unregister   chan *Client
	broadcast    chan *NotificationMessage
	projectCasts map[uint]chan *NotificationMessage
	topics       map[string]chan *NotificationMessage
	mutex        sync.RWMutex
}

// NewNotificationService creates a new notification service
func NewNotificationService(logger *utils.Logger) *NotificationService {
	service := &NotificationService{
		logger:       logger.Named("notification_service"),
		clients:      make(map[*Client]bool),
		register:     make(chan *Client),
		unregister:   make(chan *Client),
		broadcast:    make(chan *NotificationMessage),
		projectCasts: make(map[uint]chan *NotificationMessage),
		topics:       make(map[string]chan *NotificationMessage),
		mutex:        sync.RWMutex{},
	}

	go service.run()
	return service
}

// RegisterClient adds a new websocket client
func (s *NotificationService) RegisterClient(conn *websocket.Conn, userID, projectID uint) *Client {
	client := &Client{
		conn:      conn,
		userID:    userID,
		projectID: projectID,
		send:      make(chan []byte, 256),
		topics:    make(map[string]bool),
	}

	s.register <- client

	// Start goroutines for reading and writing
	go s.readPump(client)
	go s.writePump(client)

	return client
}

// SubscribeToTopic subscribes a client to a specific topic
func (s *NotificationService) SubscribeToTopic(client *Client, topic string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	client.topics[topic] = true

	// Create topic channel if it doesn't exist
	if _, exists := s.topics[topic]; !exists {
		s.topics[topic] = make(chan *NotificationMessage, 256)
		go s.handleTopicMessages(topic)
	}

	s.logger.Debug("Client subscribed to topic",
		zap.Uint("user_id", client.userID),
		zap.String("topic", topic))
}

// UnsubscribeFromTopic unsubscribes a client from a specific topic
func (s *NotificationService) UnsubscribeFromTopic(client *Client, topic string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	delete(client.topics, topic)

	s.logger.Debug("Client unsubscribed from topic",
		zap.Uint("user_id", client.userID),
		zap.String("topic", topic))
}

// Notify sends a notification to all clients
func (s *NotificationService) Notify(notificationType NotificationType, topic string, payload interface{}) {
	message := &NotificationMessage{
		Type:      notificationType,
		Timestamp: time.Now(),
		Topic:     topic,
		Payload:   payload,
	}

	s.broadcast <- message
}

// NotifyProject sends a notification to all clients in a specific project
func (s *NotificationService) NotifyProject(projectID uint, notificationType NotificationType, topic string, payload interface{}) {
	message := &NotificationMessage{
		Type:      notificationType,
		Timestamp: time.Now(),
		Topic:     topic,
		Payload:   payload,
	}

	s.mutex.RLock()
	projectChan, exists := s.projectCasts[projectID]
	s.mutex.RUnlock()

	if exists {
		projectChan <- message
	} else {
		s.mutex.Lock()
		s.projectCasts[projectID] = make(chan *NotificationMessage, 256)
		s.mutex.Unlock()

		go s.handleProjectMessages(projectID)
		s.projectCasts[projectID] <- message
	}
}

// NotifyTopic sends a notification to all clients subscribed to a specific topic
func (s *NotificationService) NotifyTopic(topic string, notificationType NotificationType, payload interface{}) {
	message := &NotificationMessage{
		Type:      notificationType,
		Timestamp: time.Now(),
		Topic:     topic,
		Payload:   payload,
	}

	s.mutex.RLock()
	topicChan, exists := s.topics[topic]
	s.mutex.RUnlock()

	if exists {
		topicChan <- message
	} else {
		s.mutex.Lock()
		s.topics[topic] = make(chan *NotificationMessage, 256)
		s.mutex.Unlock()

		go s.handleTopicMessages(topic)
		s.topics[topic] <- message
	}
}

// run processes messages in the main loop
func (s *NotificationService) run() {
	for {
		select {
		case client := <-s.register:
			s.mutex.Lock()
			s.clients[client] = true
			s.mutex.Unlock()
			s.logger.Debug("Client registered",
				zap.Uint("user_id", client.userID),
				zap.Uint("project_id", client.projectID))

		case client := <-s.unregister:
			s.mutex.Lock()
			if _, ok := s.clients[client]; ok {
				delete(s.clients, client)
				close(client.send)
			}
			s.mutex.Unlock()
			s.logger.Debug("Client unregistered",
				zap.Uint("user_id", client.userID),
				zap.Uint("project_id", client.projectID))

		case message := <-s.broadcast:
			s.mutex.RLock()
			for client := range s.clients {
				s.sendToClient(client, message)
			}
			s.mutex.RUnlock()
		}
	}
}

// handleProjectMessages handles messages for a specific project
func (s *NotificationService) handleProjectMessages(projectID uint) {
	s.mutex.RLock()
	projectChan := s.projectCasts[projectID]
	s.mutex.RUnlock()

	for {
		message, ok := <-projectChan
		if !ok {
			return // Channel closed
		}

		s.mutex.RLock()
		for client := range s.clients {
			if client.projectID == projectID {
				s.sendToClient(client, message)
			}
		}
		s.mutex.RUnlock()
	}
}

// handleTopicMessages handles messages for a specific topic
func (s *NotificationService) handleTopicMessages(topic string) {
	s.mutex.RLock()
	topicChan := s.topics[topic]
	s.mutex.RUnlock()

	for {
		message, ok := <-topicChan
		if !ok {
			return // Channel closed
		}

		s.mutex.RLock()
		for client := range s.clients {
			if client.topics[topic] {
				s.sendToClient(client, message)
			}
		}
		s.mutex.RUnlock()
	}
}

// sendToClient sends a message to a specific client
func (s *NotificationService) sendToClient(client *Client, message *NotificationMessage) {
	jsonMessage, err := json.Marshal(message)
	if err != nil {
		s.logger.Error("Failed to marshal notification message",
			zap.Error(err),
			zap.String("type", string(message.Type)),
			zap.String("topic", message.Topic))
		return
	}

	select {
	case client.send <- jsonMessage:
		// Message sent
	default:
		// Client's send buffer is full
		s.mutex.Lock()
		delete(s.clients, client)
		close(client.send)
		s.mutex.Unlock()
		s.logger.Warn("Client buffer full, connection closed",
			zap.Uint("user_id", client.userID),
			zap.Uint("project_id", client.projectID))
	}
}

// readPump reads messages from the client
func (s *NotificationService) readPump(client *Client) {
	defer func() {
		s.unregister <- client
		client.conn.Close()
	}()

	// Set limits on websocket connection
	client.conn.SetReadLimit(4096) // 4KB max message size
	client.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	client.conn.SetPongHandler(func(string) error {
		client.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	// Main read loop - handle client messages
	for {
		_, message, err := client.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(
				err,
				websocket.CloseGoingAway,
				websocket.CloseAbnormalClosure,
			) {
				s.logger.Warn("Unexpected websocket close",
					zap.Error(err),
					zap.Uint("user_id", client.userID))
			}
			break
		}

		// Process client message (e.g., topic subscription)
		var clientMsg struct {
			Action string `json:"action"`
			Topic  string `json:"topic"`
		}

		if err := json.Unmarshal(message, &clientMsg); err != nil {
			s.logger.Warn("Invalid client message",
				zap.Error(err),
				zap.ByteString("message", message))
			continue
		}

		switch clientMsg.Action {
		case "subscribe":
			if clientMsg.Topic != "" {
				s.SubscribeToTopic(client, clientMsg.Topic)
			}
		case "unsubscribe":
			if clientMsg.Topic != "" {
				s.UnsubscribeFromTopic(client, clientMsg.Topic)
			}
		}
	}
}

// writePump writes messages to the client
func (s *NotificationService) writePump(client *Client) {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		client.conn.Close()
	}()

	for {
		select {
		case message, ok := <-client.send:
			client.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				// Channel closed
				client.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := client.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Add queued messages to the current websocket message
			n := len(client.send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-client.send)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			client.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := client.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
