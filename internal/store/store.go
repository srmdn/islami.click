package store

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"slices"
	"strings"

	"github.com/srmdn/islami.click/internal/model"
	_ "modernc.org/sqlite"
)

type Store struct {
	db *sql.DB
}

func Open(ctx context.Context, path string, migrationFS embed.FS, contentFS embed.FS) (*Store, error) {
	if strings.TrimSpace(path) == "" {
		path = "file:islami-click?mode=memory&cache=shared"
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	db.SetMaxOpenConns(1)

	store := &Store{db: db}
	if _, err := db.ExecContext(ctx, "PRAGMA foreign_keys = ON"); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("enable foreign keys: %w", err)
	}
	if err := store.Migrate(ctx, migrationFS); err != nil {
		_ = db.Close()
		return nil, err
	}
	if err := store.SeedContent(ctx, contentFS); err != nil {
		_ = db.Close()
		return nil, err
	}

	return store, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) Migrate(ctx context.Context, migrationFS embed.FS) error {
	if _, err := s.db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version TEXT PRIMARY KEY,
			applied_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`); err != nil {
		return fmt.Errorf("prepare schema migrations: %w", err)
	}

	entries, err := fs.ReadDir(migrationFS, "migrations")
	if err != nil {
		return fmt.Errorf("read migrations: %w", err)
	}
	slices.SortFunc(entries, func(a, b fs.DirEntry) int {
		return strings.Compare(a.Name(), b.Name())
	})

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}

		var exists bool
		err := s.db.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version = ?)", entry.Name()).Scan(&exists)
		if err != nil {
			return fmt.Errorf("check migration %s: %w", entry.Name(), err)
		}
		if exists {
			continue
		}

		sqlText, err := migrationFS.ReadFile("migrations/" + entry.Name())
		if err != nil {
			return fmt.Errorf("read migration %s: %w", entry.Name(), err)
		}

		tx, err := s.db.BeginTx(ctx, nil)
		if err != nil {
			return fmt.Errorf("begin migration %s: %w", entry.Name(), err)
		}
		if _, err := tx.ExecContext(ctx, string(sqlText)); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("apply migration %s: %w", entry.Name(), err)
		}
		if _, err := tx.ExecContext(ctx, "INSERT INTO schema_migrations (version) VALUES (?)", entry.Name()); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("record migration %s: %w", entry.Name(), err)
		}
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit migration %s: %w", entry.Name(), err)
		}
	}

	return nil
}

func (s *Store) SeedContent(ctx context.Context, contentFS embed.FS) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin content seed: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, "DELETE FROM content_collections"); err != nil {
		return fmt.Errorf("clear content seed: %w", err)
	}

	if err := seedAlMatsurat(ctx, tx, contentFS, "almatsurat-sugro", "content/almatsurat-sugro.json", 10); err != nil {
		return err
	}
	if err := seedAlMatsurat(ctx, tx, contentFS, "almatsurat-kubro", "content/almatsurat-kubro.json", 20); err != nil {
		return err
	}
	if err := seedDoa(ctx, tx, contentFS, "doa-harian", "content/doa-harian.json", 30); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit content seed: %w", err)
	}
	return nil
}

func seedAlMatsurat(ctx context.Context, tx *sql.Tx, contentFS embed.FS, collectionID, path string, order int) error {
	data, err := contentFS.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}

	var content model.AlMatsurat
	if err := json.Unmarshal(data, &content); err != nil {
		return fmt.Errorf("parse %s: %w", path, err)
	}

	if err := insertCollection(ctx, tx, collectionID, "dhikr", content.Title, content.Description, path, checksum(data), order); err != nil {
		return err
	}

	for i, section := range content.Sections {
		_, err := tx.ExecContext(ctx, `
			INSERT INTO content_items (
				collection_id, slug, kind, title, arabic, translation, repeat_count,
				source, display_order
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, collectionID, section.ID, section.Type, section.Title, section.Arabic, section.Translation, section.Repeat, section.Source, i+1)
		if err != nil {
			return fmt.Errorf("seed %s item %s: %w", collectionID, section.ID, err)
		}
	}

	return nil
}

func seedDoa(ctx context.Context, tx *sql.Tx, contentFS embed.FS, collectionID, path string, order int) error {
	data, err := contentFS.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}

	var page model.DoaPageData
	if err := json.Unmarshal(data, &page); err != nil {
		return fmt.Errorf("parse %s: %w", path, err)
	}

	if err := insertCollection(ctx, tx, collectionID, "doa", page.Title, page.Description, path, checksum(data), order); err != nil {
		return err
	}

	for categoryIndex, category := range page.Categories {
		result, err := tx.ExecContext(ctx, `
			INSERT INTO content_categories (collection_id, slug, title, description, display_order)
			VALUES (?, ?, ?, ?, ?)
		`, collectionID, category.ID, category.Title, category.Description, categoryIndex+1)
		if err != nil {
			return fmt.Errorf("seed doa category %s: %w", category.ID, err)
		}
		categoryID, err := result.LastInsertId()
		if err != nil {
			return fmt.Errorf("read doa category id %s: %w", category.ID, err)
		}

		for itemIndex, item := range category.Items {
			result, err := tx.ExecContext(ctx, `
				INSERT INTO content_items (
					collection_id, slug, kind, title, arabic, latin, translation,
					source, source_url, verification, display_order
				) VALUES (?, ?, 'doa', ?, ?, ?, ?, ?, ?, ?, ?)
			`, collectionID, item.ID, item.Title, item.Arabic, item.Latin, item.Translation, item.Source, item.SourceURL, item.Verification, itemIndex+1)
			if err != nil {
				return fmt.Errorf("seed doa item %s: %w", item.ID, err)
			}
			itemID, err := result.LastInsertId()
			if err != nil {
				return fmt.Errorf("read doa item id %s: %w", item.ID, err)
			}
			if _, err := tx.ExecContext(ctx, `
				INSERT INTO content_item_categories (item_id, category_id, display_order)
				VALUES (?, ?, ?)
			`, itemID, categoryID, itemIndex+1); err != nil {
				return fmt.Errorf("link doa item %s: %w", item.ID, err)
			}
		}
	}

	return nil
}

func insertCollection(ctx context.Context, tx *sql.Tx, id, kind, title, description, sourcePath, sourceChecksum string, order int) error {
	_, err := tx.ExecContext(ctx, `
		INSERT INTO content_collections (
			id, kind, title, description, source_path, source_checksum, display_order
		) VALUES (?, ?, ?, ?, ?, ?, ?)
	`, id, kind, title, description, sourcePath, sourceChecksum, order)
	if err != nil {
		return fmt.Errorf("seed collection %s: %w", id, err)
	}
	return nil
}

func checksum(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
