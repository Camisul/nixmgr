package cloudflare

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

const apiBase = "https://api.cloudflare.com/client/v4"

type Client struct {
	token  string
	zoneID string
	http   *http.Client
}

func NewClient() (*Client, error) {
	token := os.Getenv("CLOUDFLARE_API_TOKEN")
	if token == "" {
		return nil, fmt.Errorf("CLOUDFLARE_API_TOKEN environment variable is not set")
	}
	zoneID := os.Getenv("CLOUDFLARE_ZONE_ID")
	if zoneID == "" {
		return nil, fmt.Errorf("CLOUDFLARE_ZONE_ID environment variable is not set")
	}
	return &Client{
		token:  token,
		zoneID: zoneID,
		http:   &http.Client{},
	}, nil
}

type dnsRecord struct {
	Type    string `json:"type"`
	Name    string `json:"name"`
	Content string `json:"content"`
	TTL     int    `json:"ttl"`
	Proxied bool   `json:"proxied"`
}

type apiResponse struct {
	Success bool              `json:"success"`
	Errors  []json.RawMessage `json:"errors"`
}

// CreateARecord creates an A record: <name>.<domain> -> ip
func (c *Client) CreateARecord(name, domain, ip string) error {
	fqdn := fmt.Sprintf("%s.%s", name, domain)

	record := dnsRecord{
		Type:    "A",
		Name:    fqdn,
		Content: ip,
		TTL:     1, // automatic
		Proxied: false,
	}

	body, err := json.Marshal(record)
	if err != nil {
		return fmt.Errorf("marshaling DNS record: %w", err)
	}

	url := fmt.Sprintf("%s/zones/%s/dns_records", apiBase, c.zoneID)
	req, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("calling Cloudflare API: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading response: %w", err)
	}

	var apiResp apiResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return fmt.Errorf("parsing response: %w", err)
	}

	if !apiResp.Success {
		return fmt.Errorf("cloudflare API error: %s", string(respBody))
	}

	return nil
}
