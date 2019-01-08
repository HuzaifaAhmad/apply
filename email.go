package main

import (
	"os"

	mailgun "github.com/mailgun/mailgun-go"
)

type mailgunConfig struct {
	domain    string
	apiKey    string
	publicKey string
}

//MailRequest holds info about email
type MailRequest struct {
	from        string
	title       string
	htmlMessage string
	to          []string
}

var config mailgunConfig

func init() {
	config.domain = os.Getenv("MAIL_DOMAIN")
	config.apiKey = os.Getenv("MAIL_API")
	config.publicKey = os.Getenv("MAIL_PUBLIC")
}

//NewMailRequest creates a request
func NewMailRequest(from string, title string, htmlMessage string, receivers []string) *MailRequest {
	return &MailRequest{
		from:        from,
		title:       title,
		htmlMessage: htmlMessage,
		to:          receivers,
	}

}

// SendMail is used to send message, it will ask user about title, htmlMessage, textMessage and list of recipient
func (mailRequest *MailRequest) SendMail() (bool, error) {

	// NewMailGun creates a new client instance.
	mg := mailgun.NewMailgun(config.domain, config.apiKey, config.publicKey)

	// Create message
	message := mg.NewMessage(
		mailRequest.from,
		mailRequest.title,
		"",
		mailRequest.to...,
	)
	message.SetHtml(mailRequest.htmlMessage)

	// send message and get result
	if _, _, err := mg.Send(message); err != nil {
		return false, err
	}

	return true, nil
}
