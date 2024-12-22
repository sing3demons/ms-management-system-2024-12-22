package ms

// type microservice struct {
// 	r   *mux.Router
// 	Log *zap.Logger
// }

// func NewMicroservice() *microservice {
// 	r := mux.NewRouter()
// 	newLog := NewLogger()
// r.Use(middleware.Logger)
// 	return &microservice{r, newLog}
// }

// func (m *microservice) Start() {
// 	port := os.Getenv("PORT")
// 	svc := &http.Server{
// 		Addr:    fmt.Sprintf(":%s", port),
// 		Handler: m.r,
// 	}

// 	stopServer := make(chan os.Signal, 1)
// 	signal.Notify(stopServer, syscall.SIGINT, syscall.SIGTERM)
// 	defer signal.Stop(stopServer)
// 	defer m.Log.Sync()
// 	// channel to listen for errors coming from the listener.
// 	serverErrors := make(chan error, 1)
// 	var wg sync.WaitGroup
// 	wg.Add(1)
// 	go func(wg *sync.WaitGroup) {
// 		defer wg.Done()
// 		m.Log.Info(fmt.Sprintf("server is listening on port %s", port))
// 		serverErrors <- svc.ListenAndServe()
// 	}(&wg)

// 	// blocking run and waiting for shutdown.
// 	select {
// 	case err := <-serverErrors:
// 		log.Fatalf("error: starting rest api server: %v\n", err)
// 	case <-stopServer:
// 		fmt.Println("server received stop signal")
// 		// asking listener to shutdown
// 		err := svc.Shutdown(context.Background())
// 		if err != nil {
// 			log.Fatalf("graceful shutdown did not complete: %v\n", err)
// 		}
// 		wg.Wait()
// 		fmt.Println("server was shut down gracefully")
// 	}
// }

// func (m *microservice) Use(middleware func(http.Handler) http.Handler) {
// 	m.r.Use(middleware)
// }

// func (m *microservice) GET(path string, f func(http.ResponseWriter, *http.Request)) *mux.Route {
// 	return m.r.HandleFunc(path, f).Methods(http.MethodGet)
// }

// func (m *microservice) POST(path string, f func(http.ResponseWriter, *http.Request)) *mux.Route {
// 	return m.r.HandleFunc(path, f).Methods(http.MethodPost)
// }

// func (m *microservice) PUT(path string, f func(http.ResponseWriter, *http.Request)) *mux.Route {
// 	return m.r.HandleFunc(path, f).Methods(http.MethodPut)
// }

// func (m *microservice) DELETE(path string, f func(http.ResponseWriter, *http.Request)) *mux.Route {
// 	return m.r.HandleFunc(path, f).Methods(http.MethodDelete)
// }

// func (m *microservice) PATCH(path string, f func(http.ResponseWriter, *http.Request)) *mux.Route {
// 	return m.r.HandleFunc(path, f).Methods(http.MethodPatch)
// }
