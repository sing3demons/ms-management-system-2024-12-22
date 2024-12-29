package mlog

import (
	"context"
	"fmt"

	"github.com/sing3demons/logger-kp/logger"
)

// func NewLog(c context.Context) *zap.Logger {
// 	switch logger := c.Value(key).(type) {
// 	case *zap.Logger:
// 		return logger
// 	default:
// 		return zap.NewNop()
// 	}
// }

const (
	DetailKey  = "detailLog"
	SummaryKey = "summaryLog"
)

func newDetailLog(ctx context.Context) logger.DetailLog {
	switch log := ctx.Value(DetailKey).(type) {
	case logger.DetailLog:
		fmt.Println("===============> detail log")
		return log
	default:
		return logger.NewDetailLog("", "", "")
	}
}

func newSummaryLog(ctx context.Context) logger.SummaryLog {
	switch log := ctx.Value(SummaryKey).(type) {
	case logger.SummaryLog:
		fmt.Println("===============> summary log")
		return log
	default:
		return logger.NewSummaryLog("", "", "")
	}
}

func Log(ctx context.Context) (logger.DetailLog, logger.SummaryLog) {
	return newDetailLog(ctx), newSummaryLog(ctx)
}
