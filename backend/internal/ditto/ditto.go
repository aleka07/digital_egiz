package ditto

import (
	"context"
	"sync"

	"github.com/digital-egiz/backend/internal/config"
	"github.com/digital-egiz/backend/internal/utils"
)

// Manager is a composite service that handles both HTTP API and WebSocket connections
// to Eclipse Ditto for managing digital twins
type Manager struct {
	config       *config.DittoConfig
	logger       *utils.Logger
	httpClient   *Client
	wsClient     *WebSocketClient
	eventHandler EventHandler
	mu           sync.RWMutex
}

// NewManager creates a new Ditto Manager that combines HTTP API and WebSocket functionalities
func NewManager(cfg *config.DittoConfig, logger *utils.Logger) *Manager {
	return &Manager{
		config:     cfg,
		logger:     logger.Named("ditto_manager"),
		httpClient: NewClient(cfg, logger),
		wsClient:   NewWebSocketClient(cfg, logger),
	}
}

// SetEventHandler sets the callback for handling WebSocket events
func (m *Manager) SetEventHandler(handler EventHandler) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.eventHandler = handler
	m.wsClient.RegisterHandler("thing", handler)
}

// Connect establishes a WebSocket connection
func (m *Manager) Connect() error {
	return m.wsClient.Connect()
}

// Disconnect closes the WebSocket connection
func (m *Manager) Disconnect() error {
	return m.wsClient.Disconnect()
}

// IsConnected returns whether the WebSocket is connected
func (m *Manager) IsConnected() bool {
	return m.wsClient.IsConnected()
}

// SubscribeToThings subscribes to thing change events
func (m *Manager) SubscribeToThings(filter string) error {
	return m.wsClient.SubscribeToThings(filter)
}

// SubscribeToThing subscribes to events for a specific thing
func (m *Manager) SubscribeToThing(thingID string) error {
	return m.wsClient.SubscribeToThing(thingID)
}

// SubscribeToFeature subscribes to events for a specific feature of a thing
func (m *Manager) SubscribeToFeature(thingID, featureID string) error {
	return m.wsClient.SubscribeToFeature(thingID, featureID)
}

// Unsubscribe cancels all subscriptions
func (m *Manager) Unsubscribe() error {
	return m.wsClient.Unsubscribe()
}

// CreateThing creates a new thing
func (m *Manager) CreateThing(ctx context.Context, thing *Thing) (*Thing, error) {
	return m.httpClient.CreateThing(ctx, thing)
}

// GetThing retrieves a thing by its ID
func (m *Manager) GetThing(ctx context.Context, thingID string) (*Thing, error) {
	return m.httpClient.GetThing(ctx, thingID)
}

// UpdateThing updates an existing thing
func (m *Manager) UpdateThing(ctx context.Context, thingID string, thing *Thing) (*Thing, error) {
	return m.httpClient.UpdateThing(ctx, thingID, thing)
}

// DeleteThing deletes a thing
func (m *Manager) DeleteThing(ctx context.Context, thingID string) error {
	return m.httpClient.DeleteThing(ctx, thingID)
}

// ListThings retrieves all things
func (m *Manager) ListThings(ctx context.Context, namespaces []string, filter string, options map[string]string) ([]Thing, error) {
	return m.httpClient.ListThings(ctx, namespaces, filter, options)
}

// CreateFeature creates or updates a feature for a thing
func (m *Manager) CreateFeature(ctx context.Context, thingID, featureID string, feature *Feature) (*Feature, error) {
	return m.httpClient.CreateFeature(ctx, thingID, featureID, feature)
}

// GetFeature retrieves a feature of a thing
func (m *Manager) GetFeature(ctx context.Context, thingID, featureID string) (*Feature, error) {
	return m.httpClient.GetFeature(ctx, thingID, featureID)
}

// UpdateFeature updates a feature of a thing
func (m *Manager) UpdateFeature(ctx context.Context, thingID, featureID string, feature *Feature) (*Feature, error) {
	return m.httpClient.UpdateFeature(ctx, thingID, featureID, feature)
}

// DeleteFeature deletes a feature of a thing
func (m *Manager) DeleteFeature(ctx context.Context, thingID, featureID string) error {
	return m.httpClient.DeleteFeature(ctx, thingID, featureID)
}

// UpdateFeatureProperties updates the properties of a feature
func (m *Manager) UpdateFeatureProperties(ctx context.Context, thingID, featureID string, properties FeatureProperties) (*FeatureProperties, error) {
	return m.httpClient.UpdateFeatureProperties(ctx, thingID, featureID, properties)
}

// GetFeatureProperties retrieves the properties of a feature
func (m *Manager) GetFeatureProperties(ctx context.Context, thingID, featureID string) (*FeatureProperties, error) {
	return m.httpClient.GetFeatureProperties(ctx, thingID, featureID)
}

// CreatePolicy creates a new policy
func (m *Manager) CreatePolicy(ctx context.Context, policy *Policy) (*Policy, error) {
	return m.httpClient.CreatePolicy(ctx, policy)
}

// GetPolicy retrieves a policy by its ID
func (m *Manager) GetPolicy(ctx context.Context, policyID string) (*Policy, error) {
	return m.httpClient.GetPolicy(ctx, policyID)
}

// UpdatePolicy updates a policy
func (m *Manager) UpdatePolicy(ctx context.Context, policyID string, policy *Policy) (*Policy, error) {
	return m.httpClient.UpdatePolicy(ctx, policyID, policy)
}

// DeletePolicy deletes a policy
func (m *Manager) DeletePolicy(ctx context.Context, policyID string) error {
	return m.httpClient.DeletePolicy(ctx, policyID)
}
