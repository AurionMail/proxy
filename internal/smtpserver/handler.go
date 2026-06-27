package smtpserver

import (
	"bytes"
	"io"
	"io/ioutil"

	"aurion/proxy/internal/config"
	"aurion/proxy/internal/encryption"
	"aurion/proxy/internal/queue"
	"aurion/proxy/internal/routing"

	"github.com/emersion/go-msgauth/dkim"
	smtp "github.com/emersion/go-smtp"
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

	// 1) Authentification minimale : DKIM
	if err := s.verifyDKIM(raw); err != nil {
		return &smtp.SMTPError{
			Code:         550,
			EnhancedCode: smtp.EnhancedCode{5, 7, 1},
			Message:      err.Error(),
		}
	}

	// 2) Routing
	ctx, err := s.routingClient.Resolve(s.from, s.rcpts)
	if err != nil {
		return err
	}

	// 3) Détection PGP + chiffrement éventuel
	final := raw
	if !encryption.IsPGPEncrypted(raw) {
		// MODIFICATION ICI : On passe la map ctx.PublicKeys au lieu de la string unique ctx.PublicKey
		final, err = encryption.Encrypt(ctx.PublicKeys, raw)
		if err != nil {
			return err
		}
	}

	// 4) Enqueue pour forwarding SMTP interne
	return queue.Enqueue(ctx, final)
}

func (s *Session) Reset() {
	s.from = ""
	s.rcpts = nil
}

func (s *Session) Logout() error {
	return nil
}

// --- DKIM uniquement pour l’instant ---

func (s *Session) verifyDKIM(raw []byte) error {
	res, err := dkim.Verify(bytes.NewReader(raw))
	if err != nil {
		// Signature présente mais invalide = rejet
		return err
	}

	if res == nil {
		// Aucune signature DKIM = ok
		return nil
	}

	// Signature présente valide = ok
	return nil
}
