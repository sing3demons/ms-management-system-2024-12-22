package ms

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sing3demons/profile-service/middleware"

	"go.uber.org/zap"

	_ "github.com/go-sql-driver/mysql" // MySQL driver
	_ "github.com/lib/pq"              // PostgreSQL driver
)

type application struct {
	config Config
	logger *zap.Logger
	router *mux.Router
	conn   *sql.DB
}

type KafkaConfig struct {
	Brokers string
	GroupID string
	TimeOut int
}
type Config struct {
	Addr       string
	Db         DbConfig
	Env        string
	Name       string
	RedisCfg   RedisConfig
	KafkaCfg   KafkaConfig
	LogConfig  LogConfig
	MailServer MailServer
}

type RedisConfig struct {
	Addr    string
	Pw      string
	Db      int
	Enabled bool
}

type SummaryLogConfig struct {
	Name       string `json:"name"`
	RawData    bool   `json:"rawData"`
	LogFile    bool   `json:"logFile"`
	LogConsole bool   `json:"logConsole"`
	LogSummary *zap.Logger
}

type DetailLogConfig struct {
	Name       string `json:"name"`
	RawData    bool   `json:"rawData"`
	LogFile    bool   `json:"logFile"`
	LogConsole bool   `json:"logConsole"`
	LogDetail  *zap.Logger
}

type AppLog struct {
	Name    string `json:"name"`
	LogApp  *zap.Logger
	LogFile bool `json:"logFile"`
}

type LogConfig struct {
	ProjectName string
	Namespace   string
	AppLog      AppLog           `json:"appLog"`
	Detail      DetailLogConfig  `json:"detail"`
	Summary     SummaryLogConfig `json:"summary"`
}

type DbConfig struct {
	Addr         string
	MaxOpenConns int
	MaxIdleConns int
	MaxIdleTime  string
	Driver       string
}

type IMicroservice interface {
	GET(path string, h ServiceHandleFunc)
	POST(path string, h ServiceHandleFunc)
	PUT(path string, h ServiceHandleFunc)
	DELETE(path string, h ServiceHandleFunc)
	PATCH(path string, h ServiceHandleFunc)
	Run() error
	CleanUp()

	Log(tag string, msg string)

	Consume(topic string, h ServiceHandleFunc) error
	NewProducer() *Producer

	ConnDatabase(migrate ...string) *sql.DB
}

func ensureLogDirExists(path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		err := os.MkdirAll(path, os.ModePerm)
		if err != nil {
			return errors.New("failed to create log directory")
		}
	}
	return nil
}

func promHandler() http.Handler {
	prometheus.Register(totalRequests)
	prometheus.Register(responseStatus)
	prometheus.Register(httpDuration)
	reg := prometheus.NewRegistry()
	promHandler := promhttp.HandlerFor(reg, promhttp.HandlerOpts{})
	return promHandler
}

func NewApplication(cfg Config) IMicroservice {
	r := mux.NewRouter()
	r.Handle("/metrics", promHandler())
	r.Use(Recovery)
	r.Use(middleware.Logger)

	// r.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
	// 	w.WriteHeader(http.StatusOK)
	// 	w.Write([]byte("OK"))
	// })

	setupLogging(&cfg)

	if cfg.LogConfig.Namespace == "" {
		cfg.LogConfig.Namespace = "default"
	}

	return &application{
		config: cfg,
		logger: cfg.LogConfig.AppLog.LogApp,
		router: r,
	}
}

func setupLogging(cfg *Config) {
	setupAppLog(cfg)
	setupSummaryLog(cfg)
	setupDetailLog(cfg)
}

func setupAppLog(cfg *Config) {
	if cfg.LogConfig.AppLog.LogFile {
		if cfg.LogConfig.AppLog.Name == "" {
			cfg.LogConfig.AppLog.Name = "./logs/app"
		}
		if err := ensureLogDirExists(cfg.LogConfig.AppLog.Name); err != nil {
			log.Fatal(err)
		}

	}
	cfg.LogConfig.AppLog.LogApp = NewLogger(cfg.LogConfig.AppLog)
}

func setupSummaryLog(cfg *Config) {
	if cfg.LogConfig.Summary.LogFile {
		if cfg.LogConfig.Summary.Name == "" {
			cfg.LogConfig.Summary.Name = "./logs/summary"
		}
		if err := ensureLogDirExists(cfg.LogConfig.Summary.Name); err != nil {
			log.Fatal(err)
		}

		cfg.LogConfig.Summary.LogSummary = NewLogFile(cfg.LogConfig.Summary.Name)
	}
}

func setupDetailLog(cfg *Config) {
	if cfg.LogConfig.Detail.LogFile {
		if cfg.LogConfig.Detail.Name == "" {
			cfg.LogConfig.Detail.Name = "./logs/detail"
		}
		if err := ensureLogDirExists(cfg.LogConfig.Detail.Name); err != nil {
			log.Fatal(err)
		}

		cfg.LogConfig.Detail.LogDetail = NewLogFile(cfg.LogConfig.Detail.Name)
	}
}

func Recovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		defer func() {
			err := recover()
			if err != nil {
				fmt.Println(err) // May be log this error? Send to sentry?

				jsonBody, _ := json.Marshal(map[string]string{
					"error": "There was an internal server error",
				})

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				w.Write(jsonBody)
			}

		}()

		next.ServeHTTP(w, r)

	})
}

func (app *application) ConnDatabase(migrate ...string) *sql.DB {
	db, err := sql.Open(app.config.Db.Driver, app.config.Db.Addr)
	if err != nil {
		log.Fatal(err)
		return nil
	}

	db.SetMaxOpenConns(app.config.Db.MaxOpenConns)
	db.SetMaxIdleConns(app.config.Db.MaxIdleConns)

	duration, err := time.ParseDuration(app.config.Db.MaxIdleTime)
	if err != nil {
		log.Fatal(err)
		return nil
	}
	db.SetConnMaxIdleTime(duration)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err = db.PingContext(ctx); err != nil {
		log.Fatal(err)
		return nil
	}

	app.conn = db
	app.Log("Database", "Database connection established")

	if len(migrate) > 0 {
		// init create table
		for _, m := range migrate {
			_, err := db.Exec(m)
			if err != nil {
				app.logger.Error("Error create table", zap.Error(err))
				os.Exit(1)
			}
		}
	}

	return app.conn
}

func (app *application) Run() error {
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%s", app.config.Addr),
		Handler:      app.router,
		WriteTimeout: time.Second * 30,
		ReadTimeout:  time.Second * 10,
		IdleTimeout:  time.Minute,
	}

	shutdown := make(chan error)

	go func() {
		quit := make(chan os.Signal, 1)

		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		s := <-quit

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		app.logger.Info("shutting down server", zap.String("signal", s.String()))

		shutdown <- srv.Shutdown(ctx)
	}()

	hostName, _ := os.Hostname()
	platform := runtime.GOOS
	arch := runtime.GOARCH
	cpus := runtime.NumCPU()
	totalMemory := runtime.NumGoroutine()
	freeMemory := runtime.NumGoroutine()
	uptime := time.Since(time.Now())
	pid := os.Getpid()

	detail := map[string]interface{}{
		"startTime":    time.Now().Format(time.RFC3339),
		"addr":         app.config.Addr,
		"env":          app.config.Env,
		"app_name":     os.Getenv("SERVICE_NAME"),
		"hostname":     hostName,
		"pid":          fmt.Sprintf("%d", os.Getpid()),
		"platform":     platform,
		"arch":         arch,
		"cpus":         cpus,
		"total_memory": totalMemory,
		"free_memory":  freeMemory,
		"uptime":       uptime,
		"service_pid":  fmt.Sprintf("%d", pid),
		"go_version":   runtime.Version(),
	}
	app.logger.Info(fmt.Sprintf("server is listening on port %s", app.config.Addr), zap.Any("detail ===>", detail))

	err := srv.ListenAndServe()
	if !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	err = <-shutdown
	if err != nil {
		return err
	}

	app.logger.Info("server stopped")

	return nil
}

func (app *application) NewProducer() *Producer {
	producer := NewProducer(app.config.KafkaCfg.Brokers, app)
	return producer
}

func (m *application) Log(tag string, msg string) {
	m.logger.Info(fmt.Sprintf("[%s]: %s", tag, msg))
}

func (m *application) Use(middleware func(http.Handler) http.Handler) {
	m.router.Use(middleware)
}

func (m *application) GET(path string, h ServiceHandleFunc) {
	m.router.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		h(NewHTTPContext(w, r, m))
	}).Methods(http.MethodGet)
}

func (m *application) POST(path string, h ServiceHandleFunc) {
	m.router.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("POST")
		h(NewHTTPContext(w, r, m))
	}).Methods(http.MethodPost)
}

func (m *application) PUT(path string, h ServiceHandleFunc) {
	m.router.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		h(NewHTTPContext(w, r, m))
	}).Methods(http.MethodPut)
}

func (m *application) DELETE(path string, h ServiceHandleFunc) {
	m.router.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		h(NewHTTPContext(w, r, m))
	}).Methods(http.MethodDelete)
}

func (m *application) PATCH(path string, h ServiceHandleFunc) {
	m.router.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		h(NewHTTPContext(w, r, m))
	}).Methods(http.MethodPatch)
}

func (m *application) CleanUp() {
	if m.conn != nil {
		m.conn.Close()
		m.logger.Info("database connection closed")
	}

	if m.config.LogConfig.Summary.LogSummary != nil {
		m.config.LogConfig.Summary.LogSummary.Sync()
	}

	if m.config.LogConfig.Detail.LogDetail != nil {
		m.config.LogConfig.Detail.LogDetail.Sync()
	}

	if m.logger != nil {
		m.logger.Sync()
	}
}
