package main

import (
	"testing"
)

func TestSkipMigrations(t *testing.T) {
	migrations := []string{
		"20250427225203_create_users_table.up.sql",
		"20250427225403_create_groups_table.up.sql",
		"20250427225603_create_events_table.up.sql",
	}

	lastFile := "20250427225203_create_users_table"

	result := skipMigrations(lastFile, migrations, ".up.sql")
	if len(result) != 2 {
		t.Errorf("Expected 2 migrations to skip, got %d", len(result))
	}

	if result[0] != "20250427225403_create_groups_table.up.sql" {
		t.Errorf("Expected first skipped migration to be '20250427225403_create_groups_table.up.sql', got '%s'", result[0])
	}
}
