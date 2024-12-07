package internal

import (
	"github.com/sirupsen/logrus"
	"gopkg.in/natefinch/lumberjack.v2"
	"io"
	"os"
	"path/filepath"
)

func InitLog() {
	formatter := logrus.TextFormatter{
		ForceColors:               true,
		EnvironmentOverrideColors: true,
		TimestampFormat:           "2006-01-02 15:04:05",
		FullTimestamp:             true,
	}
	logrus.SetFormatter(&formatter)
	setLog(logrus.StandardLogger())
	logConfig := Config.Log
	if logConfig.Enable {
		process, _ := os.Executable()
		LogBaseDir := filepath.Dir(process)
		if Config.Server.Debug {
			LogBaseDir, _ = os.Getwd()
		}
		var w io.Writer = &lumberjack.Logger{
			Filename:   LogBaseDir + "/logs/" + logConfig.FileName,
			MaxSize:    logConfig.MaxSize, // megabytes
			MaxBackups: logConfig.MaxBackups,
			MaxAge:     logConfig.MaxAge,   //days
			Compress:   logConfig.Compress, // disabled by default
		}
		w = io.MultiWriter(os.Stdout, w)
		logrus.SetOutput(w)
	}
}

func setLog(l *logrus.Logger) {
	if Config.Server.Debug {
		l.SetLevel(logrus.DebugLevel)
		l.SetReportCaller(true)
	} else {
		l.SetLevel(logrus.InfoLevel)
		l.SetReportCaller(false)
	}
}
