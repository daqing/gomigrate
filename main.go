package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/daqing/gomigrate/generator"
	"github.com/daqing/gomigrate/lib"
	"github.com/daqing/gomigrate/migrate_up"
	"github.com/daqing/gomigrate/rollback_to"
	"github.com/daqing/gomigrate/status"
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

	dsn := os.Getenv("DATABASE_URL")
	lib.CheckOrCreateTable(dsn)

	command := os.Args[1]
	switch command {
	case "migrate":
		migrate(os.Args[2:])
	case "rollback":
		rollback(os.Args[2:])
	case "status":
		args := os.Args[2:]
		if len(args) < 1 {
			fmt.Printf("Usage: %s status [dir]\n", os.Args[0])
			return
		}
		status.Show(args[0])
	case "generate", "g":
		args := os.Args[2:]
		if len(args) < 2 {
			fmt.Printf("Usage: %s generate [migration_name] [dir]\n", os.Args[0])
			return
		}
		generator.Generate(args[0], args[1])
	default:
		fmt.Println("Unknown command:", command)
	}
}

func migrate(args []string) {
	if len(args) < 1 {
		fmt.Printf("Usage: %s migrate [version] [dir]\n", os.Args[0])
		return
	}

	var dir string
	var version string = ""
	if len(args) > 1 {
		version = args[0]
		dir = args[1]
	} else {
		dir = args[0]
	}

	if version == "" {
		migrate_up.All(dir)
	} else {
		migrate_up.Version(dir, version)
	}
}

func rollback(args []string) {
	if len(args) < 1 {
		fmt.Printf("Usage: %s rollback [n] [dir]\n", os.Args[0])
		return
	}

	var dir string
	var step int = 1
	var err error
	if len(args) > 1 {
		step, err = strconv.Atoi(args[0])
		if err != nil {
			fmt.Printf("Invalid step: %v\n", err)
			fmt.Printf("Usage: %s rollback [n] [dir]\n", os.Args[0])
			os.Exit(1)
		}
		dir = args[1]
	} else {
		dir = args[0]
	}

	if step <= 1 {
		rollback_to.Latest(dir)
	} else {
		rollback_to.Step(dir, step)
	}
}
