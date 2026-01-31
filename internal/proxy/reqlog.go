package proxy

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"sync"
	"time"
)

const (
	RequestLogPrefix = "requests"
	RequestLogSuffix = ".jsonl.gz"
)

// RequestLog represents a logged HTTP request/response pair
type RequestLog struct {
	Timestamp       time.Time           `json:"ts"`
	Method          string              `json:"method"`
	URL             string              `json:"url"`
	RequestHeaders  map[string][]string `json:"req_headers,omitempty"`
	RequestBody     []byte              `json:"req_body,omitempty"`
	StatusCode      int                 `json:"status,omitempty"`
	ResponseHeaders map[string][]string `json:"resp_headers,omitempty"`
	ResponseBody    []byte              `json:"resp_body,omitempty"`
	Duration        time.Duration       `json:"duration_ns,omitempty"`
	Error           string              `json:"error,omitempty"`
}

// RequestLogger writes HTTP request/response logs to rotating gzip-compressed files
type RequestLogger struct {
	writer *RotatingFileWriter
	mu     sync.Mutex
}

// NewRequestLogger creates a new request logger
func NewRequestLogger(dir string) (*RequestLogger, error) {
	writer, err := NewRotatingFileWriter(RotatingFileWriterConfig{
		Dir:    dir,
		Prefix: RequestLogPrefix,
		Suffix: RequestLogSuffix,
	})
	if err != nil {
		return nil, err
	}

	return &RequestLogger{
		writer: writer,
	}, nil
}

// Log writes a request/response pair to the log
func (rl *RequestLogger) Log(entry *RequestLog) error {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	data, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	data = append(data, '\n')

	_, err = rl.writer.Write(data)
	return err
}

// LogRequest captures request details and returns a log entry
func (rl *RequestLogger) LogRequest(req *http.Request) (*RequestLog, []byte) {
	entry := &RequestLog{
		Timestamp:      time.Now(),
		Method:         req.Method,
		URL:            req.URL.String(),
		RequestHeaders: cloneHeaders(req.Header),
	}

	// Read and restore request body
	var reqBody []byte
	if req.Body != nil {
		reqBody, _ = io.ReadAll(req.Body)
		_ = req.Body.Close()
		req.Body = io.NopCloser(bytes.NewReader(reqBody))
		entry.RequestBody = reqBody
	}

	return entry, reqBody
}

// LogResponse completes the log entry with response details
func (rl *RequestLogger) LogResponse(entry *RequestLog, resp *http.Response, startTime time.Time) []byte {
	entry.Duration = time.Since(startTime)

	if resp == nil {
		entry.Error = "no response"
		return nil
	}

	entry.StatusCode = resp.StatusCode
	entry.ResponseHeaders = cloneHeaders(resp.Header)

	// Read and restore response body
	var respBody []byte
	if resp.Body != nil {
		respBody, _ = io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		resp.Body = io.NopCloser(bytes.NewReader(respBody))
		entry.ResponseBody = respBody
	}

	return respBody
}

// Close closes the logger
func (rl *RequestLogger) Close() error {
	return rl.writer.Close()
}

func cloneHeaders(h http.Header) map[string][]string {
	if h == nil {
		return nil
	}
	clone := make(map[string][]string, len(h))
	for k, v := range h {
		clone[k] = append([]string(nil), v...)
	}
	return clone
}
