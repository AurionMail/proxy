package smtpserver

import (
    smtp "github.com/emersion/go-smtp"

    "aurion/proxy/internal/routing"
    "aurion/proxy/internal/encryption"
    "aurion/proxy/internal/queue"
    "aurion/proxy/internal/config"
	"io"
	"io/ioutil"
)

type Backend struct {
    routingClient *routing.Client
}

func NewBackend(cfg *config.Config) *Backend {
    return &Backend{
        routingClient: routing.NewClient(cfg.RoutingURL, cfg.RoutingTimeout),
    }
}

func (b *Backend) NewSession(c *smtp.Conn) (smtp.Session, error) {
    return &Session{
        routingClient: b.routingClient,
    }, nil
}

type Session struct {
    from          string
    rcpts         []string
    routingClient *routing.Client
}

func (s *Session) Mail(from string, opts *smtp.MailOptions) error {
    s.from = from
    return nil
}

func (s *Session) Rcpt(to string, opts *smtp.RcptOptions) error {
    s.rcpts = append(s.rcpts, to)
    return nil
}

func (s *Session) Data(r io.Reader) error {
    raw, err := ioutil.ReadAll(r)
    if err != nil {
        return err
    }

    // 1. Routing
    ctx, err := s.routingClient.Resolve(s.from, s.rcpts)
    if err != nil {
        return err
    }

    // 2. Détection PGP
    final := raw
    if !encryption.IsPGPEncrypted(raw) {
        // 3. Chiffrement fallback
        final, err = encryption.Encrypt(ctx.PublicKey, raw)
        if err != nil {
            return err
        }
    }

    // 4. Enqueue pour forwarding SMTP interne
    return queue.Enqueue(ctx, final)
}

func (s *Session) Reset() {}

func (s *Session) Logout() error {
    return nil
}
