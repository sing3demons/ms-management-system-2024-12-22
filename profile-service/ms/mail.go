package ms

import (
	"strings"

	"github.com/sing3demons/profile-service/constants"
	"github.com/sing3demons/profile-service/logger"
	gomail "gopkg.in/mail.v2"
)

// Attachments   []string

type MailServer struct {
	Host   string `json:"host,omitempty"`
	Port   int    `json:"port,omitempty"`
	Secure bool   `json:"secure,omitempty"`
	Auth   *Auth  `json:"auth,omitempty"`
}

type Auth struct {
	User string `json:"username,omitempty"`
	Pass string `json:"password,omitempty"`
}

type Message struct {
	From        string   `json:"from,omitempty"`
	To          string   `json:"to,omitempty"`
	Subject     string   `json:"subject,omitempty"`
	Body        string   `json:"body,omitempty"`
	Attachment  string   `json:"attachment,omitempty"`
	Attachments []string `json:"attachments,omitempty"`
}

type Result struct {
	Err        bool
	ResultDesc string
	ResultData interface{}
}

func sendMail(mailServer MailServer, params Message, detailLog logger.DetailLog, summaryLog logger.SummaryLog) Result {
	cmdName := "send_mail"
	result := Result{}
	invoke := GenerateXTid(cmdName)
	if strings.Contains(params.From, "{email_from}") {
		if mailServer.Auth != nil && mailServer.Auth.User != "" {
			params.From = strings.ReplaceAll(params.From, "{email_from}", mailServer.Auth.User)
		}
	}

	// Setup SMTP configuration
	dialer := gomail.NewDialer(mailServer.Host, mailServer.Port, "", "")
	if mailServer.Auth != nil && mailServer.Auth.User != "" && mailServer.Auth.Pass != "" {
		dialer.Username = mailServer.Auth.User
		dialer.Password = mailServer.Auth.Pass
	}
	dialer.SSL = mailServer.Secure

	detailLog.AddOutputRequest(constants.MAIL_SERVER, cmdName, invoke, params, params)
	detailLog.End()

	// Create the email message
	message := gomail.NewMessage()
	message.SetHeader("From", params.From)
	message.SetHeader("To", params.To)
	message.SetHeader("Subject", params.Subject)
	message.SetBody("text/html", params.Body)
	message.SetHeader("Return-Path", params.From)

	if params.Attachment != "" {
		message.Attach(params.Attachment)
	}

	// Sending the email
	// log.Printf("smtpBody: Host=%s, Port=%d, Secure=%t, User=%s", mailServer.Host, mailServer.Port, mailServer.Secure, dialer.Username)

	err := dialer.DialAndSend(message)
	if err != nil {
		if strings.Contains(err.Error(), "timeout") {
			result.ResultDesc = "timeout"
		} else {
			result.ResultDesc = "connection_error"
		}
		result.Err = true
		result.ResultData = err.Error()

		detailLog.AddInputRequest(constants.MAIL_SERVER, cmdName, invoke, result, result)
		summaryLog.AddErrorBlock(constants.MAIL_SERVER, cmdName, "500", result.ResultDesc)
		return result
	}

	result.ResultData = "Email sent successfully"
	detailLog.AddInputRequest(constants.MAIL_SERVER, cmdName, invoke, result, result)
	summaryLog.AddSuccessBlock(constants.MAIL_SERVER, cmdName, "200", "success")
	return result
}
