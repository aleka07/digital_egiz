package ditto

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/digital-egiz/backend/internal/config"
	"github.com/digital-egiz/backend/internal/utils"
)

// Client is a client for the Eclipse Ditto HTTP API
type Client struct {
	config     *config.DittoConfig
	logger     *utils.Logger
	httpClient *http.Client
}

// Thing represents a Digital Twin in Eclipse Ditto
type Thing struct {
	ThingID    string                 `json:"thingId,omitempty"`
	PolicyID   string                 `json:"policyId,omitempty"`
	Definition string                 `json:"definition,omitempty"`
	Attributes map[string]interface{} `json:"attributes,omitempty"`
	Features   map[string]Feature     `json:"features,omitempty"`
	Metadata   map[string]interface{} `json:"_metadata,omitempty"`
	Revision   int64                  `json:"_revision,omitempty"`
	Modified   string                 `json:"_modified,omitempty"`
	Created    string                 `json:"_created,omitempty"`
	Namespace  string                 `json:"-"`
	ID         string                 `json:"-"`
}

// Feature represents a Feature of a Thing in Eclipse Ditto
type Feature struct {
	Definition string                 `json:"definition,omitempty"`
	Properties map[string]interface{} `json:"properties,omitempty"`
}

// FeatureProperties represents the properties of a feature
type FeatureProperties map[string]interface{}

// Policy represents an access policy in Eclipse Ditto
type Policy struct {
	PolicyID  string                 `json:"policyId,omitempty"`
	Entries   map[string]PolicyEntry `json:"entries,omitempty"`
	Revision  int64                  `json:"_revision,omitempty"`
	Modified  string                 `json:"_modified,omitempty"`
	Created   string                 `json:"_created,omitempty"`
	Namespace string                 `json:"-"`
	ID        string                 `json:"-"`
}

// PolicyEntry represents an entry in a policy
type PolicyEntry struct {
	Subjects  map[string]Subject  `json:"subjects,omitempty"`
	Resources map[string]Resource `json:"resources,omitempty"`
}

// Subject represents a subject in a policy entry
type Subject struct {
	Type string `json:"type,omitempty"`
}

// Resource represents a resource in a policy entry
type Resource struct {
	Grant  []string `json:"grant,omitempty"`
	Revoke []string `json:"revoke,omitempty"`
}

// DittoError represents an error returned by the Ditto API
type DittoError struct {
	Status      int                    `json:"status"`
	ErrorCode   string                 `json:"error"`
	Message     string                 `json:"message"`
	Description string                 `json:"description,omitempty"`
	Href        string                 `json:"href,omitempty"`
	Raw         map[string]interface{} `json:"-"`
}

// Error returns the error message
func (e *DittoError) Error() string {
	return fmt.Sprintf("Ditto API error: %d %s - %s", e.Status, e.ErrorCode, e.Message)
}

