package mlog

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/sing3demons/profile-service/constants"
)

// MaskingConfig holds the configuration for masking sensitive fields.
type MaskingConfig struct {
	Highlight []string
	Mark      []string
}

// DetailLog represents the structured log entry.
type DetailLog struct {
	Name         string                 `json:"name"`
	Context      LogContext             `json:"context"`
	LogName      string                 `json:"log_name"`
	Session      string                 `json:"session,omitempty"`
	StartTime    string                 `json:"start_time"`
	EndTime      *string                `json:"end_time,omitempty"`
	Attributes   map[string]interface{} `json:"attributes"`
	Events       []LogEvent             `json:"events"`
	ResponseTime *string                `json:"response_time,omitempty"`
}

// LogContext holds trace and span information.
type LogContext struct {
	TraceID string `json:"trace_id"`
	SpanID  string `json:"span_id"`
}

// LogEvent represents an event in the log entry.
type LogEvent struct {
	Name       string                 `json:"name"`
	Timestamp  string                 `json:"timestamp"`
	Attributes map[string]interface{} `json:"attributes,omitempty"`
}

func NewDetailLog(req *http.Request) *DetailLog {
	traceID := req.Context().Value(constants.TraceIDKey).(string)
	if traceID == "" {
		traceID = uuid.New().String()
		req = req.WithContext(context.WithValue(req.Context(), constants.TraceIDKey, traceID))
	}

	spanID := req.Context().Value(constants.SpanIDKey).(string)
	if spanID == "" {
		spanID = uuid.New().String()
		req = req.WithContext(context.WithValue(req.Context(), constants.SpanIDKey, spanID))
	}

	startTime := time.Now().Format(time.RFC3339)

	return &DetailLog{
		Name:    os.Getenv("SERVICE_NAME"),
		LogName: "DETAIL",
		Context: LogContext{
			TraceID: traceID,
			SpanID:  spanID,
		},
		Session:   req.Context().Value(constants.Session).(string),
		StartTime: startTime,
		Attributes: map[string]interface{}{
			"http.route":  req.URL.Path,
			"http.method": req.Method,
			"http.device": req.UserAgent(),
		},
		Events: []LogEvent{},
	}
}

// AddEvent adds an event to the log entry.
func (d *DetailLog) AddEvent(name string, data interface{}, fieldsToMasks ...MaskingConfig) {
	fieldsToMask := MaskingConfig{
		Highlight: []string{},
		Mark:      []string{"password", "access_token", "refresh_token"},
	}

	if len(fieldsToMasks) > 0 {
		if fieldsToMasks[0].Highlight != nil {
			fieldsToMask.Highlight = append(fieldsToMask.Highlight, fieldsToMasks[0].Highlight...)
		}

		if fieldsToMasks[0].Mark != nil {
			fieldsToMask.Mark = append(fieldsToMask.Mark, fieldsToMasks[0].Mark...)
		}
	}

	convertedData, ok := data.(map[string]interface{})
	if !ok {
		jsonData, _ := json.Marshal(data)
		json.Unmarshal(jsonData, &convertedData)
	}

	attributes := d.MaskSensitiveData(convertedData, fieldsToMask)
	event := LogEvent{
		Name:       name,
		Timestamp:  time.Now().Format(time.RFC3339),
		Attributes: attributes,
	}

	d.Events = append(d.Events, event)
}

// MaskSensitiveData masks sensitive data based on the provided config.
func (d *DetailLog) MaskSensitiveData(data map[string]interface{}, config MaskingConfig) map[string]interface{} {
	if data == nil {
		return nil
	}

	maskedData := make(map[string]interface{})
	for key, value := range data {
		if contains(config.Mark, key) {
			maskedData[key] = "******" // Fully mask fields in 'mark'
		} else if contains(config.Highlight, key) {
			maskedData[key] = d.ApplyPartialMask(value)
		} else {
			maskedData[key] = value
		}
	}
	return maskedData
}

// ApplyPartialMask applies partial masking logic to sensitive strings.
func (d *DetailLog) ApplyPartialMask(value interface{}) interface{} {
	if str, ok := value.(string); ok {
		if isEmail(str) {
			parts := strings.Split(str, "@")
			if len(parts) == 2 {
				localPart := parts[0]
				domainPart := parts[1]

				if len(localPart) > 2 {
					mask := localPart[3:]
					notMasked := localPart[:3]
					localPart = notMasked + strings.Repeat("X", len(mask))
				} else {
					localPart = string(localPart[0]) + strings.Repeat("X", len(localPart)-1)
				}
				return fmt.Sprintf("%s@%s", localPart, domainPart)
			}
		}
		return maskGeneralString(str)
	}
	return value
}

// isEmail checks if a string is a valid email format.
func isEmail(value string) bool {
	re := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	return re.MatchString(value)
}

// maskGeneralString partially masks non-email strings.
func maskGeneralString(value string) string {
	if len(value) > 2 {
		return value[:2] + strings.Repeat("X", len(value)-2)
	}
	return value
}

// SetResponseTime sets the response time for the log entry.
func (d *DetailLog) SetResponseTime(duration string) {
	d.ResponseTime = &duration
}

// ResponseWriterWrapper is a custom response writer to capture status and body.
type ResponseWriterWrapper struct {
	http.ResponseWriter
	status int
	body   []byte
}

// WriteHeader captures the response status.
func (rw *ResponseWriterWrapper) WriteHeader(status int) {
	rw.status = status
	rw.ResponseWriter.WriteHeader(status)
}

// Write captures the response body.
func (rw *ResponseWriterWrapper) Write(b []byte) (int, error) {
	rw.body = b
	return rw.ResponseWriter.Write(b)
}

// Utility function to check if a string exists in a slice.
func contains(slice []string, item string) bool {
	for _, a := range slice {
		if a == item {
			return true
		}
	}
	return false
}
