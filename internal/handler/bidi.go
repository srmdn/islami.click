package handler

import (
	"html/template"
	"strings"
)

// leftToRightMark forces the Unicode bidi algorithm (rule N0, bracket pairs)
// to resolve a parenthesis group in LTR order even when the group sits in
// RTL Arabic text. Without it, a parenthesized Arabic run renders mirrored
// (both parens showing as opens) because N0 assigns the pair the direction
// of its RTL content.
//
// If this ever proves insufficient in some renderer, the deterministic
// fallback is U+FD3E/U+FD3F (﴾ ﴿, ornate parentheses, Mirrored=No).
const leftToRightMark = "\u200E"

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

// ArabicHTML escapes plain Arabic text and isolates parenthesized runs so
// opening/closing parens render symmetric in RTL context.
func ArabicHTML(s string) template.HTML {
	return template.HTML(isolateParens(template.HTMLEscapeString(s)))
}

// TafsirHTML isolates parenthesized runs in already-sanitized tafsir HTML
// (only <mark class="tafsir-key"> tags may be present, enforced by tests).
func TafsirHTML(h template.HTML) template.HTML {
	return template.HTML(isolateParens(string(h)))
}
