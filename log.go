package ptcp

import "log4go"

type LogConfig struct {
	LogPrefix string
	ConsoleLogLevel int
	SysLogLevel int
	FileLogLevel int
	LogFile string
}

func NewLogger(logConfig *LogConfig) (logger log4go.Logger) {
	logger = log4go.NewDefaultLogger(log4go.DEBUG)
	if logConfig.ConsoleLogLevel > 0 {
		logger.AddFilter("stdout", log4go.LogLevel(logConfig.ConsoleLogLevel), log4go.NewConsoleLogWriter())
	} else {
		logger["stdout"] = nil, false
	}
	
	if logConfig.FileLogLevel > 0 {
		logger.AddFilter("logfile", log4go.LogLevel(logConfig.FileLogLevel), log4go.NewFileLogWriter(logConfig.LogFile, false)) 
	}
	
	if logConfig.SysLogLevel > 0 {
		logger.AddFilter("syslog", log4go.LogLevel(logConfig.SysLogLevel), log4go.NewSysLogWriter()) 
	}
	return
}