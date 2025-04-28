package migrate_up

import (
	"context"
	"fmt"
	"os"

	"github.com/daqing/gomigrate/lib"
)

// Migrate up to a specific version
func Version(dir, version, dsn string) {
	ctx := context.Background()

	files, err := lib.DirEntries(dir, ".up.sql")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to read directory %s: %v\n", dir, err)
		os.Exit(1)
	}

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

		if ts == version {
			fmt.Printf("Migration %s stopped.\n", fileName)
			break
		}
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
