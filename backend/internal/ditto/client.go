package ditto

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/digital-egiz/backend/internal/config"
	"github.com/digital-egiz/backend/internal/utils"
	"go.uber.org/zap"
)

// Client provides access to the Eclipse Ditto API
type Client struct {
	config      *config.DittoConfig
	httpClient  *http.Client
	logger      *utils.Logger
	baseURL     string
	authHeaders map[string]string
}

// NewClient creates a new Ditto API client
func NewClient(cfg *config.DittoConfig, logger *utils.Logger) *Client {
	// Create HTTP client with reasonable timeouts
	httpClient := &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        10,
			MaxIdleConnsPerHost: 10,
			IdleConnTimeout:     30 * time.Second,
		},
	}

	// Determine authentication method
	authHeaders := make(map[string]string)
	if cfg.APIToken != "" {
		authHeaders["Authorization"] = "Bearer " + cfg.APIToken
	} else if cfg.Username != "" && cfg.Password != "" {
		// Basic auth will be applied in the request
	}

	return &Client{
		config:      cfg,
		httpClient:  httpClient,
		logger:      logger.Named("ditto_client"),
		baseURL:     cfg.URL + "/api/2",
		authHeaders: authHeaders,
	}
}

// APIError represents an error response from the Ditto API
type APIError struct {
	StatusCode int
	Message    string
	Details    string
}

// Error returns the error message
func (e *APIError) Error() string {
	return fmt.Sprintf("Ditto API error (%d): %s - %s", e.StatusCode, e.Message, e.Details)
}

// doRequest performs an HTTP request to the Ditto API
func (c *Client) doRequest(ctx context.Context, method, path string, body interface{}) ([]byte, error) {
	url := c.baseURL + path

	var bodyReader io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewBuffer(jsonData)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add content type for requests with body
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	// Add authentication
	if c.config.APIToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.config.APIToken)
	} else if c.config.Username != "" && c.config.Password != "" {
		req.SetBasicAuth(c.config.Username, c.config.Password)
	}

	// Add common headers
	for key, value := range c.authHeaders {
		req.Header.Set(key, value)
	}

	// Log the request
	c.logger.Debug("Sending request to Ditto API",
		zap.String("method", method),
		zap.String("url", url),
	)

	// Send request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Handle non-success responses
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var errResp struct {
			Error   string `json:"error"`
			Message string `json:"message"`
		}
		if err := json.Unmarshal(respBody, &errResp); err != nil {
			return nil, &APIError{
				StatusCode: resp.StatusCode,
				Message:    "Unknown error",
				Details:    string(respBody),
			}
		}
		return nil, &APIError{
			StatusCode: resp.StatusCode,
			Message:    errResp.Error,
			Details:    errResp.Message,
		}
	}

	return respBody, nil
}

// ThingResponse represents a Ditto thing response
type ThingResponse struct {
	ThingID    string                 `json:"thingId"`
	Attributes map[string]interface{} `json:"attributes,omitempty"`
	Features   map[string]Feature     `json:"features,omitempty"`
}

// Feature represents a Ditto thing feature
type Feature struct {
	Properties map[string]interface{} `json:"properties,omitempty"`
}

// GetThing retrieves a thing by ID
func (c *Client) GetThing(ctx context.Context, thingID string) (*ThingResponse, error) {
	path := fmt.Sprintf("/things/%s", thingID)
	respBody, err := c.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var thing ThingResponse
	if err := json.Unmarshal(respBody, &thing); err != nil {
		return nil, fmt.Errorf("failed to parse thing response: %w", err)
	}

	return &thing, nil
}

// CreateThing creates a new thing
func (c *Client) CreateThing(ctx context.Context, thingID string, attributes map[string]interface{}, features map[string]Feature) (*ThingResponse, error) {
	path := "/things"
	thing := map[string]interface{}{
		"thingId": thingID,
	}
	if attributes != nil {
		thing["attributes"] = attributes
	}
	if features != nil {
		thing["features"] = features
	}

	respBody, err := c.doRequest(ctx, http.MethodPost, path, thing)
	if err != nil {
		return nil, err
	}

	var response ThingResponse
	if err := json.Unmarshal(respBody, &response); err != nil {
		return nil, fmt.Errorf("failed to parse thing response: %w", err)
	}

	return &response, nil
}

// UpdateThing updates an existing thing
func (c *Client) UpdateThing(ctx context.Context, thingID string, thing map[string]interface{}) error {
	path := fmt.Sprintf("/things/%s", thingID)
	_, err := c.doRequest(ctx, http.MethodPut, path, thing)
	return err
}

// DeleteThing deletes a thing
func (c *Client) DeleteThing(ctx context.Context, thingID string) error {
	path := fmt.Sprintf("/things/%s", thingID)
	_, err := c.doRequest(ctx, http.MethodDelete, path, nil)
	return err
}

// GetFeature retrieves a feature of a thing
func (c *Client) GetFeature(ctx context.Context, thingID, featureID string) (*Feature, error) {
	path := fmt.Sprintf("/things/%s/features/%s", thingID, featureID)
	respBody, err := c.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var feature Feature
	if err := json.Unmarshal(respBody, &feature); err != nil {
		return nil, fmt.Errorf("failed to parse feature response: %w", err)
	}

	return &feature, nil
}

// UpdateFeature updates a feature of a thing
func (c *Client) UpdateFeature(ctx context.Context, thingID, featureID string, feature Feature) error {
	path := fmt.Sprintf("/things/%s/features/%s", thingID, featureID)
	_, err := c.doRequest(ctx, http.MethodPut, path, feature)
	return err
}

// UpdateFeatureProperty updates a property of a feature
func (c *Client) UpdateFeatureProperty(ctx context.Context, thingID, featureID, propertyPath string, value interface{}) error {
	path := fmt.Sprintf("/things/%s/features/%s/properties/%s", thingID, featureID, propertyPath)
	_, err := c.doRequest(ctx, http.MethodPut, path, value)
	return err
}

// SearchThings searches for things based on a query
func (c *Client) SearchThings(ctx context.Context, query string, options map[string]string) ([]ThingResponse, error) {
	path := "/search/things?filter=" + query
	for key, value := range options {
		path += fmt.Sprintf("&%s=%s", key, value)
	}

	respBody, err := c.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var response struct {
		Items []ThingResponse `json:"items"`
	}
	if err := json.Unmarshal(respBody, &response); err != nil {
		return nil, fmt.Errorf("failed to parse search response: %w", err)
	}

	return response.Items, nil
}
