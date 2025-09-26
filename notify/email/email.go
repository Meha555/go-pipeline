package email

import (
	"fmt"
	"net/mail"
	"net/smtp"
)

type Email struct {
	To      mail.Address
	From    mail.Address
	Subject string
	Body    string
}

type Notifier struct {
	Password       string
	SmtpSendServer string
	SmtpSendPort   int

	email *Email
}

func (n *Notifier) SetEmail(email *Email) {
	n.email = email
}

func (n *Notifier) Notify() error {
	if n.email == nil {
		return fmt.Errorf("no email set")
	}
	auth := smtp.PlainAuth("", n.email.From.Address, n.Password, n.SmtpSendServer)
	err := smtp.SendMail(fmt.Sprintf("%s:%d", n.SmtpSendServer, n.SmtpSendPort), auth, n.email.From.Address, []string{n.email.To.Address}, []byte(n.email.Body))
	if err != nil {
		return err
	}
	return nil
}
