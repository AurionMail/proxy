package smtpserver

import (
	"bytes"
	"fmt"
	"io"

	"aurion/proxy/internal/config"
	"aurion/proxy/internal/encryption"
	"aurion/proxy/internal/queue"
	"aurion/proxy/internal/routing"

	"mime/multipart"
	"net/mail"
	"net/textproto"
	"strings"

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
	raw, err := io.ReadAll(r)
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

	// 3) Détection PGP + Chiffrement PGP/MIME Global
	final := raw
	if !encryption.IsPGPEncrypted(raw) {
		msg, err := mail.ReadMessage(bytes.NewReader(raw))
		if err != nil {
			return err
		}

		// Récupération du corps d'origine (contenant texte brut, HTML et/ou pièces jointes)
		origBody, err := io.ReadAll(msg.Body)
		if err != nil {
			return err
		}

		// Chiffrement global du corps (le résultat doit être au format ASCII-Armored)
		encryptedPayload, err := encryption.Encrypt(ctx.PublicKeys, origBody)
		if err != nil {
			return err
		}

		// Génération d'un boundary unique pour l'enveloppe PGP/MIME
		var buf bytes.Buffer
		mw := multipart.NewWriter(&buf)
		boundary := mw.Boundary()

		// Écriture de la Partie 1 : Déclaration PGP/MIME version identification
		h1 := make(textproto.MIMEHeader)
		h1.Set("Content-Type", "application/pgp-encrypted")
		h1.Set("Content-Description", "PGP/MIME version identification")
		p1, err := mw.CreatePart(h1)
		if err != nil {
			return err
		}
		p1.Write([]byte("Version: 1\r\n"))

		// Écriture de la Partie 2 : Le bloc de données chiffré
		h2 := make(textproto.MIMEHeader)
		h2.Set("Content-Type", "application/octet-stream; name=\"encrypted.asc\"")
		h2.Set("Content-Description", "OpenPGP encrypted message")
		h2.Set("Content-Disposition", "inline; filename=\"encrypted.asc\"")
		p2, err := mw.CreatePart(h2)
		if err != nil {
			return err
		}
		p2.Write(encryptedPayload)

		// Fermeture du multipart pour insérer le boundary de fin
		mw.Close()

		// Reconstruction finale du message
		// Reconstruction finale du message
		var finalBuf bytes.Buffer

		// On réécrit les en-têtes principaux d'origine en filtrant les anciens paramètres
		for k, v := range msg.Header {
			kl := strings.ToLower(k)
			if kl == "content-type" || kl == "mime-version" || kl == "dkim-signature" {
				continue
			}
			// CORRECTION ICI : Écriture directe et optimisée dans le buffer
			fmt.Fprintf(&finalBuf, "%s: %s\r\n", k, strings.Join(v, ", "))
		}

		// Injection des nouveaux en-têtes requis pour le protocole PGP/MIME
		finalBuf.WriteString("MIME-Version: 1.0\r\n")
		// CORRECTION ICI AUSSI : Remplacement du WriteString(fmt.Sprintf(...))
		fmt.Fprintf(&finalBuf, "Content-Type: multipart/encrypted; boundary=\"%s\"; protocol=\"application/pgp-encrypted\"\r\n", boundary)
		finalBuf.WriteString("\r\n")

		// Injection du corps enveloppé
		finalBuf.Write(buf.Bytes())
		final = finalBuf.Bytes()
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
