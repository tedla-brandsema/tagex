package tagex

import "testing"

// FuzzKV asserts the key/value splitter never panics and, on success, never
// yields an empty key or value.
func FuzzKV(f *testing.F) {
	for _, s := range []string{"k=v", "k=", "=v", "k=v=w", "  k  =  v  ", "", "kv", "k==v", "="} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, pair string) {
		k, v, err := kv(pair)
		if err != nil {
			return // rejected input is fine
		}
		if k == "" || v == "" {
			t.Errorf("kv(%q): nil error but empty key/value (k=%q v=%q)", pair, k, v)
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
