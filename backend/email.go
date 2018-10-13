package backend

import (
	"crypto/tls"
	"errors"
	"fmt"
	"net/smtp"
	"net/textproto"

	"github.com/jordan-wright/email"
)

// Email - alias for jordan-wright/email Email struct
type Email = email.Email

// EmailSettings - email configuration and setup to send authtokens and stuff
var (
	EmailTLSConfig *tls.Config
	SMTPAuth       smtp.Auth
	EmailConf      = struct {
		Address  string
		Server   string
		Port     string
		FromTxt  string
		Email    string
		Password string
	}{}
	// ErrInvalidEmail bad email
	ErrInvalidEmail = errors.New(`Invalid Email Address`)
)

// startEmailer - initialize the blog's email configuration
func startEmailer() {
	SMTPAuth = smtp.PlainAuth("", EmailConf.Email, EmailConf.Password, EmailConf.Server)
	// TLS config
	EmailTLSConfig = &tls.Config{
		InsecureSkipVerify: true,
		ServerName:         EmailConf.Server,
	}

	fmt.Println(`SMTP Emailer Started`)
}

func stopEmailer() {
	//	EmailPool.Close()
}

// SendEmail send an *Email (with the correct details of course)
func SendEmail(mail *Email) error {
	if len(mail.From) == 0 {
		mail.From = EmailConf.FromTxt
	}

	if &mail.Headers == nil {
		mail.Headers = textproto.MIMEHeader{}
	}

	if !validEmail(mail.To[0]) {
		return ErrInvalidEmail
	}
	return mail.SendWithTLS(EmailConf.Address, SMTPAuth, EmailTLSConfig)
}
