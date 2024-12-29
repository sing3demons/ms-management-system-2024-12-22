package http_service

import (
	"bytes"
	"context"
	"io"
	"math/rand/v2"
	"net/http"
	"time"
)

type RetryConfig struct {
	MaxAttempts  int
	InitialDelay time.Duration
	MaxDelay     time.Duration
}

func Retry[T any](ctx context.Context, config RetryConfig, operation func() (T, error)) (T, error) {
	var result T
	var err error
	currentDelay := config.InitialDelay

	for attempt := 1; attempt <= config.MaxAttempts; attempt++ {
		if ctx.Err() != nil {
			return result, ctx.Err()
		}

		result, err = operation()
		if err == nil {
			return result, nil
		}

		if attempt == config.MaxAttempts {
			return result, err
		}

		jitter := time.Duration(rand.Float64() * float64(currentDelay))
		currentDelay += jitter
		if currentDelay > config.MaxDelay {
			currentDelay = config.MaxDelay
		}

		select {
		case <-ctx.Done():
			return result, ctx.Err()
		case <-time.After(currentDelay):
			currentDelay *= 2
		}
	}

	return result, err
}

// config := RetryConfig{
//     MaxAttempts:  3,
//     InitialDelay: 100 * time.Millisecond,
//     MaxDelay:     1 * time.Second,
// }

// result, err := Retry(context.Background(), config, func() (string, error) {
//     return "success", nil
// })
// fmt.Println(result, err)

type RetryRoundTripper struct {
	next   http.RoundTripper
	config RetryConfig
}

func NewRetryRoundTripper(next http.RoundTripper, config RetryConfig) *RetryRoundTripper {
	if next == nil {
		next = http.DefaultTransport
	}
	return &RetryRoundTripper{
		next:   next,
		config: config,
	}
}

func (rrt *RetryRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	operation := func() (*http.Response, error) {
		reqCopy := req.Clone(req.Context())
		if req.Body != nil {
			bodyBytes, err := io.ReadAll(req.Body)
			if err != nil {
				return nil, err
			}
			reqCopy.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
			req.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
		}

		resp, err := rrt.next.RoundTrip(reqCopy)
		if err != nil {
			return nil, err
		}
		resp.Body.Close()

		return resp, nil
	}

	return Retry(req.Context(), rrt.config, operation)
}

// config := RetryConfig{
//     MaxAttempts:  3,
//     InitialDelay: 100 * time.Millisecond,
//     MaxDelay:     1 * time.Second,
// }

// transport := NewRetryRoundTripper(
//     http.DefaultTransport,
//     config,
// )

// client := &http.Client{
//     Transport: transport,
// }

// req, _ := http.NewRequest("GET", "https://api.example.com/data", nil)
// resp, err := client.Do(req)
// fmt.Println(resp, err)
