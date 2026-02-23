package setup

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	"go.uber.org/zap"
)

// Run starts a temporary HTTP server serving the setup wizard.
// It blocks until the user completes setup and config.json is written.
func Run(logger *zap.Logger, port int) error {
	ws := &wizardServer{
		logger: logger,
		done:   make(chan struct{}),
	}

	r := mux.NewRouter()
	r.HandleFunc("/", ws.handleIndex).Methods("GET")
	r.HandleFunc("/api/setup/detect-ip", ws.handleDetectIP).Methods("GET")
	r.HandleFunc("/api/setup/client-modes", ws.handleClientModes).Methods("GET")
	r.HandleFunc("/api/setup/test-db", ws.handleTestDB).Methods("POST")
	r.HandleFunc("/api/setup/init-db", ws.handleInitDB).Methods("POST")
	r.HandleFunc("/api/setup/finish", ws.handleFinish).Methods("POST")

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: r,
	}

	logger.Info(fmt.Sprintf("Setup wizard available at http://localhost:%d", port))
	fmt.Printf("\n  >>> Open http://localhost:%d in your browser to configure Erupe <<<\n\n", port)

	// Start the HTTP server in a goroutine.
	errCh := make(chan error, 1)
	go func() {
		if err := srv.ListenAndServe(); err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	// Wait for either completion or server error.
	select {
	case <-ws.done:
		logger.Info("Setup complete, shutting down wizard")
		if err := srv.Shutdown(context.Background()); err != nil {
			logger.Warn("Error shutting down wizard server", zap.Error(err))
		}
		return nil
	case err := <-errCh:
		return fmt.Errorf("setup wizard server error: %w", err)
	}
}
