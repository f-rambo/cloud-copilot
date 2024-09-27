package utils

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/f-rambo/ocean/internal/conf"
	"github.com/go-kratos/kratos/v2/log"
	"gopkg.in/natefinch/lumberjack.v2"
)

const logPackageName = "log"

var _ log.Logger = (*logtool)(nil)

type logtool struct {
	logger           log.Logger
	lumberjackLogger *lumberjack.Logger
	conf             *conf.Bootstrap
}

func NewLog(conf *conf.Bootstrap) (*logtool, error) {
	logConf := conf.Log

	if conf.Server.Debug {
		return &logtool{
			logger: log.DefaultLogger,
			conf:   conf,
		}, nil
	}
	logFilePath, err := GetLogFilePath(conf.Server.Name)
	if err != nil {
		return nil, err
	}
	lumberjackLogger := &lumberjack.Logger{
		Filename:   logFilePath,
		MaxSize:    int(logConf.MaxSize),
		MaxBackups: int(logConf.MaxBackups),
		MaxAge:     int(logConf.MaxAge),
		LocalTime:  true,
	}
	return &logtool{
		logger:           log.NewStdLogger(lumberjackLogger),
		lumberjackLogger: lumberjackLogger,
		conf:             conf,
	}, nil
}

func (l *logtool) Log(level log.Level, keyvals ...interface{}) error {
	return l.logger.Log(level, keyvals...)
}

func (l *logtool) Close() error {
	if l.lumberjackLogger != nil {
		return l.lumberjackLogger.Close()
	}
	return nil
}

func (l *logtool) GetLogContenteKeyvals() []interface{} {
	if !l.conf.Server.Debug {
		return []interface{}{}
	}
	return []interface{}{
		"ts", log.DefaultTimestamp,
		"caller", log.DefaultCaller,
	}
}

func GetLogFilePath(filename string) (string, error) {
	logPath, err := GetPackageStorePathByNames(logPackageName)
	if err != nil {
		return "", err
	}
	err = os.MkdirAll(logPath, 0755)
	if err != nil {
		return "", err
	}
	return filepath.Join(logPath, fmt.Sprintf("%s.%s", filename, logPackageName)), nil
}
