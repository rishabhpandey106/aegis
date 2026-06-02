package main

import (
	"database/sql"
	"fmt"
	"os"

	_ "github.com/lib/pq"
)

func migman() {
	dbURL := "postgres://aegis_user:aegis_password@localhost:5432/aegis?sslmode=disable"
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	_, err = db.Exec(`ALTER TABLE users ADD COLUMN IF NOT EXISTS clerk_id VARCHAR(255) UNIQUE;`)
	if err != nil {
		fmt.Println("Error adding clerk_id:", err)
		os.Exit(1)
	}
	fmt.Println("Successfully added clerk_id column to users table.")
}
