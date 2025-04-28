package generator

import (
	"fmt"
	"os"
	"time"
)

func Generate(name, dir string) {
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
