package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func New(serviceName string, env string) (*zap.Logger, error) {
	cfg := zap.NewProductionConfig()
	if env == "local" {
		cfg = zap.NewDevelopmentConfig()
	}

	// sempre garantir que serviço e env entrem como campos padrão
	cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	l, err := cfg.Build(
		zap.Fields(
			zap.String("service", serviceName),
			zap.String("env", env),
		),
	)
	if err != nil {
		return nil, err
	}
	return l, nil
}
