package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	_ "net/http/pprof" // register pprof handlers on the default mux; see startPprofServer
	"os"
	"os/signal"
	"syscall"
	"time"

	"hotspot/backend-go/internal/bootstrap"
	"hotspot/backend-go/internal/platform/config"
)

func main() {
	cfg := config.Load()
	if err := cfg.Validate(); err != nil {
		log.Fatalf("invalid config: %v", err)
	}
	// Always run the insecure-defaults scan. In production Validate() has
	// already aborted on criticals; in dev it returns warnings so operators
	// can't miss them in the startup log.
	warnInsecureDefaults(cfg)

	app, err := bootstrap.NewApp(cfg)
	if err != nil {
		log.Fatalf("bootstrap app: %v", err)
	}

	// pprof is mounted on a separate server bound to localhost only, so the
	// profiling endpoints are never reachable from outside the host. The
	// address is configurable via PPROF_ADDR (default 127.0.0.1:6060); set
	// PPROF_ADDR= to disable.
	startPprofServer(envOrDefault("PPROF_ADDR", "127.0.0.1:6060"))

	server := &http.Server{
		Addr:              cfg.HTTPAddr(),
		Handler:           app.Router(),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		app.Logger().Info("http server listening", "addr", cfg.HTTPAddr())
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen and serve: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("shutdown server: %v", err)
	}
	app.Shutdown()
}

// warnInsecureDefaults prints a loud banner to stderr whenever the loaded
// config still carries a known default/placeholder value. In production the
// process has already refused to boot via Validate(); this is the dev-mode
// safety net so a forgotten default is impossible to miss in the log tail.
func warnInsecureDefaults(cfg config.Config) {
	findings := cfg.CheckInsecureDefaultsPublic()
	if len(findings) == 0 {
		return
	}
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!")
	fmt.Fprintln(os.Stderr, "!! INSECURE DEFAULT CONFIGURATION DETECTED")
	fmt.Fprintln(os.Stderr, "!! The following fields still hold known placeholder/dev values.")
	fmt.Fprintln(os.Stderr, "!! Change them before this process serves real traffic.")
	fmt.Fprintln(os.Stderr, "!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!")
	for _, f := range findings {
		fmt.Fprintf(os.Stderr, "!! [%s] %s\n", f.Severity, f.Message)
	}
	fmt.Fprintln(os.Stderr, "!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!!")
	fmt.Fprintln(os.Stderr, "")
}

// startPprofServer serves net/http/pprof on addr. Bound to localhost by
// default so it is unreachable from outside the host. Empty addr disables it.
func startPprofServer(addr string) {
	if addr == "" {
		return
	}
	srv := &http.Server{
		Addr:              addr,
		Handler:           http.DefaultServeMux,
		ReadHeaderTimeout: 10 * time.Second,
	}
	go func() {
		log.Printf("pprof server listening on http://%s/debug/pprof/", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("pprof server stopped: %v", err)
		}
	}()
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

