package migrate_up

import (
	"context"
	"fmt"
	"os"

	"github.com/daqing/gomigrate/lib"
	"github.com/jackc/pgx/v5"
)

func All(dir string) {
	ctx := context.Background()

	files, err := lib.DirEntries(dir, ".up.sql")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to read directory %s: %v\n", dir, err)
		os.Exit(1)
	}

	dsn := os.Getenv("DATABASE_URL")

	alreadyMigrated := lib.CurrentMigrated(dsn)

	conn := lib.Connect(ctx, dsn)
	defer conn.Close(ctx)

	// run each sql file inside a transaction
	tx, err := conn.Begin(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to begin transaction: %v\n", err)
		os.Exit(1)
	}
	defer tx.Rollback(ctx)

	var versions []string

	for _, fileName := range files {
		ts, err := lib.ExtractTimestampPrefix(fileName)
		if err != nil {
			fmt.Printf("Unable to extract timestamp from file %s: %v\n", fileName, err)
			continue
		}

		if _, ok := alreadyMigrated[ts]; ok {
			fmt.Printf("Migration %s already applied, skipping...\n", fileName)
			continue
		}

		migrateUp(ctx, tx, dir, fileName)
		versions = append(versions, ts)
	}

	err = tx.Commit(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to commit transaction: %v\n", err)
		os.Exit(1)
	}

	// save the migration versions to the database
	lib.SaveMigrationVersions(ctx, conn, versions)

	fmt.Printf("Migration files inside %s executed successfully\n", dir)
}

func migrateUp(ctx context.Context, tx pgx.Tx, dir string, fileName string) {
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
