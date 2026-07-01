package routing

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"
)

var (
	ErrMixedIdentities = errors.New("routing: mixed identities in RCPT TO")
)

type Client struct {
	baseURL    string
	httpClient *http.Client
}

func NewClient(baseURL string, timeout time.Duration) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

func (c *Client) Resolve(from string, rcpts []string) (*MessageContext, error) {
	pubKeys := make(map[string]string)

	for _, rcpt := range rcpts {
		res, err := c.callResolve(rcpt)
		if err != nil {
			return nil, err
		}

		// On associe chaque destinataire à sa clé publique trouvée
		if res.PublicKey != "" {
			pubKeys[rcpt] = res.PublicKey
		}
	}

	return &MessageContext{
		From:          from,
		OriginalRcpts: rcpts,
		PublicKeys:    pubKeys,
	}, nil
}

func (c *Client) callResolve(rcpt string) (*ResolveResponse, error) {
	log.Printf("[DEBUG] Resolving routing for for recipient: %s", rcpt)
	reqBody, err := json.Marshal(ResolveRequest{Email: rcpt})
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), c.httpClient.Timeout)
	defer cancel()

	url := c.baseURL + "/internal/routing/resolve"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(reqBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		log.Printf("[DEBUG] Erreur 23: %v", err)
		return nil, err
	}
	defer resp.Body.Close()
	log.Printf("[DEBUG] reponse: %v", resp)
	if resp.StatusCode >= 500 {
		return nil, fmt.Errorf("routing: transient error %d", resp.StatusCode)
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("routing: permanent error %d", resp.StatusCode)
	}

	var out ResolveResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}

	return &out, nil
}
