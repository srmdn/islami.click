package islamiclick_test

import (
	"encoding/json"
	"net/url"
	"strings"
	"testing"

	islamiclick "github.com/srmdn/islami.click"
	"github.com/srmdn/islami.click/internal/model"
)

func TestDoaContentIsComplete(t *testing.T) {
	data, err := islamiclick.ContentFS.ReadFile("content/doa-harian.json")
	if err != nil {
		t.Fatalf("read doa content: %v", err)
	}

	var page model.DoaPageData
	if err := json.Unmarshal(data, &page); err != nil {
		t.Fatalf("parse doa content: %v", err)
	}

	if strings.TrimSpace(page.Title) == "" {
		t.Fatal("page title is required")
	}
	if len(page.Categories) == 0 {
		t.Fatal("at least one doa category is required")
	}

	for _, category := range page.Categories {
		if strings.TrimSpace(category.ID) == "" {
			t.Fatal("category id is required")
		}
		if strings.TrimSpace(category.Title) == "" {
			t.Fatalf("category %q title is required", category.ID)
		}
		if len(category.Items) == 0 {
			t.Fatalf("category %q must contain items", category.ID)
		}

		for _, item := range category.Items {
			if strings.TrimSpace(item.ID) == "" {
				t.Fatalf("category %q has item with empty id", category.ID)
			}
			checkRequired(t, item.ID, "title", item.Title)
			checkRequired(t, item.ID, "arabic", item.Arabic)
			checkRequired(t, item.ID, "latin", item.Latin)
			checkRequired(t, item.ID, "translation", item.Translation)
			checkRequired(t, item.ID, "source", item.Source)
			checkRequired(t, item.ID, "verification", item.Verification)

			sourceURL := strings.TrimSpace(item.SourceURL)
			if sourceURL == "" {
				t.Fatalf("item %q source_url is required", item.ID)
			}
			parsed, err := url.ParseRequestURI(sourceURL)
			if err != nil {
				t.Fatalf("item %q source_url is invalid: %v", item.ID, err)
			}
			if parsed.Scheme != "https" {
				t.Fatalf("item %q source_url must use https", item.ID)
			}
		}
	}
}

func checkRequired(t *testing.T, itemID, field, value string) {
	t.Helper()
	if strings.TrimSpace(value) == "" {
		t.Fatalf("item %q %s is required", itemID, field)
	}
}
