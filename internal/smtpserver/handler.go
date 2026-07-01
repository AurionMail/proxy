package smtpserver

import (
	"bytes"
	"fmt"
	"io"

	"aurion/proxy/internal/config"
	"aurion/proxy/internal/encryption"
	"aurion/proxy/internal/queue"
	"aurion/proxy/internal/routing"

	"mime"
	"mime/multipart"
	"net/mail"
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

	// 3) Détection PGP + Chiffrement sélectif
	final := raw
	if !encryption.IsPGPEncrypted(raw) {
		msg, err := mail.ReadMessage(bytes.NewReader(raw))
		if err != nil {
			return err
		}

		contentType := msg.Header.Get("Content-Type")
		mediaType, params, _ := mime.ParseMediaType(contentType)

		// CAS 1 : L'e-mail a des pièces jointes (Multipart)
		if strings.HasPrefix(mediaType, "multipart/") {
			mr := multipart.NewReader(msg.Body, params["boundary"])
			var newBody bytes.Buffer
			mw := multipart.NewWriter(&newBody)

			// On copie le boundary d'origine pour ne pas casser les headers existants
			mw.SetBoundary(params["boundary"])

			for {
				part, err := mr.NextPart()
				if err == io.EOF {
					break
				}
				if err != nil {
					return err
				}

				// On lit les headers de cette partie
				partContentType := part.Header.Get("Content-Type")
				partMediaType, _, _ := mime.ParseMediaType(partContentType)

				// Crée la nouvelle partie MIME avec les mêmes headers
				partWriter, err := mw.CreatePart(part.Header)
				if err != nil {
					return err
				}

				partData, err := io.ReadAll(part)
				if err != nil {
					return err
				}

				// IMPORTANT : On ne chiffre QUE le texte.
				// Les pièces jointes (application/*, image/*) restent intactes.
				if strings.HasPrefix(partMediaType, "text/") {
					encryptedText, err := encryption.Encrypt(ctx.PublicKeys, partData)
					if err != nil {
						return err
					}
					partWriter.Write(encryptedText)
				} else {
					// C'est une pièce jointe, on l'écrit telle quelle sans y toucher
					partWriter.Write(partData)
				}
			}
			mw.Close()

			// On reconstruit l'e-mail complet (Headers principaux + Nouveau Corps Multipart)
			var buf bytes.Buffer
			for k, v := range msg.Header {
				buf.WriteString(fmt.Sprintf("%s: %s\r\n", k, strings.Join(v, ", ")))
			}
			buf.WriteString("\r\n")
			buf.Write(newBody.Bytes())
			final = buf.Bytes()

		} else {
			// CAS 2 : Émail simple (uniquement du texte, pas de pièce jointe)
			bodyData, err := io.ReadAll(msg.Body)
			if err != nil {
				return err
			}

			encryptedBody, err := encryption.Encrypt(ctx.PublicKeys, bodyData)
			if err != nil {
				return err
			}

			var buf bytes.Buffer
			for k, v := range msg.Header {
				buf.WriteString(fmt.Sprintf("%s: %s\r\n", k, strings.Join(v, ", ")))
			}
			buf.WriteString("\r\n")
			buf.Write(encryptedBody)
			final = buf.Bytes()
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
