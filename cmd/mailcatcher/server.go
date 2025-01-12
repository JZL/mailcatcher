package main

import (
	"io"
	"log"
	"net/smtp"
	"strconv"
	"strings"

	"regexp"
	"fmt"
	gosmtp "github.com/emersion/go-smtp"
	"github.com/veqryn/go-email/email"
)

var config *Configuration

type Backend struct{}

func (bkd *Backend) Login(_ *gosmtp.Conn, _, _ string) (gosmtp.Session, error) {
	return &Session{
		to: make([]string, 0),
	}, nil
}

func (bkd *Backend) AnonymousLogin(_ *gosmtp.Conn) (gosmtp.Session, error) {
	return &Session{
		to: make([]string, 0),
	}, nil
}

func (bkd *Backend) NewSession(c *gosmtp.Conn) (gosmtp.Session, error) {
	return &Session{}, nil
}

type Session struct {
	from string
	to   []string
}

func (s *Session) Mail(from string, opts *gosmtp.MailOptions) error {
	s.from = from
	return nil
}

func (s *Session) AuthPlain(username, password string) error {
	return nil
}


func (s *Session) Rcpt(to string, opts *gosmtp.RcptOptions) error {
	s.to = append(s.to, to)
	return nil
}

func (s *Session) Data(r io.Reader) error {
	log.Printf("New message from '%s' to '%s' received", s.from, s.to)
	if isRecipientValid(s.to) {
		if msg, err := email.ParseMessage(r); err != nil {
			log.Fatal("error", err)
			return err
		} else {
			justPrefixREA := regexp.MustCompile("> *$")
			justPrefixREB := regexp.MustCompile("^.*<")
			justPrefixREC := regexp.MustCompile("@.*")
			toField := msg.Header.To()[0]
			toField = justPrefixREA.ReplaceAllString(toField, "")
			toField = justPrefixREB.ReplaceAllString(toField, "")
			toField = justPrefixREC.ReplaceAllString(toField, "")

			msg.Header.SetSubject(fmt.Sprintf("🧤 %s - %s", toField, msg.Header.Subject()))

			// msg.Header.SetSubject(fmt.Sprintf("🧤 %s - %s",
			// 	justPrefixREC.ReplaceAllString(
			// 		justPrefixREB.ReplaceAllString(
			// 			justPrefixREA.ReplaceAllString(
			// 				msg.Header.To()[0],
			// 				""),""),""),
			// 	msg.Header.Subject()))

			noBrackets := regexp.MustCompile("[<>]")

			msg.Header.SetTo(fmt.Sprintf("\"%s\" <%s>",
				noBrackets.ReplaceAllString(msg.Header.To()[0], "%"),
				config.MC_REDIRECT_TO))
			msg.Header.SetFrom(fmt.Sprintf("\"%s\" <%s>", "mailcatch", config.MC_SENDER_MAIL))

			/*
			msg.Header.SetTo("mainEmail@gmail")
			msg.Header.SetFrom("smtpEmail@gmail")
			*/

			sendMail(msg)

			if err != nil {
				log.Printf("smtp error: %s", err)
			}
		}
	} else {
		log.Print("ignoring message")
	}
	return nil
}

func (s *Session) Reset() {}

func (s *Session) Logout() error {
	return nil
}

func isRecipientValid(recipients []string) bool {
	for _, recipient := range recipients {
		if strings.HasSuffix(recipient, config.MC_HOST) {
			return true
		}
	}
	return false
}

func sendMail(msg *email.Message) {
	if err := msg.Save(); err != nil {
		log.Printf("can't save message: %s", err)
		return
	}
	b, err := msg.Bytes()
	if err != nil {
		log.Printf("can't convert message: %s", err)
		return
	}

	err = smtp.SendMail(fmt.Sprintf("%s:%d", config.MC_SMTP_HOST, config.MC_SMTP_PORT),
		smtp.PlainAuth("", config.MC_SMTP_USER, config.MC_SMTP_PASSWORD, config.MC_SMTP_HOST),
		config.MC_SENDER_MAIL, []string{config.MC_REDIRECT_TO}, b)

	if err != nil {
		log.Printf("smtp error: %s", err)
		return
	}
}

func NewServer(configuration *Configuration) error {
	config = configuration
	be := &Backend{}

	s := gosmtp.NewServer(be)

	s.Addr = ":" + strconv.Itoa(config.MC_PORT)
	s.Domain = config.MC_HOST
	s.MaxMessageBytes = 1024 * 1024 * 20
	s.MaxRecipients = 50
	s.AllowInsecureAuth = true
	s.AuthDisabled = true

	log.Println("Starting server at", s.Addr)
	return s.ListenAndServe()
}
