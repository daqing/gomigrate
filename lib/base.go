package lib

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5"
)

const (
	MIGRATE_TABLE_NAME = "_gomigrate"
)

func DirEntries(dir string, extension string) ([]string, error) {
	files, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("unable to read directory %s: %v", dir, err)
	}

	var upFiles []string
	for _, file := range files {
		extLen := len(extension)
		if !file.IsDir() && len(file.Name()) > extLen && file.Name()[len(file.Name())-extLen:] == extension {
			upFiles = append(upFiles, file.Name())
		}
	}

	// sort files ascending
	sort.Slice(upFiles, func(i, j int) bool {
		return upFiles[i] < upFiles[j]
	})

	return upFiles, nil
}

// ExtractTimestampPrefix extracts the timestamp prefix from a filename
// Example: "20250427214832_create_users.down.sql" → "20250427214832"
func ExtractTimestampPrefix(filename string) (string, error) {
	// Split on first underscore
	parts := strings.SplitN(filename, "_", 2)
	if len(parts) < 2 {
		return "", fmt.Errorf("filename doesn't contain an underscore separator")
	}

	timestamp := parts[0]

	// Validate it's a proper timestamp (14 digits for YYYYMMDDHHMMSS)
	if len(timestamp) != 14 {
		return "", fmt.Errorf("timestamp should be 14 digits, got %d", len(timestamp))
	}

	// Verify all characters are digits
	for _, c := range timestamp {
		if c < '0' || c > '9' {
			return "", fmt.Errorf("timestamp contains non-digit characters")
		}
	}

	return timestamp, nil
}

func CheckOrCreateTable() {
	ctx := context.Background()
	conn := Connect(ctx)

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
		version VARCHAR(255) NOT NULL
	);
	`, MIGRATE_TABLE_NAME)

	_, err := conn.Exec(context.Background(), sql)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to create table: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Table %s created successfully\n", MIGRATE_TABLE_NAME)
}

func SaveMigrationVersions(ctx context.Context, conn *pgx.Conn, versions []string) {
	tx, err := conn.Begin(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to begin transaction: %v\n", err)
		os.Exit(1)
	}

	defer tx.Rollback(ctx)
	// update the migration table
	for _, version := range versions {
		commandTag, err := tx.Exec(ctx, "INSERT INTO "+MIGRATE_TABLE_NAME+" (version) VALUES ($1)", version)
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
}

func RemoveMigrationVersion(ctx context.Context, conn *pgx.Conn, version string) {
	commandTag, err := conn.Exec(ctx, "DELETE FROM "+MIGRATE_TABLE_NAME+" WHERE version = $1", version)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to delete from table: %v\n", err)
		return
	}

	if commandTag.RowsAffected() == 0 {
		fmt.Fprintf(os.Stderr, "No rows affected: %v\n", err)
		return
	}
}
