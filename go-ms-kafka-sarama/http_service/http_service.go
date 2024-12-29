package http_service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"runtime"

	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/sing3demons/logger-kp/logger"
)

type TMap map[string]string

type HTTPMethod string

const (
	GET    HTTPMethod = "GET"
	POST   HTTPMethod = "POST"
	PUT    HTTPMethod = "PUT"
	DELETE HTTPMethod = "DELETE"
)

type RequestAttributes struct {
	Headers        TMap
	Method         HTTPMethod
	Params         TMap
	Query          TMap
	Body           TMap
	RetryCondition string
	RetryCount     int
	Timeout        int
	Service        string
	Command        string
	Invoke         string
	URL            string
	Auth           *BasicAuth
	StatusSuccess  []int
}

type OptionAttributes interface{}

type BasicAuth struct {
	Username string
	Password string
}

// attr.Service, attr.Command, attr.Invoke
type attrDetailLog struct {
	Service string
	Command string
	Invoke  string
	Method  HTTPMethod
}

type ApiResponse struct {
	Err        error
	Header     http.Header
	Body       interface{}
	Status     int
	StatusText string
	attr       attrDetailLog
}

type httpService struct {
	requestAttributes []RequestAttributes
	detailLog         logger.DetailLog
	summaryLog        logger.SummaryLog
}

func RequestHttp(optionAttributes OptionAttributes, detailLog logger.DetailLog, summaryLog logger.SummaryLog) (any, error) {
	var requestAttributes []RequestAttributes
	switch attr := optionAttributes.(type) {
	case []RequestAttributes:
		requestAttributes = attr
	case RequestAttributes:
		requestAttributes = []RequestAttributes{attr}
	default:
		return nil, errors.New("invalid optionAttributes type")
	}

	service := httpService{
		requestAttributes: requestAttributes,
		detailLog:         detailLog,
		summaryLog:        summaryLog,
	}

	return service.requestHttp()
}

func (svc *httpService) requestHttp() (any, error) {
	var wg sync.WaitGroup

	// Use a channel to collect responses
	responseChan := make(chan ApiResponse, len(svc.requestAttributes))
	semaphore := make(chan struct{}, 100) // limit to 100 goroutines

	for _, attr := range svc.requestAttributes {
		semaphore <- struct{}{}
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		config := RetryConfig{
			MaxAttempts:  3,
			InitialDelay: 100 * time.Millisecond,
			MaxDelay:     1 * time.Second,
		}

		wg.Add(1)
		go func(attr RequestAttributes) {
			defer wg.Done()
			defer func() { <-semaphore }()

			transport := NewRetryRoundTripper(
				http.DefaultTransport,
				config,
			)

			req, err := createRequest(ctx, attr, svc.detailLog, svc.summaryLog)
			if err != nil {
				responseChan <- ApiResponse{
					Status:     500,
					attr:       attrDetailLog{Service: attr.Service, Command: attr.Command, Invoke: attr.Invoke, Method: attr.Method},
					Header:     nil,
					Body:       nil,
					StatusText: err.Error(),
					Err:        err,
				}
				return
			}

			client := &http.Client{
				Timeout:   time.Duration(attr.Timeout) * time.Second,
				Transport: transport,
			}
			response := executeRequest(client, req, attr)

			// mu.Lock()
			// defer mu.Unlock()
			responseChan <- *response
		}(attr)
	}

	fmt.Println("Number of goroutines:", runtime.NumGoroutine())
	wg.Wait()
	close(responseChan)
	svc.detailLog.AutoEnd()

	var responses []any
	for response := range responseChan {
		service := response.attr.Service
		command := response.attr.Command
		invoke := response.attr.Invoke
		method := response.attr.Method

		resultCode := fmt.Sprintf("%d", response.Status)
		svc.summaryLog.AddSuccess(service, command, resultCode, response.StatusText)
		// remove attr from response
		// delete(response.Body, "attr")
		svc.detailLog.AddInputResponse(service, command, invoke, nil, response, "http", string(method))
		responses = append(responses, response.Body)
	}

	if len(responses) == 1 {
		return responses[0], nil
	}
	return responses, nil
}

