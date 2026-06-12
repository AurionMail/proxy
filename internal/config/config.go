package config

import (
    "log"
    "os"
    "strconv"
    "time"

    "github.com/joho/godotenv"
)

type Config struct {
    // SMTP proxy
    ListenAddr      string
    Domain          string
    MaxMessageBytes int

    // Routing API
    RoutingURL     string
    RoutingTimeout time.Duration

    // Forwarding SMTP
    ForwardAddr string

    // Queue
    QueueSize   int
    WorkerCount int

    // TLS
    TLSCert string
    TLSKey  string
}

func Load() *Config {
    // Charge .env si présent
    _ = godotenv.Load()

    cfg := &Config{
        ListenAddr:      getEnv("LISTEN_ADDR", ":25"),
        Domain:          getEnv("DOMAIN", "aurion.local"),
        MaxMessageBytes: getEnvInt("MAX_MESSAGE_BYTES", 10<<20),

        RoutingURL:     getEnv("ROUTING_URL", "http://app-core.internal"),
        RoutingTimeout: getEnvDuration("ROUTING_TIMEOUT", 3*time.Second),

        ForwardAddr: getEnv("FORWARD_ADDR", "mail.internal:10025"),

        QueueSize:   getEnvInt("QUEUE_SIZE", 1000),
        WorkerCount: getEnvInt("WORKER_COUNT", 4),

        TLSCert: getEnv("TLS_CERT", "server.crt"),
        TLSKey:  getEnv("TLS_KEY", "server.key"),
    }

    return cfg
}

func getEnv(key, def string) string {
    if v := os.Getenv(key); v != "" {
        return v
    }
    return def
}

func getEnvInt(key string, def int) int {
    if v := os.Getenv(key); v != "" {
        i, err := strconv.Atoi(v)
        if err != nil {
            log.Fatalf("invalid int for %s: %v", key, err)
        }
        return i
    }
    return def
}

func getEnvDuration(key string, def time.Duration) time.Duration {
    if v := os.Getenv(key); v != "" {
        d, err := time.ParseDuration(v)
        if err != nil {
            log.Fatalf("invalid duration for %s: %v", key, err)
        }
        return d
    }
    return def
}
