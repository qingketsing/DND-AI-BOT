package bootstrap

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

// RunEmbeddedMigrations 执行内嵌的 SQL migration，确保运行入口具备完整表结构。
func RunEmbeddedMigrations(ctx context.Context, db *sql.DB) error {
	for _, name := range embeddedMigrationNames() {
		content, err := os.ReadFile(filepath.Join("migrations", name))
		if err != nil {
			return fmt.Errorf("read migration %s: %w", name, err)
		}
		if _, err := db.ExecContext(ctx, string(content)); err != nil {
			return fmt.Errorf("apply migration %s: %w", name, err)
		}
	}

	return nil
}

func embeddedMigrationNames() []string {
	entries, err := os.ReadDir("migrations")
	if err != nil {
		return nil
	}

	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		names = append(names, entry.Name())
	}
	sort.Strings(names)
	return names
}
