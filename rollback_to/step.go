package rollback_to

import (
	"context"
	"fmt"

	"github.com/daqing/gomigrate/lib"
)

func Step(dir string, step int) {
	alreadyMigrated := lib.CurrentMigrated().ToArray()

	if len(alreadyMigrated) == 0 {
		fmt.Printf("Already at the latest migration\n")
		return
	}

	if step > len(alreadyMigrated) {
		fmt.Printf("Step %d is greater than the number of migrations (%d)\n", step, len(alreadyMigrated))
		return
	}

	ctx := context.Background()
	conn := lib.Connect(ctx)
	defer conn.Close(ctx)

	tx, err := conn.Begin(ctx)
	if err != nil {
		fmt.Printf("Unable to begin transaction: %v\n", err)
		return
	}

	files, err := lib.DirEntries(dir, ".down.sql")
	if err != nil {
		fmt.Printf("Unable to read directory %s: %v\n", dir, err)
		return
	}

	lastIdx := len(alreadyMigrated) - 1
	for i := 0; i < step; i++ {
		version := alreadyMigrated[lastIdx-i]
		err = rollbackVersion(ctx, tx, files, dir, version)
		if err != nil {
			fmt.Printf("Unable to rollback version %s: %v\n", version, err)
			return
		}
	}

	err = tx.Commit(ctx)
	if err != nil {
		fmt.Printf("Unable to commit transaction: %v\n", err)
		return
	}
	fmt.Printf("Rolled back %d migrations\n", step)
}
