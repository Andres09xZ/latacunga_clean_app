package clients

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
)

// IncidentClient for calling incident-service API
type IncidentClient struct {
	baseURL string
	client  *http.Client
}

// NewIncidentClient creates a new incident client
func NewIncidentClient(baseURL string) *IncidentClient {
	return &IncidentClient{
		baseURL: baseURL,
		client:  &http.Client{},
	}
}

// IncidentResponse from incident-service
type IncidentResponse struct {
	ID             string   `json:"id"`
	ReporterKind   string   `json:"reporter_kind"`
	ReporterID     string   `json:"reporter_id"`
	Type           string   `json:"type"`
	Status         string   `json:"status"`
	Description    string   `json:"description"`
	Latitude       *float64 `json:"latitude"`
	Longitude      *float64 `json:"longitude"`
	IncidentDay    string   `json:"incident_day"`
	PhotosCount    int      `json:"photos_count"`
	IdempotencyKey string   `json:"idempotency_key"`
	CreatedAt      string   `json:"created_at"`
}

// GetIncident retrieves incident by ID from incident-service
func (ic *IncidentClient) GetIncident(incidentID string, token string) (*IncidentResponse, error) {
	url := fmt.Sprintf("%s/api/v1/incidents/%s", ic.baseURL, incidentID)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	resp, err := ic.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call incident-service: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		log.Printf("Incident service returned %d: %s", resp.StatusCode, string(body))
		return nil, fmt.Errorf("incident not found (status %d)", resp.StatusCode)
	}

	var incident IncidentResponse
	if err := json.Unmarshal(body, &incident); err != nil {
		return nil, fmt.Errorf("failed to unmarshal incident: %w", err)
	}

	return &incident, nil
}

// UpdateIncidentStatusRequest for updating incident status
type UpdateIncidentStatusRequest struct {
	Status string `json:"status"`
}

// UpdateIncidentStatus updates the status of an incident
func (ic *IncidentClient) UpdateIncidentStatus(incidentID string, status string, token string) error {
	url := fmt.Sprintf("%s/api/v1/incidents/%s/status", ic.baseURL, incidentID)

	updateReq := UpdateIncidentStatusRequest{
		Status: status,
	}

	body, err := json.Marshal(updateReq)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("PUT", url, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	resp, err := ic.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to call incident-service: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		log.Printf("Failed to update incident status: %d - %s", resp.StatusCode, string(respBody))
		return fmt.Errorf("failed to update incident status (status %d)", resp.StatusCode)
	}

	log.Printf("✅ Incident %s status updated to %s", incidentID, status)
	return nil
}
