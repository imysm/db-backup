package logger

import (
	"io"
	"log"
	"os"
	"strings"
)

// LogLevel 日志级别
type LogLevel int

const (
	DebugLevel LogLevel = iota
	InfoLevel
	WarnLevel
	ErrorLevel
)

var (
	// Info 标准输出日志
	Info *log.Logger
	// Warning 警告日志
	Warning *log.Logger
	// Error 错误日志
	Error *log.Logger

	currentLevel LogLevel
)

// Init 初始化日志
func Init(logFile string, level string) error {
	currentLevel = parseLevel(level)

	var writers []io.Writer

	// 总是输出到控制台
	writers = append(writers, os.Stdout)

	// 如果指定了日志文件，同时输出到文件
	if logFile != "" {
		file, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return err
		}
		writers = append(writers, file)
	}

	multiWriter := io.MultiWriter(writers...)

	Info = log.New(newLevelWriter(multiWriter, InfoLevel), "[INFO] ", log.Ldate|log.Ltime|log.Lshortfile)
	Warning = log.New(newLevelWriter(multiWriter, WarnLevel), "[WARN] ", log.Ldate|log.Ltime|log.Lshortfile)
	Error = log.New(newLevelWriter(multiWriter, ErrorLevel), "[ERROR] ", log.Ldate|log.Ltime|log.Lshortfile)

	return nil
}

// InitDefault 使用默认参数初始化（兼容旧调用）
func InitDefault() {
	Init("", "info")
}

// parseLevel 解析日志级别字符串
func parseLevel(level string) LogLevel {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "debug":
		return DebugLevel
	case "warn", "warning":
		return WarnLevel
	case "error":
		return ErrorLevel
	default:
		return InfoLevel
	}
}

// levelWriter 带级别过滤的日志写入器
type levelWriter struct {
	writer io.Writer
	level  LogLevel
}

func newLevelWriter(w io.Writer, level LogLevel) *levelWriter {
	return &levelWriter{writer: w, level: level}
}

func (w *levelWriter) Write(p []byte) (n int, err error) {
	if w.level >= currentLevel {
		return w.writer.Write(p)
	}
	return len(p), nil
}

func init() {
	// 默认初始化（仅控制台，info 级别）
	InitDefault()
}
