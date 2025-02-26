package dep

import (
	"fmt"
	"users/internal/conf"

	"github.com/go-kratos/kratos/v2/log"
	"go.uber.org/zap"
)

type LogWrapper func(level log.Level, keyvals ...interface{}) error

func (f LogWrapper) Log(level log.Level, keyvals ...interface{}) error {
	return f(level, keyvals...)
}

func NewZapLogger(bc *conf.Bootstrap) (log.Logger, error) {
	env := bc.GetMetadata().GetEnv().String()
	var cfg zap.Config
	switch env {
	case "PROD":
		cfg = zap.NewProductionConfig()
	case "DEV":
		cfg = zap.NewDevelopmentConfig()
	case "TEST":
		cfg = zap.NewDevelopmentConfig()
	default:
		cfg = zap.NewProductionConfig()
	}
	cfg.OutputPaths = []string{"stderr"}
	logfile := bc.GetLog().GetFilepath()
	if logfile != "" {
		cfg.OutputPaths = append(cfg.OutputPaths, logfile)
	}
	logger, err := cfg.Build()
	if err != nil {
		return nil, err
	}
	return LogWrapper(func(level log.Level, keyvals ...interface{}) error {
		zapLevel := zap.DebugLevel
		switch level {
		case log.LevelDebug:
			zapLevel = zap.DebugLevel
		case log.LevelInfo:
			zapLevel = zap.InfoLevel
		case log.LevelWarn:
			zapLevel = zap.WarnLevel
		case log.LevelError:
			zapLevel = zap.ErrorLevel
		case log.LevelFatal:
			zapLevel = zap.FatalLevel
		}
		var fields []zap.Field
		var msg string
		for i := 0; i < len(keyvals); i += 2 {
			key := fmt.Sprintf("%v", keyvals[i])
			value := fmt.Sprintf("%v", keyvals[i+1])

			if key == "msg" {
				msg = value
			} else if key == "ts" {
				fields = append(fields, zap.String("timestamp", value))
			} else {
				fields = append(fields, zap.String(key, value))
			}
		}
		logger = logger.With(fields...)
		logger = logger.WithOptions(zap.WithCaller(false))
		logger.Sugar().Log(zapLevel, msg)
		//logger.Log(zapLevel, msg, fields...)
		return nil
	}), nil
}
