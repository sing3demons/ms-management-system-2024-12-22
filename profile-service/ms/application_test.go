package ms

import (
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

var cfg = Config{
	Addr: "8080",
	Db: DbConfig{
		Addr:         "mock_db",
		MaxOpenConns: 10,
		MaxIdleConns: 5,
		MaxIdleTime:  "15m",
		Driver:       "sqlmock",
	},
	Env:  "development",
	Name: "profile-service",
	LogConfig: LogConfig{
		ProjectName: "car-management-system",
		Namespace:   "default",
		AppLog: AppLog{
			Name:    "./logs/app",
			LogFile: false,
			LogApp:  zap.NewNop(),
		},
		Detail: DetailLogConfig{
			Name:       "./logs/detail",
			LogFile:    false,
			LogConsole: true,
			LogDetail:  zap.NewNop(),
		},
		Summary: SummaryLogConfig{
			Name:       "./logs/summary",
			LogFile:    false,
			LogConsole: true,
			LogSummary: zap.NewNop(),
		},
	},
}

func TestNewApplication(t *testing.T) {
	app := NewApplication(cfg)

	assert.NotNil(t, app)
	assert.NotNil(t, cfg, app.(*application).config)
	assert.IsType(t, &mux.Router{}, app.(*application).router)
	assert.NotNil(t, cfg.LogConfig.AppLog.LogApp, app.(*application).logger)
	assert.NotNil(t, app.(*application).router)
}