type ProcessLog struct {
	Header      TMap       `json:"Header"`
	Url         string     `json:"Url"`
	QueryString TMap       `json:"QueryString"`
	Body        TMap       `json:"Body"`
	Method      HTTPMethod `json:"Method"`
	RetryCount  int        `json:"RetryCount,omitempty"`
	Timeout     int        `json:"Timeout,omitempty"`
	Auth        *BasicAuth `json:"Auth,omitempty"`
}

func createRequest(
	ctx context.Context,
	attr RequestAttributes,
	detailLog logger.DetailLog,
	summaryLog logger.SummaryLog,
) (*http.Request, error) {
	processLog := ProcessLog{
		Header:      attr.Headers,
		Url:         attr.URL,
		QueryString: attr.Query,
		Body:        attr.Body,
		Method:      attr.Method,
		RetryCount:  attr.RetryCount,
		Timeout:     attr.Timeout,
		Auth:        attr.Auth,
	}

	if len(attr.Params) > 0 {
		// Replace URL params
		for key, value := range attr.Params {
			// startWith "{}"
			if strings.Contains(attr.URL, "{"+key+"}") {
				attr.URL = strings.ReplaceAll(attr.URL, "{"+key+"}", value)
			} else if strings.Contains(attr.URL, ":"+key) {
				attr.URL = strings.ReplaceAll(attr.URL, ":"+key, value)
			} else {
				attr.URL = strings.ReplaceAll(attr.URL, key, value)
			}
		}
		processLog.Url = attr.URL
	}

	if len(attr.Query) > 0 {
		query := url.Values{}
		for key, value := range attr.Query {
			query.Add(key, value)
		}
		attr.URL = fmt.Sprintf("%s?%s", attr.URL, query.Encode())
		processLog.QueryString = attr.Query
	}
	var body io.Reader
	// var bodyBytes []byte
	if len(attr.Body) > 0 {
		bodyBytes, err := json.Marshal(attr.Body)
		if err != nil {
			return nil, err
		}
		body = bytes.NewBuffer(bodyBytes)
		processLog.Body = attr.Body
	}

	detailLog.AddOutputRequest(attr.Service, attr.Command, attr.Invoke, processLog, processLog)

	// Create request
	req, err := http.NewRequestWithContext(ctx, string(attr.Method), attr.URL, body)
	if err != nil {
		// detailLog.AutoEnd()
		summaryLog.AddError(attr.Service, attr.Command, "500", err.Error())
		// detailLog.AddInputResponse(attr.Service, attr.Command, attr.Invoke, err.Error(), err.Error(), "http", string(attr.Method))
		return nil, err
	}

	// Set headers
	for key, value := range attr.Headers {
		req.Header.Set(key, value)
	}

	// Set BasicAuth
	if attr.Auth != nil {
		req.SetBasicAuth(attr.Auth.Username, attr.Auth.Password)
	}

	return req, nil
}

func executeRequest(client *http.Client, req *http.Request, attr RequestAttributes) *ApiResponse {
	apiResponse := &ApiResponse{
		Err:        nil,
		Header:     nil,
		Body:       nil,
		Status:     0,
		StatusText: "",
		attr: attrDetailLog{
			Service: attr.Service,
			Command: attr.Command,
			Invoke:  attr.Invoke,
			Method:  attr.Method,
		},
	}

	response, err := client.Do(req)
	if err != nil {
		apiResponse.Status = 500
		apiResponse.StatusText = err.Error()
		apiResponse.Err = err
		return apiResponse
	}
	defer response.Body.Close()

	// Read response body
	bodyBytes, err := io.ReadAll(response.Body)
	if err != nil {
		apiResponse.Status = 500
		apiResponse.StatusText = err.Error()
		apiResponse.Err = err
		return apiResponse
	}

	var body interface{}
	json.Unmarshal(bodyBytes, &body)

	apiResponse.Header = response.Header
	apiResponse.Body = body
	apiResponse.Status = response.StatusCode
	apiResponse.StatusText = response.Status

	// if _, ok := statusSuccess[response.StatusCode]; !ok {
	// 	return nil, fmt.Errorf("unexpected status: %d", response.StatusCode)
	// }

	return apiResponse
}
