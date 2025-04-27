package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

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
		fmt.Printf("  generate(g)\n")

		os.Exit(1)
	}

	checkOrCreateTable()

	command := os.Args[1]
	switch command {
	case "migrate":
		migrate(os.Args[2:])
	case "rollaback":
		rollback()
	case "status":
		status()
	case "generate", "g":
		generate(os.Args[2:])
	default:
		fmt.Println("Unknown command:", command)
	}
}

func generate(args []string) {
	if len(args) < 2 {
		fmt.Printf("Usage: %s generate [migration_name] [dir]\n", os.Args[0])
		return
	}

	name := args[0]
	dir := args[1]
	fmt.Printf("Generating migration file %s in %s...\n", name, dir)

	timestamp := fmt.Sprintf("%s", time.Now().Format("20060102150405"))

	upFile := fmt.Sprintf("%s/%s_%s.up.sql", dir, timestamp, name)
	downFile := fmt.Sprintf("%s/%s_%s.down.sql", dir, timestamp, name)

	// create empty files
	err := os.WriteFile(upFile, []byte("-- up migration"), 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to create file %s: %v\n", upFile, err)
		os.Exit(1)
	}

	err = os.WriteFile(downFile, []byte("-- down migration"), 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to create file %s: %v\n", downFile, err)
		os.Exit(1)
	}

	fmt.Printf("Migration files %s and %s created successfully\n", upFile, downFile)
}

func migrate(args []string) {
	if len(args) < 1 {
		fmt.Printf("Usage: %s migrate [dir]\n", os.Args[0])
		return
	}

	dir := args[0]

	fmt.Printf("Running migration files inside %s...\n", dir)
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
