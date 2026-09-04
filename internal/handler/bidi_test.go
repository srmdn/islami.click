package handler

import (
	"strings"
	"testing"
)

func TestIsolateParensBasic(t *testing.T) {
	out := isolateParens("أَصْبَحْتُ (أَمْسَيتُ) مِنْكَ")
	if !strings.Contains(out, `<span dir="ltr">(`+"\u200E"+`<bdi>أَمْسَيتُ</bdi>`+"\u200E"+`)</span>`) {
		t.Fatalf("unexpected isolation: %s", out)
	}
	if strings.Count(out, "(") != 1 || strings.Count(out, ")") != 1 {
		t.Fatalf("paren count changed: %s", out)
	}
}

func TestIsolateParensMultipleGroups(t *testing.T) {
	out := isolateParens("(أ) dan (ب)")
	if strings.Count(out, `<span dir="ltr">`) != 2 {
		t.Fatalf("expected 2 groups: %s", out)
	}
}

func TestIsolateParensPreservesTags(t *testing.T) {
	in := `أبتدئ <mark class="tafsir-key">(اللهِ)</mark> علم`
	out := isolateParens(in)
	// Outer markup stays outside the isolation span; inner text is isolated.
	if !strings.Contains(out, `<mark class="tafsir-key"><span dir="ltr">(`) {
		t.Fatalf("group not isolated inside mark: %s", out)
	}
	if !strings.Contains(out, "<bdi>اللهِ</bdi>") {
		t.Fatalf("inner text not in bdi: %s", out)
	}
	if strings.Contains(out, "&lt;") {
		t.Fatalf("tag escaped unexpectedly: %s", out)
	}
}

func TestIsolateParensLeavesUnbalancedAlone(t *testing.T) {
	for _, in := range []string{"tanpa kurung", "(buka saja", "tutup saja)"} {
		out := isolateParens(in)
		if strings.Contains(out, "<span") {
			t.Fatalf("should not wrap %q: %s", in, out)
		}
	}
}

func TestIsolateParensNestedWrapsInnermostOnly(t *testing.T) {
	// No vendored content nests parens (enforced by content tests), but the
	// isolator degrades gracefully: outer group stays raw, inner is isolated.
	out := isolateParens("(nest (ed))")
	if !strings.HasPrefix(out, "(nest ") {
		t.Fatalf("outer group should stay raw: %s", out)
	}
	if !strings.Contains(out, "<bdi>ed</bdi>") {
		t.Fatalf("inner group should be isolated: %s", out)
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
	// Logical order stays open-first for clean copy-paste; visual hug comes
	// from .paren-flip CSS, not character order.
	want := `<span class="paren-orn"><span class="paren-flip">﴾</span>ي<span class="paren-flip">﴿</span></span>`
	if !strings.Contains(s, want) {
		t.Fatalf("expected flipped ornate pair, got: %s", s)
	}
	if strings.Contains(s, "(ي)") {
		t.Fatalf("ascii parens remain: %s", s)
	}
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
	if !strings.Contains(out, `<span class="paren-flip">﴾</span>ed<span class="paren-flip">﴿</span>`) {
		t.Fatalf("inner group should convert: %s", out)
	}
}
