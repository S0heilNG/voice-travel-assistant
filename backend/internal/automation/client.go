// Package automation holds the HTTP client that talks to the
// automation-service (Node.js + Playwright) for flight/hotel searches on platform 780.
package automation

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
)

type AutomationClient struct {
	BaseURL    string
	httpClient *http.Client
}

func NewAutomationClient() *AutomationClient {
	baseURL := os.Getenv("AUTOMATION_SERVICE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:4000"
	}
	return &AutomationClient{
		BaseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

type searchRequest struct {
	Origin      string `json:"origin"`
	Destination string `json:"destination"`
	Date        string `json:"date"`
}

type SearchResult struct {
	OK     bool   `json:"ok"`
	Title  string `json:"title"`
	TookMs int64  `json:"tookMs"`
	Error  string `json:"error"`
}

func (c *AutomationClient) SearchFlights(origin, destination, date string) (*SearchResult, error) {
	return c.post("/search/flights", origin, destination, date)
}

func (c *AutomationClient) SearchHotels(origin, destination, date string) (*SearchResult, error) {
	return c.post("/search/hotels", origin, destination, date)
}

func (c *AutomationClient) post(path, origin, destination, date string) (*SearchResult, error) {
	body, err := json.Marshal(searchRequest{Origin: origin, Destination: destination, Date: date})
	if err != nil {
		return nil, fmt.Errorf("encoding automation-service request: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, c.BaseURL+path, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("building automation-service request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("calling automation-service: %w", err)
	}
	defer resp.Body.Close()

	var result SearchResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decoding automation-service response: %w", err)
	}

	if !result.OK {
		return &result, fmt.Errorf("automation-service error: %s", result.Error)
	}

	return &result, nil
}
