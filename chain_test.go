package tagex

import (
	"reflect"
	"strings"
	"testing"
)

// trimDirective is a MutMode string directive used to prove chain ordering: it
// changes the value (and its length), so it interacts with LengthDirective.
type trimDirective struct{}

func (d *trimDirective) Name() string        { return "trim" }
func (d *trimDirective) Mode() DirectiveMode { return MutMode }
func (d *trimDirective) Handle(val string) (string, error) {
	return strings.TrimSpace(val), nil
}

func chainTag(t *testing.T) *Tag {
	t.Helper()
	tag := NewTag(valTagKey)
	MustRegisterDirective(tag, &trimDirective{})
	MustRegisterDirective(tag, &LengthDirective{})
	return tag
}

// Order is semantic: trim then length sees the shortened value; length then trim
// sees the original. Same input, opposite outcomes.
func TestProcessDirective_ChainOrderMatters(t *testing.T) {
	tag := chainTag(t)

	// trim first: "  ab  " -> "ab" (len 2) -> length min=3 fails.
	s1 := "  ab  "
	v1 := reflect.ValueOf(&s1).Elem()
	if err := processDirective(tag, "trim;length, min=3, max=100", v1); err == nil {
		t.Errorf("trim;length: expected length failure after trim, got nil (value=%q)", s1)
	}

	// length first: "  ab  " (len 6) passes -> trim -> "ab".
	s2 := "  ab  "
	v2 := reflect.ValueOf(&s2).Elem()
	if err := processDirective(tag, "length, min=3, max=100;trim", v2); err != nil {
		t.Fatalf("length;trim: unexpected error: %v", err)
	}
	if s2 != "ab" {
		t.Errorf("length;trim: expected trimmed %q, got %q", "ab", s2)
	}
}

// A failed segment stops the chain: a later MutMode segment must not run.
func TestProcessDirective_ChainStopsAtFirstFailure(t *testing.T) {
	tag := chainTag(t)

	s := "  ab  " // len 6, below min=10
	v := reflect.ValueOf(&s).Elem()
	if err := processDirective(tag, "length, min=10, max=100;trim", v); err == nil {
		t.Fatal("expected length failure, got nil")
	}
	if s != "  ab  " {
		t.Errorf("trim after a failed length must not run; value changed to %q", s)
	}
}

// Caveat: under ProcessStructAll a MutMode segment that already ran leaves the
// field mutated even though a later segment in the same chain fails. The partial
// (trimmed) value persists alongside the recorded error.
func TestProcessStructAll_ChainPartialMutationPersists(t *testing.T) {
	tag := chainTag(t)

	type form struct {
		Name string `val:"trim;length, min=10, max=100"`
	}
	f := &form{Name: "  ab  "}
	if err := ProcessStructAll(f, tag); err == nil {
		t.Fatal("expected length failure, got nil")
	}
	if f.Name != "ab" {
		t.Errorf("expected partial mutation %q to persist, got %q", "ab", f.Name)
	}
}

// Empty, doubled, leading, and trailing ';' segments are skipped, not errors.
func TestProcessDirective_ChainSkipsEmptySegments(t *testing.T) {
	tag := chainTag(t)

	s := "  ab  "
	v := reflect.ValueOf(&s).Elem()
	if err := processDirective(tag, ";trim;;length, min=1, max=10;", v); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s != "ab" {
		t.Errorf("expected %q, got %q", "ab", s)
	}
}
