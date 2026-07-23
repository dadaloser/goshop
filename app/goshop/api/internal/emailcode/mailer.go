package emailcode

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"net/smtp"
	"strconv"

	"goshop/app/pkg/options"
)

type Sender interface {
	Send(ctx context.Context, to, purpose string) error
}

type SMTPSender struct {
	opts  *options.EmailOptions
	store Store
}

func NewSMTPSender(opts *options.EmailOptions, store Store) *SMTPSender {
	return &SMTPSender{opts: opts, store: store}
}

func (s *SMTPSender) Send(ctx context.Context, to, purpose string) error {
	if s == nil || s.opts == nil || !s.opts.Enabled {
		return fmt.Errorf("email service is disabled")
	}
	code, err := randomCode()
	if err != nil {
		return err
	}
	if err = s.store.Issue(ctx, to, purpose, code, s.opts.CodeTTL, s.opts.SendInterval); err != nil {
		return err
	}
	address := s.opts.Host + ":" + strconv.Itoa(s.opts.Port)
	auth := smtp.PlainAuth("", s.opts.Username, s.opts.Password, s.opts.Host)
	message := []byte("To: " + to + "\r\nSubject: Goshop verification code\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\nYour verification code is " + code + ". It expires in " + s.opts.CodeTTL.String() + ".\r\n")
	if err = smtp.SendMail(address, auth, s.opts.From, []string{to}, message); err != nil {
		return fmt.Errorf("send verification email: %w", err)
	}
	return nil
}

func randomCode() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		return "", fmt.Errorf("generate email code: %w", err)
	}
	return fmt.Sprintf("%06d", n.Int64()), nil
}
