package backend

import (
	"crypto/tls"
	"fmt"
	"net/smtp"

	"github.com/jordan-wright/email"
)

// Email - alias for jordan-wright/email Email struct
type Email = email.Email

// EmailSettings - email configuration and setup to send authtokens and stuff
var (
	EmailTLSConfig *tls.Config
	EmailPool      *email.Pool
	SMTPAuth       smtp.Auth
	EmailConf      = struct {
		Connections int
		Address     string
		Server      string
		Port        string
		FromTxt     string
		Email       string
		Password    string
	}{}
)

// startEmailer - initialize the blog's email configuration
func startEmailer() {
	SMTPAuth = smtp.PlainAuth("", EmailConf.Email, EmailConf.Password, EmailConf.Server)
	// TLS config
	EmailTLSConfig = &tls.Config{
		InsecureSkipVerify: true,
		ServerName:         EmailConf.Server,
	}

	/*
				// PushedEmail struct with email prepped for sending and an error channel to see how it went
				type PushedEmail struct {
					Email *Email
					Error chan error
				}

				// PushEmail add emails in this channel and they will be sent
				var PushEmail = make(chan *PushedEmail)

						pool, err := email.NewPool(EmailConf.Address, EmailConf.Connections, SMTPAuth, EmailTLSConfig)
						if err != nil {
							panic(err)
						}
						EmailPool = pool

						for i := 0; i < EmailConf.Connections; i++ {
							go func() {
								for pushed := range PushEmail {

									if pushed.Email.From == "" {
										pushed.Email.From = EmailConf.FromTxt
									}

									if &pushed.Email.Headers == nil {
										pushed.Email.Headers = textproto.MIMEHeader{}
									}

									err := EmailPool.Send(pushed.Email, 10*time.Second)
									if err != nil {
										if DevMode {
											fmt.Println("trouble sending email to ", pushed.Email.To, " : ", err)
										}
									} else if DevMode {
										fmt.Println("email sent to ", pushed.Email.To, " successfully!")
									}
									pushed.Error <- err
								}
							}()
						}

					func stopEmailer() {
			//	EmailPool.Close()
		}

		// SendEmail send an *Email (with the correct details of course)
		func SendEmail(mail *Email) error {
			pe := &PushedEmail{Email: mail}
			pe.Error = make(chan error)
			PushEmail <- pe
			return <-pe.Error
		}
	*/

	fmt.Println(`SMTP Emailer Started`)
}

func stopEmailer() {
	//	EmailPool.Close()
}

// SendEmail send an *Email (with the correct details of course)
func SendEmail(mail *Email) error {
	return mail.SendWithTLS(EmailConf.Address, SMTPAuth, EmailTLSConfig)
}
