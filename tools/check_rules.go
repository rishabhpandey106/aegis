package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/lib/pq"
)

func check() {
	dbURL := "postgresql://aegis_user:aegis_password@localhost:5432/aegis"
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	rows, err := db.Query("SELECT id, project_id, rule_type, configuration FROM security_rules")
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	fmt.Println("--- Existing Security Rules in Database ---")
	for rows.Next() {
		var id, pid, rtype, conf string
		rows.Scan(&id, &pid, &rtype, &conf)
		fmt.Printf("ID: %s\nProject: %s\nType: %s\nConfig: %s\n\n", id, pid, rtype, conf)
	}
	fmt.Println("--- End ---")
}
