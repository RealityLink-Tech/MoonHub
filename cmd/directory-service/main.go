// cmd/directory-service/main.go
package main

import (
	"database/sql"
	"flag"
	"log"
	"net/http"
	"os"

	_ "github.com/lib/pq"

	"github.com/RealityLink-Tech/MoonHub/cloud/directory"
)

func main() {
	addr := flag.String("addr", ":8080", "listen address")
	dbHost := flag.String("db-host", "localhost", "PostgreSQL host")
	dbPort := flag.String("db-port", "5432", "PostgreSQL port")
	dbUser := flag.String("db-user", "moonhub", "PostgreSQL user")
	dbPass := flag.String("db-pass", "", "PostgreSQL password")
	dbName := flag.String("db-name", "moonhub", "PostgreSQL database")
	flag.Parse()

	connStr := "host=" + *dbHost + " port=" + *dbPort + " user=" + *dbUser +
		" password=" + *dbPass + " dbname=" + *dbName + " sslmode=disable"

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("failed to ping database: %v", err)
	}

	if _, err := db.Exec(directory.MigrationSQL); err != nil {
		log.Fatalf("failed to run migration: %v", err)
	}

	store := directory.NewPostgresStore(db)
	cache := directory.NewRedisCache(os.Getenv("REDIS_URL"))
	handler := directory.NewHandler(store, cache)

	log.Printf("directory service listening on %s", *addr)
	if err := http.ListenAndServe(*addr, handler); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
