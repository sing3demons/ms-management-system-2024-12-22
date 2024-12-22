package logger

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/sing3demons/profile-service/constants"
)

type SummaryLog interface {
	New(scenario string) SummaryLog
	AddField(fieldName string, fieldValue interface{})
	AddSuccessBlock(node, cmd, resultCode, resultDesc string)
	AddErrorBlock(node, cmd, resultCode, resultDesc string)
	IsEnd() bool
	End(resultCode, resultDescription string) error
}

func NewSummaryLog(req *http.Request, initInvoke string, cmd string, conf LogConfig) SummaryLog {

	session := req.Context().Value(constants.Session)

	currentTime := time.Now()
	if initInvoke == "" {
		initInvoke = fmt.Sprintf("%s_%s", conf.ProjectName, currentTime.Format("20060102150405"))
	}
	return &summaryLog{
		requestTime: &currentTime,
		session:     fmt.Sprintf("%v", session),
		initInvoke:  initInvoke,
		cmd:         cmd,
		conf:        conf,
	}
}

func (sl *summaryLog) New(scenario string) SummaryLog {
	sl.mu.Lock()
	defer sl.mu.Unlock()
	sl.cmd = scenario
	sl.blockDetail = []BlockDetail{}
	return sl
}

func (sl *summaryLog) AddField(fieldName string, fieldValue interface{}) {
	sl.mu.Lock()
	defer sl.mu.Unlock()
	if sl.optionalField == nil {
		sl.optionalField = OptionalFields{}
	}
	sl.optionalField[fieldName] = fieldValue
}

func (sl *summaryLog) AddSuccessBlock(node, cmd, resultCode, resultDesc string) {
	sl.addBlock(node, cmd, resultCode, resultDesc)
}

func (sl *summaryLog) AddErrorBlock(node, cmd, resultCode, resultDesc string) {
	sl.addBlock(node, cmd, resultCode, resultDesc)
}

func (sl *summaryLog) IsEnd() bool {
	sl.mu.Lock()
	defer sl.mu.Unlock()
	return sl.requestTime == nil
}

func (sl *summaryLog) End(resultCode, resultDescription string) error {
	sl.mu.Lock()
	defer sl.mu.Unlock()
	if sl.requestTime == nil {
		return errors.New("summaryLog is already ended")
	}
	sl.process(resultCode, resultDescription)
	sl.requestTime = nil
	return nil
}

func (sl *summaryLog) addBlock(node, cmd, resultCode, resultDesc string) {
	sl.mu.Lock()
	defer sl.mu.Unlock()

	for i := range sl.blockDetail {
		if sl.blockDetail[i].Node == node && sl.blockDetail[i].Cmd == cmd {
			sl.blockDetail[i].Result = append(sl.blockDetail[i].Result, SummaryResult{
				ResultCode: resultCode,
				ResultDesc: resultDesc,
			})
			sl.blockDetail[i].Count++
			return
		}
	}

	sl.blockDetail = append(sl.blockDetail, BlockDetail{
		Node: node,
		Cmd:  cmd,
		Result: []SummaryResult{{
			ResultCode: resultCode,
			ResultDesc: resultDesc,
		}},
		Count: 1,
	})
}

func (sl *summaryLog) process(responseResult, responseDesc string) {
	endTime := time.Now()
	elapsed := endTime.Sub(*sl.requestTime)

	seq := []map[string]interface{}{}
	for _, block := range sl.blockDetail {
		results := []map[string]string{}
		for _, res := range block.Result {
			results = append(results, map[string]string{
				"Result": res.ResultCode,
				"Desc":   res.ResultDesc,
			})
		}
		seq = append(seq, map[string]interface{}{
			"Node":   block.Node,
			"Cmd":    block.Cmd,
			"Result": results,
		})
	}

	logEntry := map[string]interface{}{
		"LogType":             "Summary",
		"InputTimeStamp":      sl.requestTime.Format(time.RFC3339),
		"Host":                getHostname(),
		"AppName":             sl.conf.ProjectName,
		"Instance":            os.Getenv("INSTANCE_ID"),
		"Session":             sl.session,
		"InitInvoke":          sl.initInvoke,
		"Scenario":            sl.cmd,
		"ResponseResult":      responseResult,
		"ResponseDesc":        responseDesc,
		"Sequences":           seq,
		"EndProcessTimeStamp": endTime.Format(time.RFC3339),
		"ProcessTime":         fmt.Sprintf("%d ms", elapsed.Milliseconds()),
	}

	if sl.optionalField != nil {
		logEntry["CustomDesc"] = sl.optionalField
	}

	b, _ := json.Marshal(logEntry)
	if sl.conf.Summary.LogConsole {
		os.Stdout.Write(b)
		os.Stdout.Write([]byte(endOfLine()))
	}

	if sl.conf.Summary.LogFile {
		sl.conf.Summary.LogSummary.Info(string(b))
	}

}

func getHostname() string {
	host, err := os.Hostname()
	if err != nil {
		return "unknown"
	}
	return host
}