// NewClient creates a new Ditto client
func NewClient(cfg *config.DittoConfig, logger *utils.Logger) *Client {
	return &Client{
		config: cfg,
		logger: logger.Named("ditto"),
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// buildURL builds a URL for the Ditto API
func (c *Client) buildURL(path string) string {
	return fmt.Sprintf("%s/api/2%s", c.config.URL, path)
}

// execute executes a request to the Ditto API
func (c *Client) execute(ctx context.Context, method, path string, body interface{}) ([]byte, error) {
	url := c.buildURL(path)

	var bodyReader io.Reader
	if body != nil {
		bodyBytes, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(bodyBytes)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// Add authentication header
	if c.config.APIToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.config.APIToken)
	} else if c.config.Username != "" && c.config.Password != "" {
		auth := c.config.Username + ":" + c.config.Password
		encoded := base64.StdEncoding.EncodeToString([]byte(auth))
		req.Header.Set("Authorization", "Basic "+encoded)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Check for errors
	if resp.StatusCode >= 400 {
		var dittoErr DittoError
		if err := json.Unmarshal(responseBody, &dittoErr); err != nil {
			return nil, fmt.Errorf("HTTP error %d: %s", resp.StatusCode, string(responseBody))
		}

		// Try to unmarshal the raw error response too
		var raw map[string]interface{}
		if err := json.Unmarshal(responseBody, &raw); err == nil {
			dittoErr.Raw = raw
		}

		return nil, &dittoErr
	}

	return responseBody, nil
}

// CreateThing creates a new thing
func (c *Client) CreateThing(ctx context.Context, thing *Thing) (*Thing, error) {
	path := "/things"

	// Ensure a policy ID is set
	if thing.PolicyID == "" {
		// Create a default policy ID if not provided
		thing.PolicyID = fmt.Sprintf("%s:policy", thing.ThingID)
	}

	responseBody, err := c.execute(ctx, http.MethodPost, path, thing)
	if err != nil {
		return nil, err
	}

	var createdThing Thing
	if err := json.Unmarshal(responseBody, &createdThing); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &createdThing, nil
}

// GetThing retrieves a thing by its ID
func (c *Client) GetThing(ctx context.Context, thingID string) (*Thing, error) {
	path := fmt.Sprintf("/things/%s", thingID)

	responseBody, err := c.execute(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var thing Thing
	if err := json.Unmarshal(responseBody, &thing); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &thing, nil
}

// UpdateThing updates an existing thing
func (c *Client) UpdateThing(ctx context.Context, thingID string, thing *Thing) (*Thing, error) {
	path := fmt.Sprintf("/things/%s", thingID)

	responseBody, err := c.execute(ctx, http.MethodPut, path, thing)
	if err != nil {
		return nil, err
	}

	var updatedThing Thing
	if err := json.Unmarshal(responseBody, &updatedThing); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &updatedThing, nil
}

// DeleteThing deletes a thing
func (c *Client) DeleteThing(ctx context.Context, thingID string) error {
	path := fmt.Sprintf("/things/%s", thingID)

	_, err := c.execute(ctx, http.MethodDelete, path, nil)
	return err
}

// ListThings retrieves all things
func (c *Client) ListThings(ctx context.Context, namespaces []string, filter string, options map[string]string) ([]Thing, error) {
	path := "/search/things"

	// Build query parameters
	queryParams := make(map[string]string)
	if filter != "" {
		queryParams["filter"] = filter
	}
	if len(namespaces) > 0 {
		namespaceQuery := ""
		for i, ns := range namespaces {
			if i > 0 {
				namespaceQuery += ","
			}
			namespaceQuery += ns
		}
		queryParams["namespaces"] = namespaceQuery
	}

	// Add options
	for k, v := range options {
		queryParams[k] = v
	}

	// Build query string
	queryString := ""
	for k, v := range queryParams {
		if queryString == "" {
			queryString = "?"
		} else {
			queryString += "&"
		}
		queryString += fmt.Sprintf("%s=%s", k, v)
	}

	responseBody, err := c.execute(ctx, http.MethodGet, path+queryString, nil)
	if err != nil {
		return nil, err
	}

	var items []Thing
	if err := json.Unmarshal(responseBody, &items); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return items, nil
}

// CreateFeature creates or updates a feature for a thing
func (c *Client) CreateFeature(ctx context.Context, thingID, featureID string, feature *Feature) (*Feature, error) {
	path := fmt.Sprintf("/things/%s/features/%s", thingID, featureID)

	responseBody, err := c.execute(ctx, http.MethodPut, path, feature)
	if err != nil {
		return nil, err
	}

	var createdFeature Feature
	if err := json.Unmarshal(responseBody, &createdFeature); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &createdFeature, nil
}

// GetFeature retrieves a feature of a thing
func (c *Client) GetFeature(ctx context.Context, thingID, featureID string) (*Feature, error) {
	path := fmt.Sprintf("/things/%s/features/%s", thingID, featureID)

	responseBody, err := c.execute(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var feature Feature
	if err := json.Unmarshal(responseBody, &feature); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &feature, nil
}

// UpdateFeature updates a feature of a thing
func (c *Client) UpdateFeature(ctx context.Context, thingID, featureID string, feature *Feature) (*Feature, error) {
	return c.CreateFeature(ctx, thingID, featureID, feature)
}

// DeleteFeature deletes a feature of a thing
func (c *Client) DeleteFeature(ctx context.Context, thingID, featureID string) error {
	path := fmt.Sprintf("/things/%s/features/%s", thingID, featureID)

	_, err := c.execute(ctx, http.MethodDelete, path, nil)
	return err
}

// UpdateFeatureProperties updates the properties of a feature
func (c *Client) UpdateFeatureProperties(ctx context.Context, thingID, featureID string, properties FeatureProperties) (*FeatureProperties, error) {
	path := fmt.Sprintf("/things/%s/features/%s/properties", thingID, featureID)

	responseBody, err := c.execute(ctx, http.MethodPut, path, properties)
	if err != nil {
		return nil, err
	}

	var updatedProperties FeatureProperties
	if err := json.Unmarshal(responseBody, &updatedProperties); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &updatedProperties, nil
}

// GetFeatureProperties retrieves the properties of a feature
func (c *Client) GetFeatureProperties(ctx context.Context, thingID, featureID string) (*FeatureProperties, error) {
	path := fmt.Sprintf("/things/%s/features/%s/properties", thingID, featureID)

	responseBody, err := c.execute(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var properties FeatureProperties
	if err := json.Unmarshal(responseBody, &properties); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &properties, nil
}

// CreatePolicy creates a new policy
func (c *Client) CreatePolicy(ctx context.Context, policy *Policy) (*Policy, error) {
	path := "/policies"

	responseBody, err := c.execute(ctx, http.MethodPost, path, policy)
	if err != nil {
		return nil, err
	}

	var createdPolicy Policy
	if err := json.Unmarshal(responseBody, &createdPolicy); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &createdPolicy, nil
}

// GetPolicy retrieves a policy by its ID
func (c *Client) GetPolicy(ctx context.Context, policyID string) (*Policy, error) {
	path := fmt.Sprintf("/policies/%s", policyID)

	responseBody, err := c.execute(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var policy Policy
	if err := json.Unmarshal(responseBody, &policy); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &policy, nil
}

// UpdatePolicy updates a policy
func (c *Client) UpdatePolicy(ctx context.Context, policyID string, policy *Policy) (*Policy, error) {
	path := fmt.Sprintf("/policies/%s", policyID)

	responseBody, err := c.execute(ctx, http.MethodPut, path, policy)
	if err != nil {
		return nil, err
	}

	var updatedPolicy Policy
	if err := json.Unmarshal(responseBody, &updatedPolicy); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &updatedPolicy, nil
}

// DeletePolicy deletes a policy
func (c *Client) DeletePolicy(ctx context.Context, policyID string) error {
	path := fmt.Sprintf("/policies/%s", policyID)

	_, err := c.execute(ctx, http.MethodDelete, path, nil)
	return err
}
