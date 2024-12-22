package logger

import (
	"net/http"
	"sync"
	"time"

	"go.uber.org/zap"
)

type LogConfig struct {
	ProjectName string
	Namespace   string
	Summary     SummaryLogConfig `json:"summary"`
	Detail      DetailLogConfig  `json:"detail"`
}

type SummaryLogConfig struct {
	Name       string `json:"name"`
	RawData    bool   `json:"rawData"`
	LogFile    bool   `json:"logFile"`
	LogConsole bool   `json:"logConsole"`
	LogSummary *zap.Logger
}

type DetailLogConfig struct {
	Name       string `json:"name"`
	RawData    bool   `json:"rawData"`
	LogFile    bool   `json:"logFile"`
	LogConsole bool   `json:"logConsole"`
	LogDetail  *zap.Logger
}

type InputOutputLog struct {
	Invoke   string      `json:"Invoke"`
	Event    string      `json:"Event"`
	Protocol *string     `json:"Protocol,omitempty"`
	Type     string      `json:"Type"`
	RawData  interface{} `json:"RawData,omitempty"`
	Data     interface{} `json:"Data"`
	ResTime  *string     `json:"ResTime,omitempty"`
}

type detailLog struct {
	LogType         string               `json:"LogType"`
	Host            string               `json:"Host"`
	AppName         string               `json:"AppName"`
	Instance        *string              `json:"Instance,omitempty"`
	Session         string               `json:"Session"`
	InitInvoke      string               `json:"InitInvoke"`
	Scenario        string               `json:"Scenario"`
	Identity        string               `json:"Identity"`
	InputTimeStamp  *string              `json:"InputTimeStamp,omitempty"`
	Input           []InputOutputLog     `json:"Input"`
	OutputTimeStamp *string              `json:"OutputTimeStamp,omitempty"`
	Output          []InputOutputLog     `json:"Output"`
	ProcessingTime  *string              `json:"ProcessingTime,omitempty"`
	conf            DetailLogConfig      `json:"-"`
	startTimeDate   time.Time            `json:"-"`
	inputTime       *time.Time           `json:"-"`
	outputTime      *time.Time           `json:"-"`
	timeCounter     map[string]time.Time `json:"-"`
	req             *http.Request
	mu              sync.Mutex
}

type logEvent struct {
	node           string
	cmd            string
	invoke         string
	logType        string
	rawData        interface{}
	data           interface{}
	resTime        string
	protocol       string
	protocolMethod string
}

type summaryLog struct {
	mu            sync.Mutex
	requestTime   *time.Time
	session       string
	initInvoke    string
	cmd           string
	blockDetail   []BlockDetail
	optionalField OptionalFields
	conf          LogConfig
}

type SummaryResult struct {
	ResultCode string `json:"ResultCode"`
	ResultDesc string `json:"ResultDesc"`
	Count      int    `json:"-"`
}

type BlockDetail struct {
	Node   string          `json:"Node"`
	Cmd    string          `json:"Cmd"`
	Result []SummaryResult `json:"Result"`
	Count  int             `json:"Count"`
}

type OptionalFields map[string]interface{}
