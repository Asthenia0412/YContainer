package sidecar

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type AccessLogRecord struct {
	Timestamp    time.Time `json:"timestamp"`
	Method       string    `json:"method"`
	Path         string    `json:"path"`
	StatusCode   int       `json:"status_code"`
	Latency      string    `json:"latency"`
	ClientIP     string    `json:"client_ip"`
	RequestSize  int64     `json:"request_size"`
	ResponseSize int64     `json:"response_size"`
	UserAgent    string    `json:"user_agent"`
}

type AccessLogger struct {
	config     LoggingConfig
	records    []AccessLogRecord
	mu         sync.Mutex
	logger     *log.Logger
}

type LoggingConfig struct {
	Enabled    bool
	LogDir     string
	AccessLog  bool
	LatencyLog bool
}

func NewAccessLogger(config LoggingConfig) *AccessLogger {
	logger := log.New(os.Stdout, "[ACCESS] ", log.LstdFlags)

	if config.LogDir != "" {
		os.MkdirAll(config.LogDir, 0755)
		logPath := filepath.Join(config.LogDir, "access.log")
		f, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err == nil {
			logger = log.New(f, "", log.LstdFlags)
		}
	}

	return &AccessLogger{
		config:  config,
		logger:  logger,
		records: make([]AccessLogRecord, 0),
	}
}

func (al *AccessLogger) Log(record AccessLogRecord) {
	if !al.config.AccessLog {
		return
	}

	al.mu.Lock()
	al.records = append(al.records, record)
	if len(al.records) > 10000 {
		al.records = al.records[len(al.records)-1000:]
	}
	al.mu.Unlock()

	al.logger.Printf("%s %s %d %s from %s",
		record.Method, record.Path, record.StatusCode, record.Latency, record.ClientIP)
}

func (al *AccessLogger) Flush(path string) error {
	al.mu.Lock()
	defer al.mu.Unlock()

	data, err := json.MarshalIndent(al.records, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal access logs: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("write access logs: %w", err)
	}

	return nil
}