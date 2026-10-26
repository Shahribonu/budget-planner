// Package testdb gives integration tests a migrated PostgreSQL pool.
// Tests are skipped unless TEST_DATABASE_URL is set.
package testdb

import (
	"context"
	"math/rand/v2"
	"os"
	"sync"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Shahribonu/budget-planner/backend/internal/db"
	"github.com/Shahribonu/budget-planner/backend/internal/db/dbq"
)

var migrateOnce sync.Once

func Pool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	var err error
	migrateOnce.Do(func() { err = db.Migrate(url) })
	if err != nil {
		t.Fatalf("migrate: %v", err)
	}
	pool, err := db.Open(context.Background(), url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	return pool
}

// User creates a fresh user so tests sharing a database never see each other's data.
func User(t *testing.T, pool *pgxpool.Pool) dbq.User {
	t.Helper()
	u, err := dbq.New(pool).UpsertUser(context.Background(), dbq.UpsertUserParams{
		TelegramID: rand.Int64N(1 << 50), FirstName: "Test", Language: "uz-Latn",
	})
	if err != nil {
		t.Fatal(err)
	}
	return u
}
