package main

import (
	"context"
	"log"
	"net/http"
	"os"
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

	// 2. Open the database, giving the connection attempt a deadline.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	db, err := store.OpenDB(ctx, dsn)
	if err != nil {
		log.Fatal("connecting to db: ", err)
	}

	// 3. Create the users table if it doesn't exist.
	if err := store.CreateTable(ctx, db); err != nil {
		log.Fatal("creating users table: ", err)
	}

	// 4. Build the HTTP router and start the server.
	handler := api.NewRouter(db)

	addr := ":" + port
	log.Printf("listening on %s", addr)
	if err := http.ListenAndServe(addr, handler); err != nil {
		log.Fatal("http server: ", err)
	}
}