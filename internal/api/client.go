package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type QuotaResponse struct {
	Subscription struct {
		Limit    int       `json:"limit"`
		Requests int       `json:"requests"`
		RenewsAt time.Time `json:"renewsAt"`
	} `json:"subscription"`
}

type Client struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
}

func NewClient(apiKey string) *Client {
	return &Client{
		apiKey:  apiKey,
		baseURL: "https://api.synthetic.new",
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *Client) GetQuotas(ctx context.Context) (*QuotaResponse, error) {
	url := fmt.Sprintf("%s/v2/quotas", c.baseURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var quota QuotaResponse
	if err := json.NewDecoder(resp.Body).Decode(&quota); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	return &quota, nil
}
