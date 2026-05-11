package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type Client struct {
	BaseURL  string
	AdminKey string
	http     *http.Client
}

func NewClient(baseURL, adminKey string) *Client {
	return &Client{
		BaseURL:  baseURL,
		AdminKey: adminKey,
		http:     &http.Client{Timeout: 5 * time.Second},
	}
}

func (c *Client) do(method, path string, body []byte) (*http.Response, error) {
	var bodyReader io.Reader
	if body != nil {
		bodyReader = bytes.NewReader(body)
	}
	req, err := http.NewRequest(method, c.BaseURL+path, bodyReader)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.AdminKey)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		defer resp.Body.Close()
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(respBody)))
	}
	return resp, nil
}

func (c *Client) ListPublicStations() ([]PublicStation, error) {
	req, err := http.NewRequest(http.MethodGet, c.BaseURL+"/stations", nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var stations []PublicStation
	if err := json.NewDecoder(resp.Body).Decode(&stations); err != nil {
		return nil, err
	}
	return stations, nil
}

func (c *Client) ListStations() ([]StationStatus, error) {
	resp, err := c.do(http.MethodGet, "/admin/stations", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var stations []StationStatus
	if err := json.NewDecoder(resp.Body).Decode(&stations); err != nil {
		return nil, err
	}
	return stations, nil
}

func (c *Client) GetStation(username string) (StationStatus, error) {
	resp, err := c.do(http.MethodGet, "/admin/stations/"+username, nil)
	if err != nil {
		return StationStatus{}, err
	}
	defer resp.Body.Close()
	var s StationStatus
	if err := json.NewDecoder(resp.Body).Decode(&s); err != nil {
		return StationStatus{}, err
	}
	return s, nil
}

func (c *Client) StopStream(username string) error {
	resp, err := c.do(http.MethodPost, "/admin/stations/"+username+"/stream/stop", nil)
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

func (c *Client) StartStream(username string) error {
	resp, err := c.do(http.MethodPost, "/admin/stations/"+username+"/stream/start", nil)
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

func (c *Client) StartRelay(username, sourceURL string) error {
	body, err := json.Marshal(map[string]string{"source_url": sourceURL})
	if err != nil {
		return err
	}
	resp, err := c.do(http.MethodPost, "/admin/stations/"+username+"/relay/start", body)
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

func (c *Client) StopRelay(username string) error {
	resp, err := c.do(http.MethodPost, "/admin/stations/"+username+"/relay/stop", nil)
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

// ClearSegments flushes the server-side HLS segment buffer for a station.
// Call this before starting a new local broadcast so HLS clients resync.
func (c *Client) ClearSegments(username string) error {
	resp, err := c.do(http.MethodPost, "/admin/stations/"+username+"/hls/clear", nil)
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}
