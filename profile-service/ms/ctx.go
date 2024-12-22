package ms

import (
	"net/http"
	"net/url"

	"github.com/sing3demons/profile-service/logger"
)

type IContext interface {
	Param(string) string
	ReadInput() InComing
	CommonLog(initInvoke, scenario, identity string) (logger.DetailLog, logger.SummaryLog)
	Response(responseCode int, responseData interface{}) error
	SendMail(message Message) error
	SendKafkaMessage(topic string, payload any) error
}

type ServiceHandleFunc func(c IContext) error

type HTTPContext struct {
	Res       http.ResponseWriter
	Req       *http.Request
	l         logger.DetailLog
	s         logger.SummaryLog
	intInvoke string
	scenario  string
	ms        *application
}

type InComing struct {
	Headers any        `json:"headers"`
	Query   url.Values `json:"query"`
	Body    any        `json:"body"`
}

type ConsumerContext struct {
	message kafkaMessage
	ms      *application
	l       logger.DetailLog
	s       logger.SummaryLog
	topic   string
	invoke  string
	payload Msg
}

type Header struct {
	Session string `json:"session"`
}

type Msg struct {
	Topic  string `json:"topic"`
	Header Header `json:"header"`
	Body   any    `json:"body"`
}

type Payload struct {
	Header Header `json:"header"`
	Body   any    `json:"body"`
}
