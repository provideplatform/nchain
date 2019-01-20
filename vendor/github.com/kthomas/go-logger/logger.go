package logger

import (
	"fmt"
	"os"
	"sync"

	"github.com/op/go-logging"
)

type Logger struct {
	console bool
	level logging.Level
	logger *logging.Logger
	mutex *sync.Mutex
	prefix string
	template string
}

func (lg *Logger) configure() {
	logging.Reset()

	formatter, err := logging.NewStringFormatter(lg.template)
	if err == nil {
		logging.SetFormatter(formatter)
	} else {
		logging.SetFormatter(logging.GlogFormatter)
	}

	var logPrefix = lg.prefix
	if len(lg.prefix) > 0 {
		logPrefix = fmt.Sprintf("%s ", logPrefix)
	}

	if lg.console {
		backend := logging.NewLogBackend(os.Stdout, logPrefix, 0)
		logging.SetBackend(backend)
	} else {
		syslogBackend, err := logging.NewSyslogBackend(logPrefix)
		if err != nil {
			logging.SetBackend(syslogBackend)
		}
	}
}

func (lg *Logger) Clone() *Logger {
	return &Logger{
		console: lg.console,
		level: lg.level,
		logger: lg.logger,
		mutex: &sync.Mutex{},
		prefix: lg.prefix,
		template: lg.template,
	}
}

func (lg *Logger) Critical(msg string) {
	if lg.level >= logging.CRITICAL {
		lg.logger.Warning(msg)
	}
}

func (lg *Logger) Criticalf(msg string, v ...interface{}) {
	if lg.level >= logging.WARNING {
		lg.logger.Warningf(msg, v...)
	}
}

func (lg *Logger) Debug(msg string) {
	if lg.level >= logging.DEBUG {
		lg.logger.Debug(msg)
	}
}

func (lg *Logger) Debugf(msg string, v ...interface{}) {
	if lg.level >= logging.DEBUG {
		lg.logger.Debugf(msg, v...)
	}
}

func (lg *Logger) Error(msg string) {
	if lg.level >= logging.ERROR {
		lg.logger.Error(msg)
	}
}

func (lg *Logger) Errorf(msg string, v ...interface{}) {
	if lg.level >= logging.ERROR {
		lg.logger.Errorf(msg, v...)
	}
}

func (lg *Logger) Info(msg string) {
	if lg.level >= logging.INFO {
		lg.logger.Info(msg)
	}
}

func (lg *Logger) Infof(msg string, v ...interface{}) {
	if lg.level >= logging.INFO {
		lg.logger.Infof(msg, v...)
	}
}
func (lg *Logger) LogOnError(err error, s string) bool {
	hasErr := false
	if err != nil {
		msg := fmt.Sprintf("Error: %s", err)
		if s != "" {
			msg = fmt.Sprintf("%s; %s", msg, s)
		}
		lg.Errorf(msg)
		hasErr = true
	}
	return hasErr
}

func (lg *Logger) Notice(msg string) {
	if lg.level >= logging.NOTICE {
		lg.logger.Notice(msg)
	}
}

func (lg *Logger) Noticef(msg string, v ...interface{}) {
	if lg.level >= logging.NOTICE {
		lg.logger.Noticef(msg, v...)
	}
}

func (lg *Logger) Panicf(msg string, v ...interface{}) {
	lg.logger.Panicf(msg, v...)
}

func (lg *Logger) PanicOnError(err error, s string) {
	if err != nil {
		msg := fmt.Sprintf("Error: %s", err)
		if s != "" {
			msg = fmt.Sprintf("%s; %s", msg, s)
		}
		lg.Panicf(msg)
	}
}

func (lg *Logger) SetTemplate(template string) {
	lg.mutex.Lock()
	defer lg.mutex.Unlock()
	lg.template = template
	lg.configure()
}

func (lg *Logger) Warning(msg string) {
	if lg.level >= logging.WARNING {
		lg.logger.Warning(msg)
	}
}

func (lg *Logger) Warningf(msg string, v ...interface{}) {
	if lg.level >= logging.WARNING {
		lg.logger.Warningf(msg, v...)
	}
}

func NewLogger(prefix string, level string, console bool) *Logger {
	var logLevel logging.Level

	switch level {
	case "CRITICAL":
		logLevel = logging.CRITICAL
	case "ERROR":
		logLevel = logging.ERROR
	case "WARNING":
		logLevel = logging.WARNING
	case "NOTICE":
		logLevel = logging.NOTICE
	case "INFO":
		logLevel = logging.INFO
	case "DEBUG":
		logLevel = logging.DEBUG
	}

	lg := Logger{}
	lg.console = console
	lg.level = logLevel
	lg.logger = logging.MustGetLogger(prefix)
	lg.mutex = &sync.Mutex{}
	lg.prefix = prefix

	lg.configure()

	return &lg
}
