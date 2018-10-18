package backend

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"math"
	"math/big"
	"net/smtp"
	"os"
	"time"

	"github.com/SaulDoesCode/mailyak"
	"github.com/driusan/dkim"
)

// EmailSettings - email configuration and setup to send authtokens and stuff
var (
	SMTPAuth      smtp.Auth
	DKIMSignature dkim.Signature
	EmailConf     = struct {
		Address  string
		Server   string
		Port     string
		FromName string
		Email    string
		Password string
	}{}
	PrivateDKIMkey *rsa.PrivateKey
	// ErrInvalidEmail bad email
	ErrInvalidEmail = errors.New(`Invalid Email Address`)
)

// startEmailer - initialize the blog's email configuration
func startEmailer() {
	SMTPAuth = smtp.PlainAuth("", EmailConf.Email, EmailConf.Password, EmailConf.Server)

	block, _ := pem.Decode(DKIMKey)
	key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		fmt.Println("the provided dkim key is bad, fix it")
		panic(err)
	}
	PrivateDKIMkey = key

	DKIMSignature, err = dkim.NewSignature(
		"relaxed/relaxed",
		"mail",
		EmailConf.Server,
		[]string{"From", "Date", "Subject", "To"},
	)
	if err != nil {
		fmt.Println("couldn't build a dkim signature")
		panic(err)
	}

	// Send a little test email
	if DevMode {
		mail := MakeEmail()
		mail.Subject("grimstack.io dev mode startup test")
		mail.To("saulvdw@gmail.com")
		mail.HTML().Set("<h1>You Seeing This?</h1>")
		err = SendEmail(mail)
		if err != nil {
			fmt.Println("emails aren't sending, whats wrong?")
			panic(err)
		}
	}

	fmt.Println(`SMTP Emailer Started`)
}

func stopEmailer() {
	//	EmailPool.Close()
}

// MakeEmail builds a new mailyak instance
func MakeEmail() *mailyak.MailYak {
	return mailyak.New(EmailConf.Address, SMTPAuth)
}

// SendEmail send a dkim signed mailyak email
func SendEmail(m *mailyak.MailYak) error {
	m.From(EmailConf.Email)
	m.FromName(EmailConf.FromName)
	mid, err := generateMessageID()
	if err == nil {
		m.AddHeader("Message-Id", mid)
	}
	return m.SignAndSend(DKIMSignature, PrivateDKIMkey)
}

var maxBigInt = big.NewInt(math.MaxInt64)

// generateMessageID generates and returns a string suitable for an RFC 2822
// compliant Message-ID, e.g.:
// <1444789264909237300.3464.1819418242800517193@DESKTOP01>
//
// The following parameters are used to generate a Message-ID:
// - The nanoseconds since Epoch
// - The calling PID
// - A cryptographically random int64
// - The sending hostname
func generateMessageID() (string, error) {
	t := time.Now().UnixNano()
	pid := os.Getpid()
	rint, err := rand.Int(rand.Reader, maxBigInt)
	if err != nil {
		return "", err
	}
	msgid := fmt.Sprintf("<%d.%d.%d@%s>", t, pid, rint, AppDomain)
	return msgid, nil
}

