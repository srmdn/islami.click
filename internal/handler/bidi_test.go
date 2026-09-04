package handler

import (
	"html/template"
	"strings"
	"testing"
)

// bidiControls must never appear in rendered Arabic: they leak into
// clipboards and permanently scramble pasted text in every target.
func assertNoBidiControls(t *testing.T, s, what string) {
	t.Helper()
	for _, c := range []string{"\u200E", "\u200F", "\u202A", "\u202B", "\u202C", "\u2066", "\u2067", "\u2069"} {
		if strings.Contains(s, c) {
			t.Fatalf("%s contains bidi control U+%04X", what, []rune(c)[0])
		}
	}
}

func TestOrnateParensBasic(t *testing.T) {
	out := ornateParens("أَصْبَحْتُ (أَمْسَيتُ) مِنْكَ")
	want := `<span class="paren-orn">﴿أَمْسَيتُ﴾</span>`
	if !strings.Contains(out, want) {
		t.Fatalf("unexpected conversion: %s", out)
	}
	assertNoBidiControls(t, out, "ornate output")
}

func TestOrnateParensMultipleGroups(t *testing.T) {
	out := ornateParens("(أ) dan (ب)")
	if strings.Count(out, `paren-orn`) != 2 {
		t.Fatalf("expected 2 groups: %s", out)
	}
}

func TestOrnateParensPreservesTags(t *testing.T) {
	in := `أبتدئ <mark class="tafsir-key">(اللهِ)</mark> علم`
	out := ornateParens(in)
	if !strings.Contains(out, `<mark class="tafsir-key"><span class="paren-orn">﴿`) {
		t.Fatalf("group not converted inside mark: %s", out)
	}
	if strings.Contains(out, "&lt;") {
		t.Fatalf("tag escaped unexpectedly: %s", out)
	}
	assertNoBidiControls(t, out, "ornate output")
}

func TestOrnateParensLeavesUnbalancedAlone(t *testing.T) {
	for _, in := range []string{"tanpa kurung", "(buka saja", "tutup saja)"} {
		if out := ornateParens(in); strings.Contains(out, "paren-orn") {
			t.Fatalf("should not convert %q: %s", in, out)
		}
	}
}

func TestOrnateParensNestedConvertsInnermostOnly(t *testing.T) {
	// No vendored content nests parens; graceful degradation only.
	out := ornateParens("(nest (ed))")
	if !strings.HasPrefix(out, "(nest ") {
		t.Fatalf("outer group should stay raw: %s", out)
	}
	if !strings.Contains(out, `﴿ed﴾`) {
		t.Fatalf("inner group should convert: %s", out)
	}
}

func TestArabicHTMLEscapesAndOrnates(t *testing.T) {
	out := ArabicHTML(`x <b>(ي)</b>`)
	s := string(out)
	if strings.Contains(s, "<b>") {
		t.Fatalf("raw tag leaked: %s", s)
	}
	if !strings.Contains(s, "&lt;b&gt;") {
		t.Fatalf("expected escaping: %s", s)
	}
	// Hug order is close-shape-first by emission: correct in every target
	// with no CSS, at the cost of logical close-first order.
	want := `<span class="paren-orn">﴿ي﴾</span>`
	if !strings.Contains(s, want) {
		t.Fatalf("expected flipped ornate pair, got: %s", s)
	}
	if strings.Contains(s, "(ي)") {
		t.Fatalf("ascii parens remain: %s", s)
	}
	assertNoBidiControls(t, s, "arabicHTML output")
}

func TestTafsirHTMLOrnatesAndStaysClean(t *testing.T) {
	out := TafsirHTML(template.HTML(`أبتدئ <mark class="tafsir-key">(اللهِ)</mark> علم`))
	s := string(out)
	if !strings.Contains(s, `<mark class="tafsir-key"><span class="paren-orn">﴿`) {
		t.Fatalf("group not converted inside mark: %s", s)
	}
	if strings.Contains(s, "(اللهِ)") {
		t.Fatalf("ascii parens remain: %s", s)
	}
	assertNoBidiControls(t, s, "tafsirHTML output")
}
