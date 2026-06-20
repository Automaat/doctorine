package db

import (
	"io/fs"
	"regexp"
	"strconv"
	"testing"
)

var migrationFilePattern = regexp.MustCompile(`\A(\d+)_.+\.sql\z`)

// TestMigrationsAreSequential mirrors tern's loader rule: migration files must be
// numbered 1..N with no gaps or duplicates. It guards against two branches each
// adding the same next number (e.g. two 003_*.sql), which only surfaces when the
// branches merge and the migrator refuses to start.
func TestMigrationsAreSequential(t *testing.T) {
	sub, err := fs.Sub(migrationFiles, "migrations")
	if err != nil {
		t.Fatal(err)
	}
	entries, err := fs.ReadDir(sub, ".")
	if err != nil {
		t.Fatal(err)
	}

	seen := map[int]string{}
	highest := 0
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		match := migrationFilePattern.FindStringSubmatch(entry.Name())
		if match == nil {
			t.Fatalf("migration %q does not match NNN_name.sql", entry.Name())
		}
		number, err := strconv.Atoi(match[1])
		if err != nil {
			t.Fatal(err)
		}
		if prev, dup := seen[number]; dup {
			t.Fatalf("duplicate migration number %d: %q and %q", number, prev, entry.Name())
		}
		seen[number] = entry.Name()
		if number > highest {
			highest = number
		}
	}

	for i := 1; i <= highest; i++ {
		if _, ok := seen[i]; !ok {
			t.Fatalf("missing migration number %d", i)
		}
	}
}
