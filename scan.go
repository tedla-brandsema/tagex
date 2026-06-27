package tagex

import "strings"

// This file is the tag-value parser: it turns a raw struct-tag string such as
//
//	"trim;length, min=3, max=20"
//	"regex, pattern='\\d{1,3}'"
//
// into directive segments, a directive name, and an args map. All splitting goes
// through one quote-aware scanner (splitTopN) so that the separators ';', ',',
// and '=' can appear literally inside a value when it is wrapped in single
// quotes. A literal single quote inside a quoted value is written doubled ('').
//
// Single quotes are used because the struct-tag value is itself delimited by
// double quotes (`val:"..."`), and Go raw-string literals by backticks; the
// single quote is the only quote character free for tagex's own grammar.

// quote is the value-quoting character for tag values.
const quote = '\''

// splitTopN splits s on sep, ignoring any sep that falls inside a single-quoted
// span. It mirrors strings.SplitN: n < 0 splits on every top-level sep, while
// n > 0 caps the result at n fields (the final field keeps the unsplit
// remainder, including further seps). A doubled quote (”) inside a quoted span
// is an escaped quote and does not end the span. splitTopN never copies: each
// field is a sub-slice of s, with quote characters left intact for unquote.
func splitTopN(s string, sep byte, n int) []string {
	if n == 0 {
		return nil
	}
	var out []string
	start := 0
	inQuote := false
	for i := 0; i < len(s); i++ {
		switch c := s[i]; {
		case c == quote:
			if inQuote && i+1 < len(s) && s[i+1] == quote {
				i++ // escaped '' — consume both, stay inside the span
				continue
			}
			inQuote = !inQuote
		case c == sep && !inQuote && (n < 0 || len(out) < n-1):
			out = append(out, s[start:i])
			start = i + 1
		}
	}
	return append(out, s[start:])
}

// isQuoted reports whether s is wrapped in a matching pair of single quotes.
func isQuoted(s string) bool {
	return len(s) >= 2 && s[0] == quote && s[len(s)-1] == quote
}

// unquote trims surrounding whitespace from s and, if the result is wrapped in
// single quotes, strips them and collapses each escaped ” to a single quote.
// An unquoted value is returned trimmed (preserving today's behaviour); a quoted
// value keeps its interior verbatim, so leading/trailing spaces and the reserved
// separators survive inside the quotes.
func unquote(s string) string {
	s = strings.TrimSpace(s)
	if !isQuoted(s) {
		return s
	}
	inner := s[1 : len(s)-1]
	if strings.IndexByte(inner, quote) >= 0 {
		inner = strings.ReplaceAll(inner, "''", "'")
	}
	return inner
}

// splitChain splits a tag value into its directive segments on ';' (outside
// quotes), dropping any empty or whitespace-only segment. A leading, trailing,
// or doubled ';' is therefore harmless rather than an error.
func splitChain(tagValue string) []string {
	parts := splitTopN(tagValue, ';', -1)
	segs := make([]string, 0, len(parts))
	for _, p := range parts {
		if strings.TrimSpace(p) != "" {
			segs = append(segs, p)
		}
	}
	return segs
}

// splitTagValue parses one directive segment into its name and args. The name is
// the text before the first top-level ',', and each remaining comma-separated
// field is a key=value pair (see kv).
func splitTagValue(tagVal string) (id string, args map[string]string, err error) {
	parts := splitTopN(tagVal, ',', -1)
	id = strings.TrimSpace(parts[0])
	if id == "" {
		return "", nil, &DirectiveParseError{TagValue: tagVal}
	}
	args, err = extractPairs(parts[1:])
	return id, args, err
}

func extractPairs(args []string) (map[string]string, error) {
	pairs := make(map[string]string)
	for _, pair := range args {
		k, v, err := kv(pair)
		if err != nil {
			return nil, err
		}
		pairs[k] = v
	}
	return pairs, nil
}

// kv splits a "key=value" pair on its first top-level '=', so a value may itself
// contain '='. The value may be single-quoted to hold ',', ';', or surrounding
// whitespace literally; a quoted empty value (”) is an explicit empty string,
// whereas a bare "key=" remains a *ParamParseError (a far likelier typo).
func kv(pair string) (k string, v string, err error) {
	parts := splitTopN(pair, '=', 2)
	if len(parts) == 2 {
		k = strings.TrimSpace(parts[0])
		raw := strings.TrimSpace(parts[1])
		v = unquote(raw)
		if k != "" && (v != "" || isQuoted(raw)) {
			return k, v, nil
		}
	}
	return "", "", &ParamParseError{Pair: strings.TrimSpace(pair)}
}
