package store

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/srmdn/islami.click/internal/model"
)

func (s *Store) AlMatsurat(ctx context.Context, collectionID string) (model.AlMatsurat, error) {
	var content model.AlMatsurat
	err := s.db.QueryRowContext(ctx, `
		SELECT title, description
		FROM content_collections
		WHERE id = ? AND kind = 'dhikr'
	`, collectionID).Scan(&content.Title, &content.Description)
	if err != nil {
		if err == sql.ErrNoRows {
			return content, fmt.Errorf("almatsurat collection %q not found", collectionID)
		}
		return content, fmt.Errorf("read almatsurat collection %q: %w", collectionID, err)
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT slug, kind, title, arabic, translation, repeat_count, source
		FROM content_items
		WHERE collection_id = ?
		ORDER BY display_order, id
	`, collectionID)
	if err != nil {
		return content, fmt.Errorf("read almatsurat items %q: %w", collectionID, err)
	}
	defer rows.Close()

	for rows.Next() {
		var entry model.DhikrEntry
		if err := rows.Scan(
			&entry.ID,
			&entry.Type,
			&entry.Title,
			&entry.Arabic,
			&entry.Translation,
			&entry.Repeat,
			&entry.Source,
		); err != nil {
			return content, fmt.Errorf("scan almatsurat item %q: %w", collectionID, err)
		}
		content.Sections = append(content.Sections, entry)
	}
	if err := rows.Err(); err != nil {
		return content, fmt.Errorf("iterate almatsurat items %q: %w", collectionID, err)
	}

	return content, nil
}

func (s *Store) DoaPage(ctx context.Context) (model.DoaPageData, error) {
	const collectionID = "doa-harian"

	var page model.DoaPageData
	err := s.db.QueryRowContext(ctx, `
		SELECT title, description
		FROM content_collections
		WHERE id = ? AND kind = 'doa'
	`, collectionID).Scan(&page.Title, &page.Description)
	if err != nil {
		if err == sql.ErrNoRows {
			return page, fmt.Errorf("doa collection %q not found", collectionID)
		}
		return page, fmt.Errorf("read doa collection: %w", err)
	}

	categoryRows, err := s.db.QueryContext(ctx, `
		SELECT id, slug, title, description
		FROM content_categories
		WHERE collection_id = ?
		ORDER BY display_order, id
	`, collectionID)
	if err != nil {
		return page, fmt.Errorf("read doa categories: %w", err)
	}

	type categoryRow struct {
		dbID     int64
		category model.DoaCategory
	}
	var categories []categoryRow
	for categoryRows.Next() {
		var row categoryRow
		if err := categoryRows.Scan(&row.dbID, &row.category.ID, &row.category.Title, &row.category.Description); err != nil {
			_ = categoryRows.Close()
			return page, fmt.Errorf("scan doa category: %w", err)
		}
		categories = append(categories, row)
	}
	if err := categoryRows.Err(); err != nil {
		_ = categoryRows.Close()
		return page, fmt.Errorf("iterate doa categories: %w", err)
	}
	if err := categoryRows.Close(); err != nil {
		return page, fmt.Errorf("close doa categories: %w", err)
	}

	for _, row := range categories {
		items, err := s.doaItemsByCategory(ctx, row.dbID, row.category.ID)
		if err != nil {
			return page, err
		}
		row.category.Items = items
		page.Categories = append(page.Categories, row.category)
	}

	return page, nil
}

func (s *Store) doaItemsByCategory(ctx context.Context, categoryID int64, categorySlug string) ([]model.DoaEntry, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT item.slug, item.title, item.arabic, item.latin, item.translation,
			item.source, item.source_url, item.verification, item.kind
		FROM content_items item
		INNER JOIN content_item_categories link ON link.item_id = item.id
		WHERE link.category_id = ?
		ORDER BY link.display_order, item.id
	`, categoryID)
	if err != nil {
		return nil, fmt.Errorf("read doa items for category %s: %w", categorySlug, err)
	}
	defer rows.Close()

	var items []model.DoaEntry
	for rows.Next() {
		var item model.DoaEntry
		if err := rows.Scan(
			&item.ID,
			&item.Title,
			&item.Arabic,
			&item.Latin,
			&item.Translation,
			&item.Source,
			&item.SourceURL,
			&item.Verification,
			&item.SourceType,
		); err != nil {
			return nil, fmt.Errorf("scan doa item for category %s: %w", categorySlug, err)
		}
		item.Category = categorySlug
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate doa items for category %s: %w", categorySlug, err)
	}

	return items, nil
}
