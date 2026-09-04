package handler

import (
	"html/template"
	"strings"
)

// leftToRightMark is used by isolateParens (tafsir pipeline): it pads an
// LTR-isolated parenthesis group so the bidi bracket-pair rule resolves the
// pair in LTR order.
const leftToRightMark = "\u200E"

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

// isolateParens wraps every balanced, non-nested (...) group in an LTR span
// with LRM padding, keeping any inner markup verbatim and leaving tags,
// unbalanced parens, and nested parens untouched. Byte-wise scanning is safe:
// '(', ')' , '<', '>' are single-byte in UTF-8 and multibyte text passes
// through unmodified.
func isolateParens(s string) string {
	var b strings.Builder
	b.Grow(len(s) + 64)
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
				b.WriteString(`<span dir="ltr">(` + leftToRightMark + `<bdi>`)
				b.WriteString(s[i+1 : end])
				b.WriteString(`</bdi>` + leftToRightMark + `)</span>`)
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

// ArabicHTML escapes plain Arabic text and converts parenthesized runs to
// ornate ﴾...﴿ pairs, which stay symmetric in RTL context by construction
// (ASCII parens cannot be made reliable: raw text mirrors, and LTR-span
// isolation renders Latin order, backwards for RTL readers).
func ArabicHTML(s string) template.HTML {
	return template.HTML(ornateParens(template.HTMLEscapeString(s)))
}

// TafsirHTML isolates parenthesized runs in already-sanitized tafsir HTML
// (only <mark class="tafsir-key"> tags may be present, enforced by tests).
func TafsirHTML(h template.HTML) template.HTML {
	return template.HTML(isolateParens(string(h)))
}
