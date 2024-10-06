package main

import (
	"fmt"
	"log"
	"strings"

	"github.com/abhinavxd/artemis/internal/user"
	"github.com/jmoiron/sqlx"
	"github.com/knadh/stuffbin"
	"github.com/lib/pq"
)

// install checks if the schema is already installed, prompts for confirmation, and installs the schema if needed.
func install(db *sqlx.DB, fs stuffbin.FileSystem) error {
	installed, err := checkSchema(db)
	if err != nil {
		log.Fatalf("error checking db schema: %v", err)
	}
	if installed {
		fmt.Printf("\033[31m** WARNING: This will wipe your entire DB - '%s' **\033[0m\n", ko.String("db.database"))
		fmt.Print("Continue (y/n)? ")
		var ok string
		fmt.Scanf("%s", &ok)
		if !strings.EqualFold(ok, "y") {
			log.Fatalf("installation cancelled")
		}
	}

	// Install schema.
	if err := installSchema(db, fs); err != nil {
		log.Fatalf("error installing schema: %v", err)
	}

	log.Println("Schema installed successfully")

	// Create system user.
	if err := user.CreateSystemUser(db); err != nil {
		log.Fatalf("error creating system user: %v", err)
	}
	return nil
}

// setSystemUserPass prompts for pass and sets system user password.
func setSystemUserPass(db *sqlx.DB) {
	user.ChangeSystemUserPassword(db)
}

// checkSchema verifies if the DB schema is already installed by querying a table.
func checkSchema(db *sqlx.DB) (bool, error) {
	if _, err := db.Exec(`SELECT * FROM settings LIMIT 1`); err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "42P01" {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// installSchema reads the schema file and installs it in the database.
func installSchema(db *sqlx.DB, fs stuffbin.FileSystem) error {
	q, err := fs.Read("/schema.sql")
	if err != nil {
		return err
	}
	_, err = db.Exec(string(q))
	return err
}
