package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/google/uuid"
	"github.com/sing3demons/profile-service/constants"
)

type HandlerResponse struct {
	ResultCode       string      `json:"resultCode"`
	ResultDesc       string      `json:"resultDesc"`
	DeveloperMessage string      `json:"developerMessage,omitempty"`
	Data             interface{} `json:"data,omitempty"`
}

type customResponseWriter struct {
	http.ResponseWriter
	statusCode int
	body       *bytes.Buffer
}

// newCustomResponseWriter initializes a new instance of CustomResponseWriter
func newCustomResponseWriter(w http.ResponseWriter) *customResponseWriter {
	return &customResponseWriter{
		ResponseWriter: w,
		statusCode:     http.StatusOK,
		body:           &bytes.Buffer{},
	}
}

// WriteHeader captures the status code
func (crw *customResponseWriter) WriteHeader(code int) {
	crw.statusCode = code
	crw.ResponseWriter.WriteHeader(code)
}

// Write captures the response body
func (crw *customResponseWriter) Write(data []byte) (int, error) {
	crw.body.Write(data) // Capture the response body
	return crw.ResponseWriter.Write(data)
}

func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var ctx context.Context = r.Context()
		var invokeId string

		if r.Header.Get(string(constants.Session)) == "" {
			uuidV7, err := uuid.NewV7()
			if err != nil {
				invokeId = uuid.New().String()
			} else {
				invokeId = uuidV7.String()
			}
		}
		r = r.WithContext(context.WithValue(r.Context(), constants.Session, invokeId))

		traceID := uuid.New().String()
		spanID := uuid.New().String()

		// Add trace_id and span_id to the request context
		ctx = context.WithValue(r.Context(), constants.TraceIDKey, traceID)
		ctx = context.WithValue(ctx, constants.SpanIDKey, spanID)

		// Store request body
		bodyBytes, _ := io.ReadAll(r.Body)

		r.Body.Close() //  must close
		r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
		// ctx = context.WithValue(ctx, constant.BodyBytes, bodyBytes)

		// concurrent_gauge
		// prometheus.ConcurrentGauge.Inc()
		// defer prometheus.ConcurrentGauge.Dec()

		ctx = SetInvoke(ctx, invokeId)

		// Create a new request with the updated context
		// Call the next handler
		crw := newCustomResponseWriter(w)

		// Call the next handler in the chain
		next.ServeHTTP(crw, r.WithContext(ctx))

		var res HandlerResponse
		resBytes := crw.body.Bytes()
		json.Unmarshal(resBytes, &res)

		// Optionally, write the log summary in the response
		// w.Write([]byte("\nLog Summary:\n" + logSummary))
	})
}

type InvokeKey struct{}

func SetInvoke(ctx context.Context, invoke string) context.Context {
	return context.WithValue(ctx, InvokeKey{}, invoke)
}

func Minify(jsonB []byte) ([]byte, error) {
	var buff *bytes.Buffer = new(bytes.Buffer)
	errCompact := json.Compact(buff, jsonB)
	if errCompact != nil {
		newErr := fmt.Errorf("failure encountered compacting json := %v", errCompact)
		return []byte{}, newErr
	}

	b, err := io.ReadAll(buff)
	if err != nil {
		readErr := fmt.Errorf("read buffer error encountered := %v", err)
		return []byte{}, readErr
	}

	return b, nil
}
