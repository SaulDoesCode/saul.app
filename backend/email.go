package backend

import (
	"crypto/tls"
	"errors"
	"fmt"
	"log"
	"net"
	"net/mail"
	"net/smtp"
	"net/textproto"

	"github.com/SaulDoesCode/saul.app/backend/email"
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
		// InsecureSkipVerify: DevMode,
		ServerName: EmailConf.Server,
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

type Mail struct {
	From string
}

func emailer() {
	from := mail.Address{"", "username@example.tld"}
	to := mail.Address{"", "username@anotherexample.tld"}

	subj := "This is the email subject"
	body := "This is an example body.\n With two lines."

	// Setup headers
	headers := make(map[string]string)
	headers["From"] = from.String()
	headers["To"] = to.String()
	headers["Subject"] = subj

	// Setup message
	message := ""
	for k, v := range headers {
		message += fmt.Sprintf("%s: %s\r\n", k, v)
	}
	message += "\r\n" + body

	// Connect to the SMTP Server
	servername := "smtp.example.tld:465"

	host, _, _ := net.SplitHostPort(servername)

	auth := smtp.PlainAuth("", "username@example.tld", "password", host)

	// TLS config
	tlsconfig := &tls.Config{
		InsecureSkipVerify: true,
		ServerName:         host,
	}

	c, err := smtp.Dial(servername)
	if err != nil {
		log.Panic(err)
	}

	c.StartTLS(tlsconfig)

	// Auth
	if err = c.Auth(auth); err != nil {
		log.Panic(err)
	}

	// To && From
	if err = c.Mail(from.Address); err != nil {
		log.Panic(err)
	}

	if err = c.Rcpt(to.Address); err != nil {
		log.Panic(err)
	}

	// Data
	w, err := c.Data()
	if err != nil {
		log.Panic(err)
	}

	_, err = w.Write([]byte(message))
	if err != nil {
		log.Panic(err)
	}

	err = w.Close()
	if err != nil {
		log.Panic(err)
	}

	c.Quit()
}
