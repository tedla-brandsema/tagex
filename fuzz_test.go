package tagex

import (
	"strings"
	"testing"
)

// FuzzKV asserts the key/value splitter never panics and, on success, never
// yields an empty key. (An empty value is now valid, but only via an explicitly
// quoted empty string — "k=”"; a bare "k=" is still rejected.)
func FuzzKV(f *testing.F) {
	for _, s := range []string{"k=v", "k=", "=v", "k=v=w", "  k  =  v  ", "", "kv", "k==v", "=", "k=''", "k='a,b'"} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, pair string) {
		k, v, err := kv(pair)
		if err != nil {
			return // rejected input is fine
		}
		if k == "" {
			t.Errorf("kv(%q): nil error but empty key (v=%q)", pair, v)
		}
	})
}

// FuzzSplitTagValue asserts the directive/args splitter never panics and, on
// success, returns a non-empty directive id and a non-nil args map.
func FuzzSplitTagValue(f *testing.F) {
	for _, s := range []string{"range, min=2, max=4", "audit", "", "   ", " , x=y", "a,b,c", "k=v=w", "range,", ",,,", "range, min = 2"} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, tagVal string) {
		id, args, err := splitTagValue(tagVal)
		if err != nil {
			return // rejected input is fine
		}
		if id == "" {
			t.Errorf("splitTagValue(%q): nil error but empty directive id", tagVal)
		}
		if args == nil {
			t.Errorf("splitTagValue(%q): nil error but nil args map", tagVal)
		}
	})
}

// FuzzSplitChain asserts the ';' directive-chain splitter never panics and never
// yields an empty or whitespace-only segment, whatever separator soup it is fed.
func FuzzSplitChain(f *testing.F) {
	for _, s := range []string{"a;b", "a", "", ";", ";;", "a;;b", " ; a ; ", "trim;length, min=3", "a;b;c", ";trim;"} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, tagVal string) {
		for _, seg := range splitChain(tagVal) {
			if strings.TrimSpace(seg) == "" {
				t.Errorf("splitChain(%q): produced empty/blank segment %q", tagVal, seg)
			}
		}
	})
}
