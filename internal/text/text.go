// Package text holds output guards used by the proxy and hooks.
package text

import (
	"strings"
	"unicode/utf8"
)

// StripANSI removes CSI/OSC sequences so filters parse plain text.
func StripANSI(s string) string {
	if !strings.ContainsRune(s, 0x1b) && !strings.ContainsRune(s, 0x9b) {
		return s
	}
	var b strings.Builder
	b.Grow(len(s))
	i := 0
	for i < len(s) {
		r, size := utf8.DecodeRuneInString(s[i:])
		if r == 0x1b {
			i += size
			if i >= len(s) {
				break
			}
			n, nsize := utf8.DecodeRuneInString(s[i:])
			switch n {
			case '[':
				i += nsize
				i = skipWhile(s, i, func(c rune) bool {
					return c < 0x40 || c > 0x7e
				})
				if i < len(s) {
					_, sz := utf8.DecodeRuneInString(s[i:])
					i += sz
				}
			case ']':
				i += nsize
				i = skipOSC(s, i)
			case '(':
				i += nsize
				if i < len(s) {
					_, sz := utf8.DecodeRuneInString(s[i:])
					i += sz
				}
			default:
				i += nsize
			}
			continue
		}
		if r == 0x9b {
			i += size
			i = skipWhile(s, i, func(c rune) bool {
				return c < 0x40 || c > 0x7e
			})
			if i < len(s) {
				_, sz := utf8.DecodeRuneInString(s[i:])
				i += sz
			}
			continue
		}
		b.WriteRune(r)
		i += size
	}
	return b.String()
}

func skipWhile(s string, i int, pred func(rune) bool) int {
	for i < len(s) {
		r, size := utf8.DecodeRuneInString(s[i:])
		if !pred(r) {
			return i
		}
		i += size
	}
	return i
}

func skipOSC(s string, i int) int {
	for i < len(s) {
		r, size := utf8.DecodeRuneInString(s[i:])
		if r == 0x07 {
			return i + size
		}
		if r == 0x1b {
			j := i + size
			if j < len(s) {
				n, nsize := utf8.DecodeRuneInString(s[j:])
				if n == '\\' {
					return j + nsize
				}
			}
		}
		i += size
	}
	return i
}
