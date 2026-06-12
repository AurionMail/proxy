package routing

import (
    "bytes"
    "context"
    "encoding/json"
    "errors"
    "fmt"
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
    var identity string
    var pubKey string
    final := make(map[string]struct{})

    for _, rcpt := range rcpts {
        res, err := c.callResolve(rcpt)
        if err != nil {
            return nil, err
        }

        if identity == "" {
            identity = res.IdentityEmail
            pubKey = res.PublicKey
        } else if identity != res.IdentityEmail {
            return nil, ErrMixedIdentities
        }

        for _, r := range res.Recipients {
            final[r] = struct{}{}
        }
    }

    finalRcpts := make([]string, 0, len(final))
    for r := range final {
        finalRcpts = append(finalRcpts, r)
    }

    return &MessageContext{
        From:          from,
        OriginalRcpts: rcpts,
        IdentityEmail: identity,
        PublicKey:     pubKey,
        FinalRcpts:    finalRcpts,
    }, nil
}

func (c *Client) callResolve(rcpt string) (*ResolveResponse, error) {
    reqBody, err := json.Marshal(ResolveRequest{Rcpt: rcpt})
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
        return nil, err
    }
    defer resp.Body.Close()

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
