package utils

import (
	"path/filepath"

	"github.com/f-rambo/cloud-copilot/internal/conf"
	"github.com/go-kratos/kratos/v2/log"
	"gopkg.in/natefinch/lumberjack.v2"
)

var _ log.Logger = (*logtool)(nil)

type logtool struct {
	logger           log.Logger
	lumberjackLogger *lumberjack.Logger
	conf             *conf.Bootstrap
}

func NewLog(conf *conf.Bootstrap) *logtool {
	logConf := conf.Log
	lumberjackLogger := &lumberjack.Logger{
		Filename:   filepath.Join(GetServerStoragePathByNames("log"), conf.Server.Name+".log"),
		MaxSize:    int(logConf.MaxSize),
		MaxBackups: int(logConf.MaxBackups),
		MaxAge:     int(logConf.MaxAge),
		LocalTime:  true,
	}
	return &logtool{
		logger:           log.NewStdLogger(lumberjackLogger),
		lumberjackLogger: lumberjackLogger,
		conf:             conf,
	}
}

func (l *logtool) Log(level log.Level, keyvals ...any) error {
	return l.logger.Log(level, keyvals...)
}

func (l *logtool) Close() error {
	if l.lumberjackLogger != nil {
		return l.lumberjackLogger.Close()
	}
	return nil
}
