package proxy

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const (
	DefaultMaxLogSize = 10 * 1024 * 1024 // 10MB
	LogFilePrefix     = "requests"
	LogFileSuffix     = ".jsonl.gz"
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

// RequestLogger writes HTTP request/response logs to gzip-compressed files
type RequestLogger struct {
	dir       string
	mu        sync.Mutex
	file      *os.File
	gzWriter  *gzip.Writer
	written   int64
	fileIndex int
}

// NewRequestLogger creates a new request logger
func NewRequestLogger(dir string) (*RequestLogger, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	rl := &RequestLogger{
		dir: dir,
	}

	if err := rl.rotate(); err != nil {
		return nil, err
	}

	return rl, nil
}

// Log writes a request/response pair to the log
func (rl *RequestLogger) Log(entry *RequestLog) error {
	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("failed to marshal log entry: %w", err)
	}
	data = append(data, '\n')

	n, err := rl.gzWriter.Write(data)
	if err != nil {
		return fmt.Errorf("failed to write log entry: %w", err)
	}
	rl.written += int64(n)

	// Flush to ensure data is written
	return rl.gzWriter.Flush()
}

// LogRequest captures request details and returns a function to log the response
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

func (rl *RequestLogger) rotate() error {
	// Close existing file if open
	if rl.gzWriter != nil {
		_ = rl.gzWriter.Close()
	}
	if rl.file != nil {
		_ = rl.file.Close()
	}

	// Find next available file index
	rl.fileIndex = rl.findNextIndex()

	filename := filepath.Join(rl.dir, fmt.Sprintf("%s_%s_%04d%s",
		LogFilePrefix,
		time.Now().Format("20060102"),
		rl.fileIndex,
		LogFileSuffix,
	))

	file, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return fmt.Errorf("failed to create log file: %w", err)
	}

	rl.file = file
	rl.gzWriter = gzip.NewWriter(file)
	rl.written = 0

	return nil
}

func (rl *RequestLogger) findNextIndex() int {
	today := time.Now().Format("20060102")
	pattern := filepath.Join(rl.dir, fmt.Sprintf("%s_%s_*%s", LogFilePrefix, today, LogFileSuffix))
	matches, _ := filepath.Glob(pattern)
	return len(matches)
}

// Close closes the logger
func (rl *RequestLogger) Close() error {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	var errs []error
	if rl.gzWriter != nil {
		if err := rl.gzWriter.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	if rl.file != nil {
		if err := rl.file.Close(); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return errs[0]
	}
	return nil
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
