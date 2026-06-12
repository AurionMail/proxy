package forwarder

import (
    "fmt"
    "net/smtp"
)

type SMTPForwarder struct {
    Addr string // ex: "mail.internal:10025"
}

func NewSMTPForwarder(addr string) *SMTPForwarder {
    return &SMTPForwarder{Addr: addr}
}

func (f *SMTPForwarder) Forward(job *ForwardJob) error {
    c, err := smtp.Dial(f.Addr)
    if err != nil {
        return fmt.Errorf("forward: dial: %w", err)
    }
    defer c.Close()

    // MAIL FROM
    if err := c.Mail(job.Ctx.From); err != nil {
        return fmt.Errorf("forward: MAIL FROM: %w", err)
    }

    // RCPT TO
    for _, rcpt := range job.Ctx.FinalRcpts {
        if err := c.Rcpt(rcpt); err != nil {
            return fmt.Errorf("forward: RCPT TO %s: %w", rcpt, err)
        }
    }

    // DATA
    wc, err := c.Data()
    if err != nil {
        return fmt.Errorf("forward: DATA: %w", err)
    }

    if _, err := wc.Write(job.Message); err != nil {
        return fmt.Errorf("forward: write: %w", err)
    }

    if err := wc.Close(); err != nil {
        return fmt.Errorf("forward: close: %w", err)
    }

    return c.Quit()
}
