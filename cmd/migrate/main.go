package main

import (
	"flag"
	"log"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

// This program handles applying and rolling back database migrations.
// It is designed to work with any Postgres instance, including Neon DB.
func main() {
	var dbURL string
	var up bool
	var down bool

	flag.StringVar(&dbURL, "db", "", "Database URL connection string (e.g., Neon DB URL)")
	flag.BoolVar(&up, "up", false, "Apply all up migrations")
	flag.BoolVar(&down, "down", false, "Apply all down migrations")
	flag.Parse()

	if dbURL == "" {
		log.Fatal("Please provide a database URL using -db flag")
	}

	if !up && !down {
		log.Fatal("Please specify either -up or -down")
	}

	// Initialize the migrate instance pointing to our local folder
	m, err := migrate.New(
		"file://deploy/migrations",
		dbURL,
	)
	if err != nil {
		log.Fatalf("Failed to initialize migrate instance: %v", err)
	}

	if up {
		log.Println("Applying migrations...")
		if err := m.Up(); err != nil && err != migrate.ErrNoChange {
			log.Fatalf("Migration up failed: %v", err)
		}
		log.Println("Migrations up applied successfully!")
	}

	if down {
		log.Println("Rolling back migrations...")
		if err := m.Down(); err != nil && err != migrate.ErrNoChange {
			log.Fatalf("Migration down failed: %v", err)
		}
		log.Println("Migrations down applied successfully!")
	}
}
