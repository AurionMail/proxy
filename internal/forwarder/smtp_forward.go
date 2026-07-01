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

	log.Printf("[DEBUG] Dialing upstream SMTP server at %s...", f.Addr)

	// 1) Connexion réseau TCP
	conn, err := net.DialTimeout("tcp", f.Addr, 15*time.Second)
	if err != nil {
		log.Printf("[ERROR] Dial failed: %v", err)
		return fmt.Errorf("forward: dial timeout: %w", err)
	}

	log.Printf("[DEBUG] TCP Connection established, initializing SMTP client...")

	// 2) Initialisation du client SMTP Go
	host, _, _ := net.SplitHostPort(f.Addr)

	// Définissons un timeout de lecture pour éviter le freeze si Stalwart reste muet
	conn.SetDeadline(time.Now().Add(15 * time.Second))

	c, err := smtp.NewClient(conn, host)
	if err != nil {
		conn.Close()
		log.Printf("[ERROR] SMTP client init failed: %v", err)
		return fmt.Errorf("forward: smtp client init: %w", err)
	}

	// On retire le deadline pour le reste de la transaction
	conn.SetDeadline(time.Time{})
	defer c.Close()

	log.Printf("[DEBUG] SMTP Handshake success, processing mail...")

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
