package logger

import (
	"io"
	"log"
	"os"
	"strings"
	"sync"

	"github.com/fatih/color"
)

// Logger struct to hold leveled loggers and configuration
type Logger struct {
	infoLogger  *log.Logger
	errorLogger *log.Logger
	debugLogger *log.Logger
	output      io.Writer
	level       LogLevel
	mutex       sync.Mutex
}

// LogLevel defines the logging levels
type LogLevel int

const (
	DEBUG LogLevel = iota
	INFO
	ERROR
)

// Global logger instance
var GlobalLogger *Logger
var once sync.Once

// InitLogger initializes the global logger with the specified output and log level
func InitLogger(output io.Writer, level string) {
	once.Do(func() {
		if output == nil {
			output = os.Stdout
		}

		logLevel := INFO
		switch strings.ToUpper(level) {
		case "DEBUG":
			logLevel = DEBUG
		case "ERROR":
			logLevel = ERROR
		case "INFO":
			logLevel = INFO
		}

		GlobalLogger = &Logger{
			infoLogger:  log.New(output, color.GreenString("INFO: "), log.Ldate|log.Ltime|log.Lshortfile),
			errorLogger: log.New(output, color.RedString("ERROR: "), log.Ldate|log.Ltime|log.Lshortfile),
			debugLogger: log.New(output, color.BlueString("DEBUG: "), log.Ldate|log.Ltime|log.Lshortfile),
			output:      output,
			level:       logLevel,
		}
	})
}

// Println logs a message at the INFO level
func (l *Logger) Println(v ...interface{}) {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	if l.level <= INFO {
		l.infoLogger.Println(v...)
	}
}

// Printf logs a formatted message at the INFO level
func (l *Logger) Printf(format string, v ...interface{}) {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	if l.level <= INFO {
		l.infoLogger.Printf(format, v...)
	}
}

// Error logs a message at the ERROR level
func (l *Logger) Error(v ...interface{}) {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	if l.level <= ERROR {
		l.errorLogger.Println(v...)
	}
}

// Errorf logs a formatted message at the ERROR level
func (l *Logger) Errorf(format string, v ...interface{}) {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	if l.level <= ERROR {
		l.errorLogger.Printf(format, v...)
	}
}

// Debug logs a message at the DEBUG level
func (l *Logger) Debug(v ...interface{}) {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	if l.level <= DEBUG {
		l.debugLogger.Println(v...)
	}
}

// Debugf logs a formatted message at the DEBUG level
func (l *Logger) Debugf(format string, v ...interface{}) {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	if l.level <= DEBUG {
		l.debugLogger.Printf(format, v...)
	}
}
