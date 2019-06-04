package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var Logger, _ = zap.Config{
	Level:             zap.NewAtomicLevelAt(zapcore.DebugLevel),
	Development:       false,
	DisableCaller:     true,
	DisableStacktrace: true,
	Encoding:          "console",
	EncoderConfig: zapcore.EncoderConfig{
		TimeKey:        "T",
		LevelKey:       "L",
		NameKey:        "N",
		CallerKey:      "C",
		MessageKey:     "M",
		StacktraceKey:  "S",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.CapitalColorLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.StringDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	},
	OutputPaths:      []string{"stderr"},
	ErrorOutputPaths: []string{"stderr"},
}.Build()
