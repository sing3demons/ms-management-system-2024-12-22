package ms

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/sing3demons/profile-service/constants"
	"github.com/sing3demons/profile-service/logger"
)

func NewHTTPContext(res http.ResponseWriter, req *http.Request, ms *application) IContext {
	return &HTTPContext{
		Res: res,
		Req: req,
		ms:  ms,
	}
}

// func (h *HTTPContext) L() *mlog.DetailLog {
// 	return mlog.NewDetailLog(h.Req)
// }

func (h *HTTPContext) ReadInput() InComing {
	data := InComing{
		Headers: h.Req.Header,
		Query:   h.Req.URL.Query(),
		Body:    map[string]any{},
	}
	bodyBytes, err := io.ReadAll(h.Req.Body)
	if err != nil {
		return data

	}

	h.Req.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
	json.Unmarshal(bodyBytes, &data.Body)

	return data
}

func (h *HTTPContext) Param(key string) string {
	return mux.Vars(h.Req)[key]
}

func (h *HTTPContext) Response(responseCode int, responseData interface{}) error {
	h.JSON(responseCode, responseData)
	return nil
}

func (h *HTTPContext) CommonLog(initInvoke, scenario, identity string) (logger.DetailLog, logger.SummaryLog) {

	conf := logger.LogConfig{}
	conf.ProjectName = h.ms.config.LogConfig.ProjectName
	conf.Namespace = h.ms.config.LogConfig.Namespace

	conf.Summary.RawData = h.ms.config.LogConfig.Summary.RawData
	conf.Summary.LogFile = h.ms.config.LogConfig.Summary.LogFile
	conf.Summary.LogConsole = h.ms.config.LogConfig.Summary.LogConsole
	conf.Summary.LogSummary = h.ms.config.LogConfig.Summary.LogSummary

	conf.Detail.RawData = h.ms.config.LogConfig.Detail.RawData
	conf.Detail.LogFile = h.ms.config.LogConfig.Detail.LogFile
	conf.Detail.LogConsole = h.ms.config.LogConfig.Detail.LogConsole
	conf.Detail.LogDetail = h.ms.config.LogConfig.Detail.LogDetail

	detailLog := logger.NewDetailLog(h.Req, initInvoke, scenario, identity, conf)
	h.s = logger.NewSummaryLog(h.Req, initInvoke, scenario, conf)
	h.intInvoke = initInvoke
	h.scenario = scenario
	detailLog.AddInputRequest(constants.CLIENT, scenario, initInvoke, nil, h.ReadInput())
	h.l = detailLog
	return h.l, h.s
}

func (h *HTTPContext) JSON(code int, data interface{}) {

	if h.l != nil {
		h.l.AddOutputRequest(constants.CLIENT, h.scenario, h.intInvoke, data, data)
		h.l.AutoEnd()
		h.l = nil
	}

	if h.s != nil {
		s := *&h.s
		if !s.IsEnd() {
			resultCode := fmt.Sprintf("%d", code)
			resultDesc := http.StatusText(code)
			s.End(resultCode, resultDesc)
			s = nil
		}
	}
	h.intInvoke = ""
	h.scenario = ""

	h.Res.Header().Set(constants.ContentType, constants.ContentTypeJSON)
	h.Res.WriteHeader(code)
	json.NewEncoder(h.Res).Encode(data)
	h = nil

	// if h.Log != nil {
	// 	h.Log.End()
	// }
}

func (b *HTTPContext) Error(code int, err error) {
	data := map[string]interface{}{
		"message": err.Error(),
	}

	b.JSON(code, data)
}

func (h *HTTPContext) GetSession() string {
	return h.Req.Context().Value(constants.Session).(string)
}

func (h *HTTPContext) SendMail(message Message) error {
	result := sendMail(h.ms.config.MailServer, message, h.l, h.s)
	if result.Err {
		return fmt.Errorf("Error sending email: %s", result.ResultDesc)
	}
	return nil
}

func (h *HTTPContext) SendKafkaMessage(topic string, payload any) error {

	return nil
}
