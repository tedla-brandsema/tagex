package tagex

import (
	"reflect"
	"strings"
	"testing"
)

func TestSplitTopN(t *testing.T) {
	tests := []struct {
		name string
		s    string
		sep  byte
		n    int
		want []string
	}{
		{"plain", "a;b;c", ';', -1, []string{"a", "b", "c"}},
		{"cap n=2 keeps remainder", "a=b=c", '=', 2, []string{"a", "b=c"}},
		{"sep inside quotes ignored", "a,'b,c',d", ',', -1, []string{"a", "'b,c'", "d"}},
		{"escaped quote stays inside", "a,'b''c',d", ',', -1, []string{"a", "'b''c'", "d"}},
		{"whole value quoted", "'a;b'", ';', -1, []string{"'a;b'"}},
		{"empty", "", ';', -1, []string{""}},
		{"trailing sep", "a;", ';', -1, []string{"a", ""}},
		{"leading sep", ";a", ';', -1, []string{"", "a"}},
		{"n=0 is nil", "a;b", ';', 0, nil},
		{"n=1 is whole", "a;b;c", ';', 1, []string{"a;b;c"}},
		{"unbalanced quote swallows sep", "'a;b", ';', -1, []string{"'a;b"}},
		{"sep right after closing quote", "'a';b", ';', -1, []string{"'a'", "b"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := splitTopN(tt.s, tt.sep, tt.n)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("splitTopN(%q, %q, %d) = %q, want %q", tt.s, tt.sep, tt.n, got, tt.want)
			}
		})
	}
}

func TestUnquote(t *testing.T) {
	tests := []struct{ in, want string }{
		{"abc", "abc"},
		{"  abc  ", "abc"},
		{"'abc'", "abc"},
		{"'  ab  '", "  ab  "}, // interior whitespace preserved
		{"''", ""},
		{"'a''b'", "a'b"},
		{"'it''s'", "it's"},
		{"'a,b;c=d'", "a,b;c=d"}, // reserved chars survive inside quotes
		{"'", "'"},               // too short to be a quoted span
		{"", ""},
		{"  '  '  ", "  "}, // trim outside, keep inside
	}
	for _, tt := range tests {
		if got := unquote(tt.in); got != tt.want {
			t.Errorf("unquote(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestKV_Quoted(t *testing.T) {
	tests := []struct {
		pair    string
		wantK   string
		wantV   string
		wantErr bool
	}{
		{`pattern='\d{1,3}'`, "pattern", `\d{1,3}`, false}, // the motivating case
		{"k='a;b'", "k", "a;b", false},
		{"k='a,b'", "k", "a,b", false},
		{"k=''", "k", "", false}, // explicit empty string
		{"k=", "", "", true},     // bare empty stays a typo guard
		{"k='it''s'", "k", "it's", false},
		{"foo=bar", "foo", "bar", false},
		{"k=v=w", "k", "v=w", false}, // option A still holds
		{"nokey", "", "", true},
		{"=v", "", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.pair, func(t *testing.T) {
			k, v, err := kv(tt.pair)
			if (err != nil) != tt.wantErr {
				t.Fatalf("kv(%q) err = %v, wantErr %v", tt.pair, err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if k != tt.wantK || v != tt.wantV {
				t.Errorf("kv(%q) = (%q, %q), want (%q, %q)", tt.pair, k, v, tt.wantK, tt.wantV)
			}
		})
	}
}

func TestSplitTagValue_Quoted(t *testing.T) {
	tests := []struct {
		in     string
		wantID string
		want   map[string]string
	}{
		{`regex, pattern='\d{1,3}'`, "regex", map[string]string{"pattern": `\d{1,3}`}},
		{"x, a='1,2,3'", "x", map[string]string{"a": "1,2,3"}},
		{"range, min=1, max=10", "range", map[string]string{"min": "1", "max": "10"}},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			id, args, err := splitTagValue(tt.in)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if id != tt.wantID {
				t.Errorf("id = %q, want %q", id, tt.wantID)
			}
			if !reflect.DeepEqual(args, tt.want) {
				t.Errorf("args = %v, want %v", args, tt.want)
			}
		})
	}
}

// splitChain must not treat a ';' inside quotes as a chain separator.
func TestSplitChain_RespectsQuotes(t *testing.T) {
	got := splitChain("a, x='p;q';b")
	want := []string{"a, x='p;q'", "b"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("splitChain = %q, want %q", got, want)
	}
}

// rxCapture records the parameter value its clone received, so the test can
// assert a reserved-char value travels intact through the whole pipeline.
type rxCapture struct {
	Pattern string `param:"pattern"`
	seen    *string
}

func (d *rxCapture) Name() string        { return "regex" }
func (d *rxCapture) Mode() DirectiveMode { return EvalMode }
func (d *rxCapture) Handle(val string) (string, error) {
	if d.seen != nil {
		*d.seen = d.Pattern
	}
	return val, nil
}

// End to end: a quoted regex containing both ',' and '{' reaches the directive
// verbatim. The struct-tag layer doubles the backslash, so the value tagex sees
// is `regex, pattern='\d{1,3}'`.
func TestProcessDirective_QuotedReservedChars(t *testing.T) {
	var seen string
	tag := NewTag(valTagKey)
	MustRegisterDirective(tag, &rxCapture{seen: &seen})

	type form struct {
		S string `val:"regex, pattern='\\d{1,3}'"`
	}
	if err := tag.ProcessStruct(&form{S: "x"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if seen != `\d{1,3}` {
		t.Errorf("directive received pattern %q, want %q", seen, `\d{1,3}`)
	}
}

// FuzzSplitTopN: the splitter never panics, and for the unlimited split joining
// the fields back with sep reconstructs the input exactly (only unquoted seps
// are ever removed, and content is never copied or altered). The capped split
// never exceeds its limit.
func FuzzSplitTopN(f *testing.F) {
	for _, s := range []string{"a;b", "a,'b,c',d", "'x'", "", ";", "a''b", "'unbalanced", "a;'b;c';d"} {
		f.Add(s, uint8(0))
	}
	seps := []byte{';', ',', '='}
	f.Fuzz(func(t *testing.T, s string, si uint8) {
		sep := seps[int(si)%len(seps)]
		got := splitTopN(s, sep, -1)
		if join := strings.Join(got, string(sep)); join != s {
			t.Errorf("splitTopN(%q, %q, -1) does not round-trip: join = %q", s, sep, join)
		}
		if capped := splitTopN(s, sep, 2); len(capped) > 2 {
			t.Errorf("splitTopN(%q, %q, 2) returned %d fields", s, sep, len(capped))
		}
	})
}

// FuzzUnquote: never panics and only ever shrinks (strip quotes / collapse ”);
// the result is never longer than the trimmed input.
func FuzzUnquote(f *testing.F) {
	for _, s := range []string{"abc", "'abc'", "''", "'a''b'", "  'x'  ", "'", "", "'a,b;c'"} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		if got := unquote(s); len(got) > len(strings.TrimSpace(s)) {
			t.Errorf("unquote(%q) = %q grew beyond trimmed input", s, got)
		}
	})
}
