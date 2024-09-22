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
}

func NewLog(conf *conf.Bootstrap) *logtool {
	logConf := conf.Log

	if conf.Server.Debug {
		return &logtool{
			logger: log.DefaultLogger,
		}
	}
	lumberjackLogger := &lumberjack.Logger{
		Filename:   getFilePath(conf.Server.Name),
		MaxSize:    int(logConf.MaxSize),
		MaxBackups: int(logConf.MaxBackups),
		MaxAge:     int(logConf.MaxAge),
		LocalTime:  true,
	}
	return &logtool{
		logger:           log.NewStdLogger(lumberjackLogger),
		lumberjackLogger: lumberjackLogger,
	}
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

func GetLogContenteKeyvals() []interface{} {
	return []interface{}{
		"ts", log.Timestamp("2006-01-02 15:04:05"),
		"caller", log.DefaultCaller,
	}
}

func getFilePath(filename string) string {
	logPath, err := GetPackageStorePathByNames(logPackageName)
	if err != nil {
		panic(err)
	}
	err = os.MkdirAll(logPath, 0755)
	if err != nil {
		panic(err)
	}
	return filepath.Join(logPath, fmt.Sprintf("%s.%s", filename, logPackageName))
}
