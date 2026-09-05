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
// ornate pair emitted in hug order: CLOSE-shape char (﴿) first, OPEN-shape
// char (﴾) last. Amiri draws FD3E (-like, opens right) and FD3F )-like,
// opens left; in RTL flow the first char lands on the right opening
// leftward toward the text and the last lands on the left opening rightward,
// so the pair hugs the text in EVERY target (page, clipboard, WhatsApp)
// with no CSS and no bidi controls. The trade-off is deliberate: logical
// order is close-first, but inner text, search, and selection are unaffected.
// Tags pass through verbatim; unbalanced and nested parens are left untouched.
func ornateParens(s string) string {
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
				b.WriteString(`<span class="paren-orn">` + ornateClose)
				b.WriteString(s[i+1 : end])
				b.WriteString(ornateOpen + `</span>`)
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

// arabicIndicDigits maps Latin digits to Arabic-Indic digits for
// mushaf-style end-of-ayah markers.
var arabicIndicDigits = [10]rune{'٠', '١', '٢', '٣', '٤', '٥', '٦', '٧', '٨', '٩'}

// ArabicDigits renders n with Arabic-Indic digits (e.g. 114 → ١١٤) for
// end-of-ayah markers that flow inside RTL verse text, like quran.com.
func ArabicDigits(n int) string {
	if n == 0 {
		return string(arabicIndicDigits[0])
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var out []rune
	for n > 0 {
		out = append([]rune{arabicIndicDigits[n%10]}, out...)
		n /= 10
	}
	if neg {
		out = append([]rune{'-'}, out...)
	}
	return string(out)
}

// ArabicHTML escapes plain Arabic text and converts parenthesized runs to
// ornate pairs, which stay symmetric in RTL context by construction.
// Output carries no bidi control characters, so copy-paste stays clean.
func ArabicHTML(s string) template.HTML {
	return template.HTML(ornateParens(template.HTMLEscapeString(s)))
}

// TafsirHTMLFor renders already-sanitized tafsir HTML for one edition
// language. Arabic editions get ornate parens (see ornateParens); Latin
// editions pass through verbatim so English parentheses stay untouched.
func TafsirHTMLFor(lang string, h template.HTML) template.HTML {
	if lang == "arabic" {
		return TafsirHTML(h)
	}
	return h
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
