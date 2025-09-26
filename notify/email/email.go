package email

import (
	"fmt"
	"net/mail"
	"net/smtp"
	"strings"
)

// Email 这里不支持附件
type Email struct {
	from    *mail.Address
	to      []*mail.Address
	cc      []*mail.Address
	subject string
	body    []byte

	b      strings.Builder
	header string
}

type Builder struct {
	email *Email
}

func NewBuilder() *Builder {
	return &Builder{
		email: &Email{},
	}
}

func (e *Builder) From(addr *mail.Address) *Builder {
	e.email.from = addr
	e.email.b.WriteString(fmt.Sprintf("From: %s\r\n", e.email.from.String()))
	return e
}

func (e *Builder) To(addr []*mail.Address) *Builder {
	e.email.to = addr
	e.email.b.WriteString(fmt.Sprintf("To: %s\r\n", addr2Str(e.email.to)))
	return e
}

func (e *Builder) Cc(addr []*mail.Address) *Builder {
	e.email.cc = addr
	e.email.b.WriteString(fmt.Sprintf("Cc: %s\r\n", addr2Str(e.email.cc)))
	return e
}

func (e *Builder) Subject(subject string) *Builder {
	e.email.subject = subject
	e.email.b.WriteString(fmt.Sprintf("Subject: %s\r\n", e.email.subject))
	return e
}

func (e *Builder) Body(body []byte) *Builder {
	e.email.body = body
	return e
}

func (e *Builder) Build() *Email {
	e.email.b.WriteString("\r\n")
	e.email.header = e.email.b.String()
	e.email.b.Reset()
	return e.email
}

func (e *Email) Header() string {
	return e.header
}

func (e *Email) Message() string {
	e.b.Reset()
	e.b.WriteString(e.header)
	e.b.Write(e.body)
	return e.b.String()
}

func (e *Email) AllRecipients() []*mail.Address {
	allRecipients := append(e.to, e.cc...)
	return allRecipients
}

type Sender struct {
	SmtpServer string
	SmtpPort   int
	Password   string

	addr string
}

func (n *Sender) Init() error {
	n.addr = fmt.Sprintf("%s:%d", n.SmtpServer, n.SmtpPort)
	return nil
}

func (n *Sender) Send(email *Email) error {
	if email == nil {
		return fmt.Errorf("no email set")
	}
	if n.addr == "" {
		n.Init()
	}

	// 这里的必须是不带<>的地址
	auth := smtp.PlainAuth("", email.from.Address, n.Password, n.SmtpServer)
	// 这里的必须是不带<>的地址
	err := smtp.SendMail(n.addr, auth, email.from.Address, addr2Strlist(email.AllRecipients()), []byte(email.Message()))
	if err != nil {
		return fmt.Errorf("send email error: %w", err)
	}
	return nil
}

func addr2Str(addrs []*mail.Address) string {
	addrList := addr2Strlist(addrs)
	return strings.Join(addrList, ", ")
}

func addr2Strlist(addrs []*mail.Address) []string {
	var addrList []string
	for _, addr := range addrs {
		addrList = append(addrList, addr.Address)
	}
	return addrList
}
