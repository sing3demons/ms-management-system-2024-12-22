package logger

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"runtime"
	"time"

	"github.com/sing3demons/profile-service/constants"
)

type DetailLog interface {
	IsRawDataEnabled() bool
	AddInputRequest(node, cmd, invoke string, rawData, data interface{})
	AddOutputRequest(node, cmd, invoke string, rawData, data interface{})
	End()
	AddInputResponse(node, cmd, invoke string, rawData, data interface{}, protocol, protocolMethod string)
	AddOutputResponse(node, cmd, invoke string, rawData, data interface{})
	AutoEnd() bool
}

func NewDetailLog(req *http.Request, initInvoke, scenario, identity string, conf LogConfig) DetailLog {
	session := req.Context().Value(constants.Session)

	host, _ := os.Hostname()
	data := &detailLog{
		LogType:       "Detail",
		Host:          host,
		AppName:       conf.ProjectName,
		Instance:      getInstance(),
		Session:       fmt.Sprintf("%v", session),
		InitInvoke:    initInvoke,
		Scenario:      scenario,
		Identity:      identity,
		Input:         []InputOutputLog{},
		Output:        []InputOutputLog{},
		conf:          conf.Detail,
		startTimeDate: time.Now(),
		timeCounter:   make(map[string]time.Time),
		req:           req,
	}

	return data
}

func getInstance() *string {
	instance, err := os.Hostname()
	if err != nil {
		instance = fmt.Sprintf("%d", os.Getpid())
	}
	if instance == "" {
		instance = fmt.Sprintf("%d", os.Getpid())
	}
	return &instance
}

func (dl *detailLog) IsRawDataEnabled() bool {
	return dl.conf.RawData
}

func (dl *detailLog) AddInputRequest(node, cmd, invoke string, rawData, data interface{}) {
	if rawData != nil {
		if _, ok := rawData.(string); !ok {
			rawData = ToJson(rawData)
		}
	}
	dl.addInput(&logEvent{
		node:           node,
		cmd:            cmd,
		invoke:         invoke,
		logType:        "req",
		rawData:        rawData,
		data:           data,
		protocol:       dl.req.Proto,
		protocolMethod: dl.req.Method,
	})
}

func (dl *detailLog) AddInputResponse(node, cmd, invoke string, rawData, data interface{}, protocol, protocolMethod string) {
	resTime := time.Now().Format(time.RFC3339)
	if rawData != nil {
		if _, ok := rawData.(string); !ok {
			rawData = ToJson(rawData)
		}
	}
	dl.addInput(&logEvent{
		node:           node,
		cmd:            cmd,
		invoke:         invoke,
		logType:        "res",
		rawData:        rawData,
		data:           ToStruct(data),
		resTime:        resTime,
		protocol:       protocol,
		protocolMethod: protocolMethod,
	})
}

func (dl *detailLog) AddOutputResponse(node, cmd, invoke string, rawData, data interface{}) {
	if rawData != nil {
		if _, ok := rawData.(string); !ok {
			// rawData = fmt.Sprintf("%v", rawData)
			rawData = ToJson(rawData)
		}
	}
	dl.AddOutput(logEvent{
		node:    node,
		cmd:     cmd,
		invoke:  invoke,
		logType: "res",
		rawData: rawData,
		data:    ToStruct(data),
	})
	// dl.End()
}

func (dl *detailLog) addInput(input *logEvent) {
	dl.mu.Lock()
	defer dl.mu.Unlock()
	now := time.Now()
	if dl.startTimeDate.IsZero() {
		dl.startTimeDate = now
	}

	var resTimeString string
	if input.resTime != "" {
		resTimeString = input.resTime
	} else if input.logType == "res" {
		if startTime, exists := dl.timeCounter[input.invoke]; exists {
			duration := time.Since(startTime).Milliseconds()
			resTimeString = fmt.Sprintf("%d ms", duration)
			delete(dl.timeCounter, input.invoke)
		}
	}

	protocolValue := dl.buildValueProtocol(&input.protocol, &input.protocolMethod)
	inputLog := InputOutputLog{
		Invoke:   input.invoke,
		Event:    fmt.Sprintf("%s.%s", input.node, input.cmd),
		Protocol: protocolValue,
		Type:     input.logType,
		RawData:  dl.isRawDataEnabledIf(input.rawData),
		Data:     input.data,
		ResTime:  &resTimeString,
	}
	dl.Input = append(dl.Input, inputLog)
}

