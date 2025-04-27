package main

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5"
)

const (
	MIGRATE_TABLE_NAME = "_gomigrate"
)

func main() {
	// reading arguments
	if len(os.Args) < 2 {
		fmt.Printf("%s [command]\n", os.Args[0])
		fmt.Printf("Available commands:\n")
		fmt.Printf("  migrate\n")
		fmt.Printf("  rollback\n")
		fmt.Printf("  status\n")

		os.Exit(1)
	}

	checkOrCreateTable()

	command := os.Args[1]
	switch command {
	case "migrate":
		migrate()
	case "rollaback":
		rollback()
	case "status":
		status()
	default:
		fmt.Println("Unknown command:", command)
	}
}

func migrate() {
	fmt.Printf("Running migration...\n")
}

func rollback() {
	fmt.Printf("Running rollback...\n")
}

func status() {
	conn := connect()
	defer conn.Close(context.Background())
	rows, err := conn.Query(context.Background(), "SELECT * FROM "+MIGRATE_TABLE_NAME)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to query table: %v\n", err)
		os.Exit(1)
	}
	defer rows.Close()

	fmt.Printf("Migration status:\n")

	for rows.Next() {
		var id int
		var name, status string
		var createdAt, updatedAt string
		err := rows.Scan(&id, &name, &status, &createdAt, &updatedAt)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Unable to scan row: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("ID: %d, Name: %s, Status: %s, Created At: %s, Updated At: %s\n", id, name, status, createdAt, updatedAt)
	}
}

func connect() *pgx.Conn {
	// urlExample := "postgres://username:password@localhost:5432/database_name"
	conn, err := pgx.Connect(context.Background(), os.Getenv("DATABASE_URL"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to connect to database: %v\n", err)
		os.Exit(1)
	}

	return conn
}

func checkOrCreateTable() {
	conn := connect()

	var n int64
	err := conn.QueryRow(context.Background(), "select 1 from information_schema.tables where table_name = $1", MIGRATE_TABLE_NAME).Scan(&n)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			createMigrateTable(conn)
			return
		}
	}

	if n == 0 {
		createMigrateTable(conn)
	}
}

func createMigrateTable(conn *pgx.Conn) {
	sql := fmt.Sprintf(`
CREATE TABLE %s (
		id SERIAL PRIMARY KEY,
		name VARCHAR(255) NOT NULL,
		status VARCHAR(255) NOT NULL,
		created_at TIMESTAMP DEFAULT NOW(),
		updated_at TIMESTAMP DEFAULT NOW()
	);
	`, MIGRATE_TABLE_NAME)

	_, err := conn.Exec(context.Background(), sql)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to create table: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Table %s created successfully\n", MIGRATE_TABLE_NAME)
}
