package logx

import (
	"fmt"

	"go.uber.org/zap"
)

const prodEnv = "prod"

func New(env string) (*zap.Logger, error) {
	if env == prodEnv {
		l, err := zap.NewProduction()
		if err != nil {
			return nil, fmt.Errorf("new production logger: %w", err)
		}
		return l, nil
	}
	l, err := zap.NewDevelopment()
	if err != nil {
		return nil, fmt.Errorf("new development logger: %w", err)
	}
	return l, nil
}
