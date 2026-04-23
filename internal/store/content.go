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

func (s *Store) DoaPage(ctx context.Context, pageNum, pageSize int) (model.DoaPageData, error) {
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

	catRows, err := s.db.QueryContext(ctx, `
		SELECT slug, title, description
		FROM content_categories
		WHERE collection_id = ?
		ORDER BY display_order, id
	`, collectionID)
	if err != nil {
		return page, fmt.Errorf("read doa categories: %w", err)
	}
	for catRows.Next() {
		var cat model.DoaCategory
		if err := catRows.Scan(&cat.ID, &cat.Title, &cat.Description); err != nil {
			_ = catRows.Close()
			return page, fmt.Errorf("scan doa category: %w", err)
		}
		page.Categories = append(page.Categories, cat)
	}
	if err := catRows.Err(); err != nil {
		return page, fmt.Errorf("iterate doa categories: %w", err)
	}
	_ = catRows.Close()

	var quranCount, hadithCount, ruqyahCount int

	collectionIDs := []string{collectionID, "ayat-doa-ruqyah"}
	placeholders := make([]byte, 0, len(collectionIDs)*2)
	argsBase := make([]any, 0, len(collectionIDs))
	for i, id := range collectionIDs {
		if i > 0 {
			placeholders = append(placeholders, ',')
		}
		placeholders = append(placeholders, '?')
		argsBase = append(argsBase, id)
	}

	countQueryBase := fmt.Sprintf(`SELECT COUNT(*) FROM content_items WHERE collection_id IN (%s) AND kind = ?`, placeholders)

	quranArgs := append(argsBase[:len(collectionIDs)], "quran")
	if err := s.db.QueryRowContext(ctx, countQueryBase, quranArgs...).Scan(&quranCount); err != nil && err != sql.ErrNoRows {
		return page, fmt.Errorf("count quran items: %w", err)
	}

	hadithArgs := append(argsBase[:len(collectionIDs)], "hadith")
	if err := s.db.QueryRowContext(ctx, countQueryBase, hadithArgs...).Scan(&hadithCount); err != nil && err != sql.ErrNoRows {
		return page, fmt.Errorf("count hadith items: %w", err)
	}

	if err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM content_items WHERE collection_id = ?
	`, "ayat-doa-ruqyah").Scan(&ruqyahCount); err != nil && err != sql.ErrNoRows {
		return page, fmt.Errorf("count ruqyah items: %w", err)
	}

	page.SourceTypes = []model.DoaSourceType{
		{ID: "quran", Title: "Al-Qur'an", Count: quranCount},
		{ID: "hadith", Title: "Hadits", Count: hadithCount},
		{ID: "ruqyah", Title: "Ruqyah", Count: ruqyahCount},
	}

	args := append(argsBase, pageSize)
	offset := (pageNum - 1) * pageSize
	args = append(args, offset)

	var total int
	totalQuery := fmt.Sprintf(`
		SELECT COUNT(*) FROM content_items
		WHERE collection_id IN (%s)
	`, placeholders)
	if err := s.db.QueryRowContext(ctx, totalQuery, argsBase...).Scan(&total); err != nil {
		return page, fmt.Errorf("count doa items: %w", err)
	}

	itemQuery := fmt.Sprintf(`
		SELECT item.slug, item.title, item.arabic, item.latin, item.translation,
			item.source, item.source_url, item.verification, item.kind, item.collection_id,
			COALESCE(cat.slug, '')
		FROM content_items item
		LEFT JOIN content_item_categories link ON link.item_id = item.id
		LEFT JOIN content_categories cat ON cat.id = link.category_id
		WHERE item.collection_id IN (%s)
		ORDER BY item.kind, link.display_order, item.id
		LIMIT ? OFFSET ?
	`, placeholders)
	itemRows, err := s.db.QueryContext(ctx, itemQuery, args...)
	if err != nil {
		return page, fmt.Errorf("read doa items: %w", err)
	}
	defer itemRows.Close()

	for itemRows.Next() {
		var item model.DoaEntry
		if err := itemRows.Scan(
			&item.ID, &item.Title, &item.Arabic, &item.Latin, &item.Translation,
			&item.Source, &item.SourceURL, &item.Verification, &item.SourceType,
			&item.CollectionID, &item.Category,
		); err != nil {
			return page, fmt.Errorf("scan doa item: %w", err)
		}
		if item.CollectionID == "ayat-doa-ruqyah" {
			item.Category = "ruqyah"
			item.IsRuqyah = true
		}
		page.Items = append(page.Items, item)
	}
	if err := itemRows.Err(); err != nil {
		return page, fmt.Errorf("iterate doa items: %w", err)
	}

	page.HasMore = offset+pageSize < total
	page.NextPage = pageNum + 1

	return page, nil
}
