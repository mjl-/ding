package main

import (
	"crypto/tls"
	"fmt"
	"io"
	"net/smtp"
	"strings"
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
	recordRcpt bool
	recipients []string
}

func (*fakeClient) StartTLS(config *tls.Config) error { return nil }
func (*fakeClient) Auth(a smtp.Auth) error            { return nil }
func (*fakeClient) Mail(from string) error            { return nil }
func (c *fakeClient) Rcpt(to string) error {
	if c.recordRcpt {
		c.recipients = append(c.recipients, to)
	}
	return nil
}
func (*fakeClient) Data() (io.WriteCloser, error) { return nopCloser{io.Discard}, nil }
func (*fakeClient) Close() error                  { return nil }

var newSMTPClient func() smtpClient

func dialSMTPClient() smtpClient {
	addr := fmt.Sprintf("%s:%d", config.Mail.SMTPHost, config.Mail.SMTPPort)
	c, err := smtp.Dial(addr)
	_checkf(err, "connecting to mail server")
	return c
}

func _sendmail(to []string, subject, textMsg string) {
	c := newSMTPClient()
	defer func() {
		if c != nil {
			c.Close()
		}
		c = nil
	}()

	if config.Mail.SMTPTLS {
		tlsconfig := &tls.Config{ServerName: config.Mail.SMTPHost}
		_checkf(c.StartTLS(tlsconfig), "starting TLS with mail server")
	}

	if config.Mail.SMTPUsername != "" || config.Mail.SMTPPassword != "" {
		auth := smtp.PlainAuth("", config.Mail.SMTPUsername, config.Mail.SMTPPassword, config.Mail.SMTPHost)
		_checkf(c.Auth(auth), "authenticating to mail server")
	}

	_checkf(c.Mail(config.Mail.FromEmail), "setting from address")
	for _, rcpt := range to {
		_checkf(c.Rcpt(rcpt), "setting recipient address")
	}

	data, err := c.Data()
	_checkf(err, "preparing to write mail")

	var msg string
	if config.Mail.ReplyToEmail != "" {
		msg = fmt.Sprintf("Reply-To: %s <%s>\n", config.Mail.ReplyToName, config.Mail.ReplyToEmail)
	}
	// todo: should wrap long line, encode name/address.
	var tohdr string
	for i, rcpt := range to {
		if i > 0 {
			tohdr += ", "
		}
		tohdr += fmt.Sprintf("<%s>", rcpt)
	}
	msg += fmt.Sprintf(`From: %s <%s>
To: %s
Subject: %s

%s
`, config.Mail.FromName, config.Mail.FromEmail, tohdr, subject, textMsg)
	msg = strings.ReplaceAll(msg, "\n", "\r\n")

	_, err = fmt.Fprint(data, msg)
	_checkf(err, "writing message")

	_checkf(data.Close(), "closing mail body")
	_checkf(c.Close(), "closing mail connection")
	c = nil
}
