package main

import (
	"crypto/tls"
	"fmt"
	"io"
	"net/smtp"
)

type smtpClient interface {
	StartTLS(config *tls.Config) error
	Auth(a smtp.Auth) error
	Mail(from string) error
	Rcpt(to string) error
	Data() (io.WriteCloser, error)
	Close() error
}

type nopCloser struct {
	io.Writer
}

func (nopCloser) Close() error { return nil }

type fakeClient struct {
}

func (*fakeClient) StartTLS(config *tls.Config) error { return nil }
func (*fakeClient) Auth(a smtp.Auth) error            { return nil }
func (*fakeClient) Mail(from string) error            { return nil }
func (*fakeClient) Rcpt(to string) error              { return nil }
func (*fakeClient) Data() (io.WriteCloser, error)     { return nopCloser{io.Discard}, nil }
func (*fakeClient) Close() error                      { return nil }

func newSMTPClient() smtpClient {
	if !config.Mail.Enabled {
		return &fakeClient{}
	}
	addr := fmt.Sprintf("%s:%d", config.Mail.SMTPHost, config.Mail.SMTPPort)
	c, err := smtp.Dial(addr)
	sherpaCheck(err, "connecting to mail server")
	return c
}

func _sendmail(toName, toEmail, subject, textMsg string) {
	c := newSMTPClient()
	defer func() {
		if c != nil {
			c.Close()
		}
		c = nil
	}()

	if config.Mail.SMTPTLS {
		tlsconfig := &tls.Config{ServerName: config.Mail.SMTPHost}
		sherpaCheck(c.StartTLS(tlsconfig), "starting TLS with mail server")
	}

	if config.Mail.SMTPUsername != "" || config.Mail.SMTPPassword != "" {
		auth := smtp.PlainAuth("", config.Mail.SMTPUsername, config.Mail.SMTPPassword, config.Mail.SMTPHost)
		sherpaCheck(c.Auth(auth), "authenticating to mail server")
	}

	sherpaCheck(c.Mail(config.Mail.FromEmail), "setting from address")
	sherpaCheck(c.Rcpt(toEmail), "setting recipient address")

	data, err := c.Data()
	sherpaCheck(err, "preparing to write mail")
	if config.Mail.ReplyToEmail != "" {
		_, err = fmt.Fprintf(data, "Reply-To: %s <%s>\n", config.Mail.ReplyToName, config.Mail.ReplyToEmail)
		sherpaCheck(err, "writing reply-to header")
	}
	_, err = fmt.Fprintf(data, `From: %s <%s>
To: %s <%s>
Subject: %s

`, config.Mail.FromName, config.Mail.FromEmail, toName, toEmail, subject)
	sherpaCheck(err, "writing mail headers")

	_, err = fmt.Fprint(data, textMsg)
	sherpaCheck(err, "writing message")

	sherpaCheck(data.Close(), "closing mail body")
	sherpaCheck(c.Close(), "closing mail connection")
	c = nil
}
