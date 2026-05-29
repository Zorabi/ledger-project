package mailer

import (
	"bytes"
	"crypto/rand"
	"crypto/tls"
	"encoding/base64"
	"errors"
	"fmt"
	"mime"
	"mime/multipart"
	"net/smtp"
	"net/textproto"
	"strings"

	"ledger-backend/internal/config"
)

type Attachment struct {
	FileName    string
	ContentType string
	Content     []byte
}

func SendBackup(recipients []string, subject, body string, attachment Attachment) error {
	cfg := config.AppConfig.Email
	if cfg.Host == "" || cfg.Port == 0 || cfg.From == "" {
		return errors.New("邮件服务未配置")
	}
	recipients = cleanRecipients(recipients)
	if len(recipients) == 0 {
		return errors.New("备份收件人为空")
	}

	var msg bytes.Buffer
	writer := multipart.NewWriter(&msg)
	headers := []string{
		fmt.Sprintf("From: %s", cfg.From),
		fmt.Sprintf("To: %s", strings.Join(recipients, ",")),
		"Subject: " + encodeHeader(subject),
		"MIME-Version: 1.0",
		"Content-Type: " + mime.FormatMediaType("multipart/mixed", map[string]string{"boundary": writer.Boundary()}),
		"",
	}
	msg.WriteString(strings.Join(headers, "\r\n"))
	msg.WriteString("\r\n")
	textHeader := textproto.MIMEHeader{}
	textHeader.Set("Content-Type", "text/plain; charset=UTF-8")
	textHeader.Set("Content-Transfer-Encoding", "8bit")
	textPart, err := writer.CreatePart(textHeader)
	if err != nil {
		return err
	}
	if _, err := textPart.Write([]byte(body)); err != nil {
		return err
	}

	partHeader := textproto.MIMEHeader{}
	partHeader.Set("Content-Type", attachment.ContentType)
	partHeader.Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": attachment.FileName}))
	partHeader.Set("Content-Transfer-Encoding", "base64")
	part, err := writer.CreatePart(partHeader)
	if err != nil {
		return err
	}
	encoded := make([]byte, base64.StdEncoding.EncodedLen(len(attachment.Content)))
	base64.StdEncoding.Encode(encoded, attachment.Content)
	for len(encoded) > 76 {
		part.Write(encoded[:76])
		part.Write([]byte("\r\n"))
		encoded = encoded[76:]
	}
	part.Write(encoded)
	if err := writer.Close(); err != nil {
		return err
	}

	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	auth := smtp.Auth(nil)
	if cfg.Username != "" {
		auth = smtp.PlainAuth("", cfg.Username, cfg.Password, cfg.Host)
	}
	if cfg.Port == 465 {
		conn, err := tls.Dial("tcp", addr, &tls.Config{ServerName: cfg.Host, MinVersion: tls.VersionTLS12})
		if err != nil {
			return err
		}
		client, err := smtp.NewClient(conn, cfg.Host)
		if err != nil {
			return err
		}
		defer client.Close()
		if cfg.Username != "" {
			if err := client.Auth(auth); err != nil {
				return err
			}
		}
		if err := client.Mail(cfg.From); err != nil {
			return err
		}
		for _, recipient := range recipients {
			if err := client.Rcpt(recipient); err != nil {
				return err
			}
		}
		w, err := client.Data()
		if err != nil {
			return err
		}
		if _, err := w.Write(msg.Bytes()); err != nil {
			return err
		}
		if err := w.Close(); err != nil {
			return err
		}
		return client.Quit()
	}
	return smtp.SendMail(addr, auth, cfg.From, recipients, msg.Bytes())
}

func cleanRecipients(values []string) []string {
	out := make([]string, 0, len(values))
	seen := map[string]bool{}
	for _, value := range values {
		email := strings.TrimSpace(value)
		if email == "" || seen[email] {
			continue
		}
		seen[email] = true
		out = append(out, email)
	}
	return out
}

func encodeHeader(value string) string {
	return "=?UTF-8?B?" + base64.StdEncoding.EncodeToString([]byte(value)) + "?="
}

func MessageID() string {
	raw := make([]byte, 12)
	rand.Read(raw)
	return base64.RawURLEncoding.EncodeToString(raw)
}
