package main

import (
	"fmt"
	"net/mail"

	"github.com/Meha555/go-pipeline/notify/email"
)

func main() {
	e := &email.Email{
		To:      mail.Address{Name: "abc", Address: "huangzy@sense.com.cn"},
		From:    mail.Address{Name: "def", Address: "huangzy@sense.com.cn"},
		Subject: "Test Email",
		Body:    "This is email body",
	}
	notifier := email.Notifier{
		Password:       "",
		SmtpSendServer: "exmail.qq.com",
		SmtpSendPort:   1,
	}
	notifier.SetEmail(e)
	err := notifier.Notify()
	if err != nil {
		fmt.Println("Error:", err)
	}
}