/*

// Mail container for email fields
type Mail struct {
	From    string
	To      string
	Subject string
	Text    string
	HTML    string
	Headers map[string]string
}


func sendmail(mail *Mail) error {
	useTLS := false
	useStartTLS := true

	if mail.Headers == nil {
		mail.Headers = make(map[string]string)
	}

	if _, ok := mail.Headers["From"]; !ok {
		if len(mail.From) == 0 {
			mail.From = EmailConf.Email
		}
		mail.Headers["From"] = mail.From
	}

	if _, ok := mail.Headers["To"]; !ok {
		mail.Headers["To"] = mail.To
	}
	if _, ok := mail.Headers["Subject"]; !ok {
		mail.Headers["Subject"] = mail.Subject
	}
	if _, ok := mail.Headers["MIME-Version"]; !ok {
		mail.Headers["MIME-Version"] = "1.0"
	}
	if _, ok := mail.Headers["Content-Type"]; !ok {
		mail.Headers["Content-Type"] = "text/html; charset=\"utf-8\""
	}
	mail.Headers["Content-Transfer-Encoding"] = "base64"

	if _, ok := mail.Headers["Message-Id"]; !ok {
		id, err := generateMessageID()
		if err != nil {
			return err
		}
		mail.Headers["Message-Id"] = id
	}

	if _, ok := mail.Headers["Date"]; !ok {
		mail.Headers["Date"] = time.Now().Format(time.RFC1123Z)
	}

	message := ""
	for k, v := range mail.Headers {
		message += fmt.Sprintf("%s: %s\r\n", k, v)
	}
	message += "\r\n" + base64.StdEncoding.EncodeToString([]byte(mail.HTML))

	mailstring, err := dkimEmail(message)
	if err != nil {
		if DevMode {
			fmt.Printf("DKIM signing SendEmail Error: %s\n", err)
		}
		return err
	}

	conn, err := net.Dial("tcp", EmailConf.Address)
	if err != nil {
		if DevMode {
			fmt.Printf("net.Dial SendEmail Error: %s\n", err)
		}
		return err
	}

	// TLS
	tlsconfig := &tls.Config{
		InsecureSkipVerify: DevMode,
		ServerName:         EmailConf.Server,
	}

	if useTLS {
		conn = tls.Client(conn, tlsconfig)
	}

	client, err := smtp.NewClient(conn, EmailConf.Server)
	if err != nil {
		if DevMode {
			fmt.Printf("SMTP-Client SendEmail Error: %s\n", err)
		}
		return err
	}

	if err = client.Hello(EmailConf.Server); err != nil {
		if DevMode {
			fmt.Printf("HELLO SendEmail Error: %s\n", err)
		}
		return err
	}

	hasStartTLS, _ := client.Extension("STARTTLS")
	if useStartTLS && hasStartTLS {
		fmt.Println("STARTTLS ...")
		if err = client.StartTLS(tlsconfig); err != nil {
			if DevMode {
				fmt.Printf("STARTTLS SendEmail Error: %s\n", err)
			}
			return err
		}
	}

	// Set up authentication information.
	auth := smtp.PlainAuth("", EmailConf.Email, EmailConf.Password, EmailConf.Server)

	if ok, _ := client.Extension("AUTH"); ok {
		if err := client.Auth(auth); err != nil {
			if DevMode {
				fmt.Printf("SendEmail Error during AUTH %s\n", err)
			}
			return err
		}
	}

	if err := client.Mail(mail.From); err != nil {
		if DevMode {
			fmt.Printf("From SendEmail Error: %s\n", err)
			fmt.Println("It Was: ", mail.From)
		}
		return err
	}

	if err := client.Rcpt(mail.To); err != nil {
		if DevMode {
			fmt.Printf("To SendEmail Error: %s\n", err)
		}
		return err
	}

	w, err := client.Data()
	if err != nil {
		if DevMode {
			fmt.Printf("Data SendEmail Error: %s\n", err)
		}
		return err
	}

	_, err = w.Write(mailstring)
	if err != nil {
		if DevMode {
			fmt.Printf("Write Mesasge SendEmail Error: %s\n", err)
		}
		return err
	}

	err = w.Close()
	if err != nil {
		if DevMode {
			fmt.Printf("SendEmail Error: %s\n", err)
		}
		return err
	}

	return client.Quit()
}

var maxBigInt = big.NewInt(math.MaxInt64)

func generateMessageID() (string, error) {
	t := time.Now().UnixNano()
	pid := os.Getpid()
	rint, err := rand.Int(rand.Reader, maxBigInt)
	if err != nil {
		return "", err
	}
	msgid := fmt.Sprintf("<%d.%d.%d@%s>", t, pid, rint, AppDomain)
	return msgid, nil
}
*/
