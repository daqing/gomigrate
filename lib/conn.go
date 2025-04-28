package lib

import (
	"context"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5"
)

func Connect(ctx context.Context) *pgx.Conn {
	// urlExample := "postgres://username:password@localhost:5432/database_name"
	conn, err := pgx.Connect(ctx, os.Getenv("DATABASE_URL"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to connect to database: %v\n", err)
		os.Exit(1)
	}

	return conn
}
