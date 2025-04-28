package rollback_to

import (
	"context"
	"fmt"
	"os"

	"github.com/daqing/gomigrate/lib"
	"github.com/jackc/pgx/v5"
)

func Latest(dir string) {
	alreadyMigrated := lib.CurrentMigrated().ToArray()

	// rollback the last migration
	if len(alreadyMigrated) == 0 {
		fmt.Printf("Already at the latest migration\n")
		return
	}

	last := alreadyMigrated[len(alreadyMigrated)-1]

	ctx := context.Background()
	conn := lib.Connect(ctx)
	rollbackVersion(ctx, conn, dir, last)
}

func rollbackVersion(ctx context.Context, conn *pgx.Conn, dir string, version string) {
	tx, err := conn.Begin(ctx)
	if err != nil {
		fmt.Printf("Unable to begin transaction: %v\n", err)
		return
	}

	defer tx.Rollback(ctx)

	files, err := lib.DirEntries(dir, ".down.sql")
	if err != nil {
		fmt.Printf("Unable to read directory %s: %v\n", dir, err)
		return
	}

	for _, fileName := range files {
		ts, err := lib.ExtractTimestampPrefix(fileName)
		if err != nil {
			fmt.Printf("Unable to extract timestamp from file %s: %v\n", fileName, err)
			continue
		}

		if ts != version {
			continue
		}

		migrateDown(ctx, tx, dir, fileName)
		break
	}
	err = tx.Commit(ctx)
	if err != nil {
		fmt.Printf("Unable to commit transaction: %v\n", err)
		return
	}
	// remove the migration version from the database
	lib.RemoveMigrationVersion(ctx, conn, version)

	fmt.Printf("Migration %s rolled back successfully\n", version)
}

func migrateDown(ctx context.Context, tx pgx.Tx, dir string, fileName string) {
	filePath := fmt.Sprintf("%s/%s", dir, fileName)
	sql, err := os.ReadFile(filePath)
	if err != nil {
		fmt.Printf("Unable to read file %s: %v\n", filePath, err)
		return
	}

	_, err = tx.Exec(ctx, string(sql))
	if err != nil {
		fmt.Printf("Unable to execute file %s: %v\n", filePath, err)
		return
	}
}
