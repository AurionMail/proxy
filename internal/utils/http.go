package utils

import (
    "net"
    "net/http"
    "time"
)

func NewHTTPClient(timeout time.Duration) *http.Client {
    transport := &http.Transport{
        MaxIdleConns:        100,
        MaxIdleConnsPerHost: 10,
        IdleConnTimeout:     90 * time.Second,
        DisableCompression:  false,
        DialContext: (&net.Dialer{
            Timeout:   2 * time.Second,
            KeepAlive: 30 * time.Second,
        }).DialContext,
    }

    return &http.Client{
        Timeout:   timeout,
        Transport: transport,
    }
}
