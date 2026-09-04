package handler

import (
	"html/template"
	"strings"
)

// Arabic ornate parentheses (Mirrored=No): symmetric by construction in
// every renderer, the standard mushaf choice. Slightly more compact than
// ASCII parens; .paren-orn in input.css compensates the size.
const (
	ornateOpen  = "﴾"
	ornateClose = "﴿"
)

// ornateParens replaces every balanced, non-nested (...) group with an
// ornate pair. Logical order stays open-first (﴾...﴿) so copy-paste, screen
// readers, and search see clean data; each glyph is visually mirrored via
// .paren-flip CSS so the pair hugs RTL text (open on the right opens
// leftward toward the text and vice versa). Tags pass through verbatim;
// unbalanced and nested parens are left untouched.
func ornateParens(s string) string {
	var b strings.Builder
	b.Grow(len(s) + 96)
	i := 0
	for i < len(s) {
		c := s[i]
		if c == '<' {
			j := strings.IndexByte(s[i:], '>')
			if j < 0 {
				b.WriteString(s[i:])
				break
			}
			b.WriteString(s[i : i+j+1])
			i += j + 1
			continue
		}
		if c == '(' {
			if end, ok := parenGroupEnd(s, i); ok {
				b.WriteString(`<span class="paren-orn"><span class="paren-flip">` + ornateOpen + `</span>`)
				b.WriteString(s[i+1 : end])
				b.WriteString(`<span class="paren-flip">` + ornateClose + `</span></span>`)
				i = end + 1
				continue
			}
			b.WriteByte(c)
			i++
			continue
		}
		b.WriteByte(c)
		i++
	}
	return b.String()
}

// ArabicHTML escapes plain Arabic text and converts parenthesized runs to
// ornate pairs, which stay symmetric in RTL context by construction.
// Output carries no bidi control characters, so copy-paste stays clean.
func ArabicHTML(s string) template.HTML {
	return template.HTML(ornateParens(template.HTMLEscapeString(s)))
}

// TafsirHTML converts parenthesized runs in already-sanitized tafsir HTML
// (only <mark class="tafsir-key"> tags may be present, enforced by tests) to
// ornate pairs. Output carries no bidi control characters, so copy-paste
// stays clean.
func TafsirHTML(h template.HTML) template.HTML {
	return template.HTML(ornateParens(string(h)))
}

// parenGroupEnd returns the index of the ')' closing the group opened at
// openIdx. Only balanced, non-nested groups outside markup qualify.
func parenGroupEnd(s string, openIdx int) (int, bool) {
	inTag := false
	for j := openIdx + 1; j < len(s); j++ {
		switch s[j] {
		case '<':
			inTag = true
		case '>':
			inTag = false
		case '(':
			if !inTag {
				return 0, false
			}
		case ')':
			if !inTag {
				return j, true
			}
		}
	}
	return 0, false
}