func (dl *detailLog) AddOutputRequest(node, cmd, invoke string, rawData, data interface{}) {
	if rawData != nil {
		if _, ok := rawData.(string); !ok {
			rawData = ToJson(rawData)
		}
	}
	dl.AddOutput(logEvent{
		node:           node,
		cmd:            cmd,
		invoke:         invoke,
		logType:        "rep",
		rawData:        rawData,
		data:           ToStruct(data),
		protocol:       dl.req.Proto,
		protocolMethod: dl.req.Method,
	})
}

func (dl *detailLog) AddOutput(out logEvent) {
	dl.mu.Lock()
	defer dl.mu.Unlock()
	now := time.Now()
	if out.invoke != "" && out.logType != "res" {
		dl.timeCounter[out.invoke] = now
	}

	protocolValue := dl.buildValueProtocol(&out.protocol, &out.protocolMethod)
	outputLog := InputOutputLog{
		Invoke:   out.invoke,
		Event:    fmt.Sprintf("%s.%s", out.node, out.cmd),
		Protocol: protocolValue,
		Type:     out.logType,
		RawData:  dl.isRawDataEnabledIf(out.rawData),
		Data:     out.data,
	}
	dl.Output = append(dl.Output, outputLog)
}

func (dl *detailLog) End() {
	if dl.startTimeDate.IsZero() {
		log.Fatal("end() called without any input/output")
	}

	processingTime := fmt.Sprintf("%d ms", time.Since(dl.startTimeDate).Milliseconds())
	dl.ProcessingTime = &processingTime

	inputTimeStamp := dl.formatTime(dl.inputTime)
	dl.InputTimeStamp = inputTimeStamp

	outputTimeStamp := dl.formatTime(dl.outputTime)
	dl.OutputTimeStamp = outputTimeStamp

	logDetail, _ := json.Marshal(dl)
	if dl.conf.LogConsole {
		os.Stdout.Write(logDetail)
		os.Stdout.Write([]byte(endOfLine()))
	}

	if dl.conf.LogFile {
		dl.conf.LogDetail.Info(string(logDetail))
	}

	dl.clear()
}

func (dl *detailLog) buildValueProtocol(protocol, method *string) *string {
	if protocol == nil {
		return nil
	}
	result := *protocol
	if method != nil {
		result += "." + *method
	}
	return &result
}

func (dl *detailLog) AutoEnd() bool {
	if dl.startTimeDate.IsZero() {
		return false
	}
	if len(dl.Input) == 0 && len(dl.Output) == 0 {
		return false
	}

	dl.End()
	return true
}

func (dl *detailLog) isRawDataEnabledIf(rawData interface{}) interface{} {
	if dl.conf.RawData {
		return rawData
	}
	return nil
}

func (dl *detailLog) formatTime(t *time.Time) *string {
	if t == nil {
		return nil
	}
	ts := t.Format(time.RFC3339)
	return &ts
}

func endOfLine() string {
	if runtime.GOOS == "windows" {
		return "\r\n"
	}
	return "\n"
}

func (dl *detailLog) clear() {
	dl.ProcessingTime = nil
	dl.InputTimeStamp = nil
	dl.OutputTimeStamp = nil
	dl.Input = nil
	dl.Output = nil
	dl.startTimeDate = time.Time{}
}

func ToJson(data interface{}) string {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Sprintf("%v", data)
	}
	return string(jsonData)
}

// convert struct to json
func ToStruct(data interface{}) (result interface{}) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return data
	}

	err = json.Unmarshal(jsonData, &result)
	if err != nil {
		return data
	}
	return result
}
