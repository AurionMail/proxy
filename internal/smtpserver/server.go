package smtpserver

import (
    "crypto/tls"
    "log"
    "net"

    smtp "github.com/emersion/go-smtp"
    "aurion/proxy/internal/config"
)

func Start(cfg *config.Config) {
    be := NewBackend(cfg)

    s := smtp.NewServer(be)
    s.Addr = cfg.ListenAddr
    s.Domain = cfg.Domain
    s.MaxMessageBytes = int64(cfg.MaxMessageBytes)
    s.MaxRecipients = 100
    s.AllowInsecureAuth = false

    cert, err := tls.LoadX509KeyPair(cfg.TLSCert, cfg.TLSKey)
    if err != nil {
        log.Fatal(err)
    }
    s.TLSConfig = &tls.Config{
        Certificates: []tls.Certificate{cert},
        MinVersion:   tls.VersionTLS12,
    }

    l, err := net.Listen("tcp", s.Addr)
    if err != nil {
        log.Fatal(err)
    }

    log.Printf("SMTP Proxy listening on %s", s.Addr)
    if err := s.Serve(l); err != nil {
        log.Fatal(err)
    }
}
