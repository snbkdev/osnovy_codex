package mail

import (
	"context"
	"crypto/tls"
	"fmt"
	"mime"
	"net"
	stdmail "net/mail"
	"net/smtp"
	"strings"
	"time"
	"uptime_app/backend/internal/auth"
)

type Sender interface {
	Send(context.Context, auth.Letter) error
}
type SMTP struct {
	Addr, User, Password, From string
	AllowInsecure              bool
}

func (s SMTP) Send(ctx context.Context, l auth.Letter) error {
	host, _, err := net.SplitHostPort(s.Addr)
	if err != nil {
		return err
	}
	for _, value := range []string{s.From, l.To, l.Subject} {
		if strings.ContainsAny(value, "\r\n") {
			return fmt.Errorf("invalid mail header")
		}
	}
	from, err := stdmail.ParseAddress(s.From)
	if err != nil {
		return err
	}
	to, err := stdmail.ParseAddress(l.To)
	if err != nil {
		return err
	}
	conn, err := (&net.Dialer{Timeout: 5 * time.Second}).DialContext(ctx, "tcp", s.Addr)
	if err != nil {
		return err
	}
	defer conn.Close()
	deadline := time.Now().Add(10 * time.Second)
	if d, ok := ctx.Deadline(); ok && d.Before(deadline) {
		deadline = d
	}
	conn.SetDeadline(deadline)
	client, err := smtp.NewClient(conn, host)
	if err != nil {
		return err
	}
	defer client.Close()
	if ok, _ := client.Extension("STARTTLS"); ok {
		if err := client.StartTLS(&tls.Config{ServerName: host, MinVersion: tls.VersionTLS12}); err != nil {
			return err
		}
	} else if !s.AllowInsecure {
		return fmt.Errorf("SMTP requires STARTTLS")
	}
	if s.User != "" {
		if err := client.Auth(smtp.PlainAuth("", s.User, s.Password, host)); err != nil {
			return err
		}
	}
	if err := client.Mail(from.Address); err != nil {
		return err
	}
	if err := client.Rcpt(to.Address); err != nil {
		return err
	}
	writer, err := client.Data()
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(writer, "From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n%s", s.From, l.To, mime.QEncoding.Encode("UTF-8", l.Subject), strings.ReplaceAll(l.Body, "\n", "\r\n"))
	if err != nil {
		writer.Close()
		return err
	}
	if err := writer.Close(); err != nil {
		return err
	}
	return client.Quit()
}
