package store_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"path/filepath"
	"testing"

	islamiclick "github.com/srmdn/islami.click"
	"github.com/srmdn/islami.click/internal/model"
	"github.com/srmdn/islami.click/internal/store"

	_ "modernc.org/sqlite"
)

func TestSeededContentMatchesJSON(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "content.db")

	contentStore, err := store.Open(ctx, dbPath, islamiclick.MigrationFS, islamiclick.ContentFS)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer contentStore.Close()

	assertAlMatsuratMatchesJSON(t, ctx, contentStore, "almatsurat-sugro", "content/almatsurat-sugro.json")
	assertAlMatsuratMatchesJSON(t, ctx, contentStore, "almatsurat-kubro", "content/almatsurat-kubro.json")
	assertDoaMatchesJSON(t, ctx, contentStore)
}

func assertAlMatsuratMatchesJSON(t *testing.T, ctx context.Context, contentStore *store.Store, collectionID, path string) {
	t.Helper()

	data, err := islamiclick.ContentFS.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}

	var fromJSON model.AlMatsurat
	if err := json.Unmarshal(data, &fromJSON); err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}

	fromDB, err := contentStore.AlMatsurat(ctx, collectionID)
	if err != nil {
		t.Fatalf("read %s from db: %v", collectionID, err)
	}

	if fromDB.Title != fromJSON.Title {
		t.Fatalf("%s title mismatch: got %q want %q", collectionID, fromDB.Title, fromJSON.Title)
	}
	if len(fromDB.Sections) != len(fromJSON.Sections) {
		t.Fatalf("%s section count mismatch: got %d want %d", collectionID, len(fromDB.Sections), len(fromJSON.Sections))
	}
	if len(fromDB.Sections) > 0 && fromDB.Sections[0].ID != fromJSON.Sections[0].ID {
		t.Fatalf("%s first section mismatch: got %q want %q", collectionID, fromDB.Sections[0].ID, fromJSON.Sections[0].ID)
	}
}

func assertDoaMatchesJSON(t *testing.T, ctx context.Context, contentStore *store.Store) {
	t.Helper()

	data, err := islamiclick.ContentFS.ReadFile("content/doa-harian.json")
	if err != nil {
		t.Fatalf("read doa JSON: %v", err)
	}

	var fromJSON model.DoaPageData
	if err := json.Unmarshal(data, &fromJSON); err != nil {
		t.Fatalf("parse doa JSON: %v", err)
	}

	totalFromJSON := 0
	for _, cat := range fromJSON.Categories {
		totalFromJSON += len(cat.Items)
	}

	ruqyahData, err := islamiclick.ContentFS.ReadFile("content/ayat-doa-ruqyah.json")
	if err != nil {
		t.Fatalf("read ruqyah JSON: %v", err)
	}
	var ruqyahJSON model.DoaPageData
	if err := json.Unmarshal(ruqyahData, &ruqyahJSON); err != nil {
		t.Fatalf("parse ruqyah JSON: %v", err)
	}
	for _, cat := range ruqyahJSON.Categories {
		totalFromJSON += len(cat.Items)
	}

	fromDB, err := contentStore.DoaPage(ctx, 1, 1000)
	if err != nil {
		t.Fatalf("read doa from db: %v", err)
	}

	if fromDB.Title != fromJSON.Title {
		t.Fatalf("doa title mismatch: got %q want %q", fromDB.Title, fromJSON.Title)
	}
	if len(fromDB.Categories) != len(fromJSON.Categories) {
		t.Fatalf("doa category count mismatch: got %d want %d", len(fromDB.Categories), len(fromJSON.Categories))
	}
	if len(fromDB.Items) != totalFromJSON {
		t.Fatalf("doa item count mismatch: got %d want %d", len(fromDB.Items), totalFromJSON)
	}
}

func TestChecksumSkipOnReopen(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "content.db")

	store1, err := store.Open(ctx, dbPath, islamiclick.MigrationFS, islamiclick.ContentFS)
	if err != nil {
		t.Fatalf("first open: %v", err)
	}
	store1.Close()

	store2, err := store.Open(ctx, dbPath, islamiclick.MigrationFS, islamiclick.ContentFS)
	if err != nil {
		t.Fatalf("second open: %v", err)
	}
	defer store2.Close()

	content, err := store2.AlMatsurat(ctx, "almatsurat-sugro")
	if err != nil {
		t.Fatalf("read almatsurat-sugro: %v", err)
	}
	if content.Title == "" {
		t.Fatal("title is empty after checksum skip")
	}
}

func TestChecksumReSeedOnChange(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "content.db")

	store1, err := store.Open(ctx, dbPath, islamiclick.MigrationFS, islamiclick.ContentFS)
	if err != nil {
		t.Fatalf("first open: %v", err)
	}
	store1.Close()

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("open db for corrupt: %v", err)
	}
	if _, err := db.Exec("UPDATE content_collections SET source_checksum = 'corrupted' WHERE id = 'doa-harian'"); err != nil {
		db.Close()
		t.Fatalf("update checksum: %v", err)
	}
	db.Close()

	store2, err := store.Open(ctx, dbPath, islamiclick.MigrationFS, islamiclick.ContentFS)
	if err != nil {
		t.Fatalf("second open: %v", err)
	}
	defer store2.Close()

	data, err := islamiclick.ContentFS.ReadFile("content/doa-harian.json")
	if err != nil {
		t.Fatalf("read doa json: %v", err)
	}
	var fromJSON model.DoaPageData
	if err := json.Unmarshal(data, &fromJSON); err != nil {
		t.Fatalf("parse doa json: %v", err)
	}

	fromDB, err := store2.DoaPage(ctx, 1, 1000)
	if err != nil {
		t.Fatalf("read doa from db: %v", err)
	}
	if fromDB.Title != fromJSON.Title {
		t.Fatalf("title mismatch after re-seed: got %q want %q", fromDB.Title, fromJSON.Title)
	}
}

func TestNoDuplicateRowsAfterReopen(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "content.db")

	store1, err := store.Open(ctx, dbPath, islamiclick.MigrationFS, islamiclick.ContentFS)
	if err != nil {
		t.Fatalf("first open: %v", err)
	}
	store1.Close()

	store2, err := store.Open(ctx, dbPath, islamiclick.MigrationFS, islamiclick.ContentFS)
	if err != nil {
		t.Fatalf("second open: %v", err)
	}
	defer store2.Close()

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("open db for check: %v", err)
	}
	defer db.Close()

	var total, distinct int
	err = db.QueryRowContext(ctx, "SELECT COUNT(*), COUNT(DISTINCT collection_id || ':' || slug) FROM content_items").Scan(&total, &distinct)
	if err != nil {
		t.Fatalf("count rows: %v", err)
	}
	if total != distinct {
		t.Fatalf("duplicate rows: total=%d distinct=%d", total, distinct)
	}
}
