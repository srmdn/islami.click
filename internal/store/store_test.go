package store_test

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"

	islamiclick "github.com/srmdn/islami.click"
	"github.com/srmdn/islami.click/internal/model"
	"github.com/srmdn/islami.click/internal/store"
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

	fromDB, err := contentStore.DoaPage(ctx)
	if err != nil {
		t.Fatalf("read doa from db: %v", err)
	}

	if fromDB.Title != fromJSON.Title {
		t.Fatalf("doa title mismatch: got %q want %q", fromDB.Title, fromJSON.Title)
	}
	if len(fromDB.Categories) != len(fromJSON.Categories) {
		t.Fatalf("doa category count mismatch: got %d want %d", len(fromDB.Categories), len(fromJSON.Categories))
	}
	for i, category := range fromDB.Categories {
		if len(category.Items) != len(fromJSON.Categories[i].Items) {
			t.Fatalf("category %s item count mismatch: got %d want %d", category.ID, len(category.Items), len(fromJSON.Categories[i].Items))
		}
	}
}
