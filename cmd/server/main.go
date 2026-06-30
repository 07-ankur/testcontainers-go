package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"testcontainers/internal/api"
	"testcontainers/internal/store"
)

func main(){
	// 1. Read configuration from the environment.
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("DATABASE_URL is required")
	}
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// 2. Startup work (connect + migrate) with a deadline.
	startupCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	db, err := store.OpenDB(startupCtx, dsn)
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	if err := store.CreateTable(startupCtx, db); err != nil {
		log.Fatalf("failed to create table: %v", err)
	}

	//3. Configure the server explicitly with a shutdown timeout.
	srv := &http.Server{
		Addr:    ":" + port,
		Handler: api.NewRouter(db),
		ReadTimeout: 5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout: 60 * time.Second,
	}

	// 4. Start the server in a goroutine.
	go func() {
		log.Printf("Server is listening on port %s", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("failed to listen and serve: %v", err)
		}
	}()

	// 5. Block until a signal is received.
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop
	log.Println("Shutting down server...")

	// 6. Shutdown the server gracefully with a timeout.
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("failed to shutdown server: %v", err)
	}
	log.Println("Server gracefully stopped")
}