package forwarder

import (
	"fmt"
	"log"
	"net"
	"net/smtp"
	"time"
)

type SMTPForwarder struct {
	Addr string // ex: "127.0.0.1:10025"
}

func NewSMTPForwarder(addr string) *SMTPForwarder {
	return &SMTPForwarder{Addr: addr}
}

func (f *SMTPForwarder) Forward(job *ForwardJob) error {
	log.Printf("[DEBUG] Dialing upstream SMTP server at %s...", f.Addr)

	// 1) Connexion réseau TCP avec un Timeout de 10 secondes
	conn, err := net.DialTimeout("tcp", f.Addr, 10*time.Second)
	if err != nil {
		log.Printf("[INFO] Timeout or error connecting to upstream SMTP server: %v", err)
		return fmt.Errorf("forward: dial timeout: %w", err)
	}

	// 2) Initialisation du client SMTP Go sur la connexion TCP
	host, _, _ := net.SplitHostPort(f.Addr)
	c, err := smtp.NewClient(conn, host)
	if err != nil {
		conn.Close()
		log.Printf("[ERROR] Timeout or error connecting to upstream SMTP server: %v", err)
		return fmt.Errorf("forward: smtp client init: %w", err)
	}
	defer c.Close()

	// 3) Commande MAIL FROM
	if err := c.Mail(job.Ctx.From); err != nil {
		log.Printf("[ERROR] Failed to send MAIL FROM: %v", err)
		return fmt.Errorf("forward: MAIL FROM: %w", err)
	}

	// 4) Commande RCPT TO pour chaque destinataire
	for _, rcpt := range job.Ctx.OriginalRcpts {
		if err := c.Rcpt(rcpt); err != nil {
			log.Printf("[ERROR] Failed to send RCPT TO %s: %v", rcpt, err)
			return fmt.Errorf("forward: RCPT TO %s: %w", rcpt, err)
		}
	}

	// 5) Commande DATA
	wc, err := c.Data()
	if err != nil {
		log.Printf("[ERROR] Failed to initialize DATA: %v", err)
		return fmt.Errorf("forward: DATA init: %w", err)
	}

	// Écriture du corps du mail (potentiellement chiffré par le proxy)
	if _, err := wc.Write(job.Message); err != nil {
		log.Printf("[ERROR] Failed to write DATA: %v", err)
		wc.Close()
		return fmt.Errorf("forward: write data: %w", err)
	}

	if err := wc.Close(); err != nil {
		log.Printf("[ERROR] Failed to close data stream: %v", err)
		return fmt.Errorf("forward: close data stream: %w", err)
	}

	// 6) Clôture propre de la session SMTP
	if err := c.Quit(); err != nil {
		log.Printf("[ERROR] Failed to quit SMTP session: %v", err)
		return fmt.Errorf("forward: quit session: %w", err)
	}

	log.Printf("[INFO] FORWARDED SUCCESS: From: %s, To: %v, Size: %d bytes", job.Ctx.From, job.Ctx.OriginalRcpts, len(job.Message))
	return nil
}
