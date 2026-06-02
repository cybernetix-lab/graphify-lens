package main

import (
	"flag"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/cybernetix-lab/graphify-lens/internal/api"
	"github.com/cybernetix-lab/graphify-lens/internal/config"
	"github.com/cybernetix-lab/graphify-lens/internal/scheduler"
	"github.com/cybernetix-lab/graphify-lens/web"
)

func main() {
	configPath := flag.String("config", "", "path to config file")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	canonicalPath := config.CanonicalPath()
	log.Printf("[config] canonical path: %s", canonicalPath)

	if err := ensureDirs(cfg); err != nil {
		log.Fatalf("ensure dirs: %v", err)
	}

	sched, err := scheduler.New(cfg)
	if err != nil {
		log.Fatalf("init scheduler: %v", err)
	}
	sched.Start()
	defer sched.Stop()

	handler := api.NewHandler(sched, cfg, canonicalPath)

	mux := http.NewServeMux()
	handler.Register(mux)

	staticFS, err := fs.Sub(web.Static, "static")
	if err != nil {
		log.Fatalf("static fs: %v", err)
	}
	mux.Handle("/", http.FileServer(http.FS(staticFS)))

	server := &http.Server{
		Addr:    addr(cfg.Port),
		Handler: withCORS(mux),
	}

	go func() {
		log.Printf("[server] listening on %s", server.Addr)
		log.Printf("[server] work_dirs=%v", cfg.WorkDirs)
		log.Printf("[server] auto_commit=%v interval=%s", cfg.GitAutoCommit, cfg.CommitInterval)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("[server] shutting down...")
}

func ensureDirs(cfg *config.Config) error {
	dirs := []string{
		cfg.QualityHistory,
		cfg.DataDir,
	}
	for _, d := range cfg.WorkDirs {
		dirs = append(dirs, d, filepath.Join(d, "graphify-out"))
	}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0755); err != nil {
			return err
		}
	}
	return nil
}

func addr(port int) string {
	return fmt.Sprintf(":%d", port)
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}
