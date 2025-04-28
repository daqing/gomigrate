package status

import (
	"fmt"
	"os"
	"strings"

	"github.com/daqing/gomigrate/lib"
)

func Show(args []string) {
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

	var names = make(map[string]bool)

	for _, fileName := range files {
		ts, name, err := split(fileName)
		if _, ok := names[name]; ok {
			continue
		}

		names[name] = true

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

		fmt.Printf("%s\t%s\t%s\n", ts, name, status)
	}
}

// filename: 20230101010101_migration_name.up.sql
func split(filename string) (string, string, error) {
	var name string
	name = strings.TrimSuffix(filename, ".up.sql")
	name = strings.TrimSuffix(name, ".down.sql")

	parts := strings.SplitN(name, "_", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid filename format: %s", filename)
	}

	return parts[0], parts[1], nil
}
