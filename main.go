package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

const (
	MIGRATE_TABLE_NAME = "_gomigrate"
)

type MigrationStatus struct {
	ID     int
	Name   string
	Status string
}

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
	case "rollback":
		rollback(os.Args[2:])
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

	var lastUpFile string = ""
	ms := currentStatus()
	if len(ms) > 0 {
		last := ms[len(ms)-1]
		if last.Status == "UP" {
			lastUpFile = last.Name
		}
	}

	run_sql(args, ".up.sql", "UP", lastUpFile)
}

func rollback(args []string) {
	if len(args) < 1 {
		fmt.Printf("Usage: %s migrate [dir]\n", os.Args[0])
		return
	}

	var lastDownFile string = ""
	ms := currentStatus()
	if len(ms) > 0 {
		last := ms[len(ms)-1]
		if last.Status == "DOWN" {
			lastDownFile = last.Name
		}
	}

	run_sql(args, ".down.sql", "DOWN", lastDownFile)
}

func run_sql(args []string, extension string, action string, lastFile string) {
	if len(args) < 1 {
		fmt.Printf("Usage: %s migrate [dir]\n", os.Args[0])
		return
	}

	dir := args[0]

	fmt.Printf("Running migration files inside %s...\n", dir)

	ctx := context.Background()

	conn := connect(ctx)
	defer conn.Close(ctx)

	files, err := os.ReadDir(dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to read directory %s: %v\n", dir, err)
		os.Exit(1)
	}

	var upFiles []os.DirEntry
	for _, file := range files {
		extLen := len(extension)
		if !file.IsDir() && len(file.Name()) > extLen && file.Name()[len(file.Name())-extLen:] == extension {
			upFiles = append(upFiles, file)
		}
	}

	// sort files ascending
	sort.Slice(upFiles, func(i, j int) bool {
		return files[i].Name() < files[j].Name()
	})

	upFileNames := make([]string, len(upFiles))
	for i, file := range upFiles {
		upFileNames[i] = file.Name()
	}

	upFileNames = skipMigrations(lastFile, upFileNames, extension)

	// run each sql file inside a transaction
	tx, err := conn.Begin(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to begin transaction: %v\n", err)
		os.Exit(1)
	}
	defer tx.Rollback(ctx)

	for _, fileName := range upFileNames {
		filePath := fmt.Sprintf("%s/%s", dir, fileName)
		sql, err := os.ReadFile(filePath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Unable to read file %s: %v\n", filePath, err)
			os.Exit(1)
		}

		fmt.Printf("Running migration file %s...\n", fileName)

		_, err = tx.Exec(ctx, string(sql))
		if err != nil {
			fmt.Fprintf(os.Stderr, "Unable to execute file %s: %v\n", filePath, err)
			tx.Rollback(ctx)
			os.Exit(1)
		}
	}

	err = tx.Commit(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to commit transaction: %v\n", err)
		os.Exit(1)
	}

	tx, err = conn.Begin(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to begin transaction: %v\n", err)
		os.Exit(1)
	}

	defer tx.Rollback(ctx)
	// update the migration table
	for _, fileName := range upFileNames {
		filePart := strings.TrimSuffix(fileName, extension)
		commandTag, err := tx.Exec(ctx, "INSERT INTO "+MIGRATE_TABLE_NAME+" (name, status) VALUES ($1, $2)", filePart, action)
		fmt.Println("---> Command Tag: ", commandTag)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Unable to insert into table: %v\n", err)
			tx.Rollback(ctx)
			os.Exit(1)
		}
		if commandTag.RowsAffected() == 0 {
			fmt.Fprintf(os.Stderr, "No rows affected: %v\n", err)
			tx.Rollback(ctx)
			os.Exit(1)
		}

	}

	err = tx.Commit(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to commit transaction: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Migration files inside %s executed successfully\n", dir)
}

func skipMigrations(lastFile string, files []string, extension string) []string {
	var result []string
	for _, file := range files {
		filePart := strings.TrimSuffix(file, extension)

		if filePart <= lastFile {
			// skip this file
			continue
		}

		result = append(result, file)
	}
	return result
}

func currentStatus() []MigrationStatus {
	ctx := context.Background()
	conn := connect(ctx)
	defer conn.Close(ctx)
	rows, err := conn.Query(ctx, "SELECT * FROM "+MIGRATE_TABLE_NAME)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to query table: %v\n", err)
		os.Exit(1)
	}
	defer rows.Close()

	var ms []MigrationStatus
	for rows.Next() {
		var id int
		var name, status string
		var createdAt, updatedAt time.Time
		err := rows.Scan(&id, &name, &status, &createdAt, &updatedAt)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Unable to scan row: %v\n", err)
			os.Exit(1)
		}

		ms = append(ms, MigrationStatus{
			ID:     id,
			Name:   name,
			Status: status,
		})
	}

	return ms
}

func status() {
	ms := currentStatus()

	fmt.Printf("ID\tName\t\t\t\tStatus\n")
	for _, m := range ms {
		fmt.Printf("%d\t%s\t%s\n", m.ID, m.Name, m.Status)
	}
}

func connect(ctx context.Context) *pgx.Conn {
	// urlExample := "postgres://username:password@localhost:5432/database_name"
	conn, err := pgx.Connect(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to connect to database: %v\n", err)
		os.Exit(1)
	}

	return conn
}

func checkOrCreateTable() {
	ctx := context.Background()
	conn := connect(ctx)

	var n int64
	err := conn.QueryRow(ctx, "select 1 from information_schema.tables where table_name = $1", MIGRATE_TABLE_NAME).Scan(&n)
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
