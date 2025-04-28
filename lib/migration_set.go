package lib

import (
	"context"
	"fmt"
	"os"
	"slices"
)

type MigrationSet map[string]bool

func (ms MigrationSet) ToArray() []string {
	var result []string

	for version := range ms {
		result = append(result, version)
	}

	slices.Sort(result)

	return result
}

func CurrentMigrated(dsn string) MigrationSet {
	ctx := context.Background()
	conn := Connect(ctx, dsn)
	defer conn.Close(ctx)
	rows, err := conn.Query(ctx, "SELECT * FROM "+MIGRATE_TABLE_NAME)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to query table: %v\n", err)
		os.Exit(1)
	}
	defer rows.Close()

	var ms MigrationSet = make(MigrationSet)
	for rows.Next() {
		var id int
		var version string
		err := rows.Scan(&id, &version)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Unable to scan row: %v\n", err)
			os.Exit(1)
		}

		ms[version] = true
	}

	return ms
}
