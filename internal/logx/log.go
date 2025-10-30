package logx

import "go.uber.org/zap"

func New(env string) *zap.Logger {
	if env == "prod" {
		l, _ := zap.NewProduction()
		return l
	}
	l, _ := zap.NewDevelopment()
	return l
}
