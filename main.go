package main

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/daqing/gomigrate/lib"
	"github.com/daqing/gomigrate/migrate_up"
	"github.com/daqing/gomigrate/rollback_to"
)

type MigrationStatus struct {
	ID      int
	Version string
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

	lib.CheckOrCreateTable()

	command := os.Args[1]
	switch command {
	case "migrate":
		migrate(os.Args[2:])
	case "rollback":
		rollback(os.Args[2:])
	case "status":
		status(os.Args[2:])
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
	var version string = ""
	if len(args) > 1 {
		version = args[1]
	}

	if version == "" {
		migrate_up.All(dir)
	} else {
		migrate_up.Version(dir, version)
	}

}

func rollback(args []string) {
	if len(args) < 1 {
		fmt.Printf("Usage: %s migrate [dir]\n", os.Args[0])
		return
	}

	dir := args[0]
	var step int = 1
	var err error
	if len(args) > 1 {
		step, err = strconv.Atoi(args[1])
		if err != nil {
			fmt.Printf("Invalid step: %v\n", err)
			os.Exit(1)
		}
	}

	if step <= 1 {
		rollback_to.Latest(dir)
	} else {
		rollback_to.Step(dir, step)
	}
}

func status(args []string) {
	if len(args) < 1 {
		fmt.Printf("Usage: %s status [dir]\n", os.Args[0])
		return
	}

	dir := args[0]
	alreadyMigrated := lib.CurrentMigrated()

	files, err := lib.DirEntries(dir, ".sql")
	if err != nil {
		fmt.Printf("Error reading directory %s: %v\n", dir, err)
		os.Exit(1)
	}

	for _, fileName := range files {
		ts, err := lib.ExtractTimestampPrefix(fileName)
		if err != nil {
			fmt.Printf("Error extracting timestamp from file %s: %v\n", fileName, err)
			continue
		}

		var status string
		if _, ok := alreadyMigrated[ts]; ok {
			status = "UP"
		} else {
			status = "DOWN"
		}

		fmt.Printf("%s %s\n", fileName, status)
	}
}
