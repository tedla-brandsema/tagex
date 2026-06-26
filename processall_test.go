package tagex

import (
	"errors"
	"testing"
)

// countLeafErrors returns how many leaf errors a (possibly errors.Join'd) error
// holds — the multi-error tree that ProcessStructAll produces.
func countLeafErrors(err error) int {
	if err == nil {
		return 0
	}
	if j, ok := err.(interface{ Unwrap() []error }); ok {
		n := 0
		for _, e := range j.Unwrap() {
			n += countLeafErrors(e)
		}
		return n
	}
	return 1
}

func TestProcessStructAll_AccumulatesVsShortCircuit(t *testing.T) {
	tag := NewTag(valTagKey)
	MustRegisterDirective(tag, &RangeDirective{})

	type Form struct {
		A int `val:"range, min=0, max=10"`
		B int `val:"range, min=0, max=10"`
		C int `val:"range, min=0, max=10"`
	}
	bad := Form{A: 99, B: 5, C: -1} // A and C fail, B passes

	// Short-circuit: ProcessStruct still returns exactly one error.
	if err := tag.ProcessStruct(&bad); err == nil {
		t.Fatal("ProcessStruct: expected an error")
	} else if n := countLeafErrors(err); n != 1 {
		t.Fatalf("ProcessStruct: want 1 error, got %d: %v", n, err)
	}

	// Accumulate: ProcessStructAll returns both failing fields, joined.
	err := ProcessStructAll(&bad, tag)
	if err == nil {
		t.Fatal("ProcessStructAll: expected errors")
	}
	if n := countLeafErrors(err); n != 2 {
		t.Fatalf("ProcessStructAll: want 2 errors, got %d: %v", n, err)
	}

	// Each leaf stays typed and inspectable — the uniform error handhold holds.
	var pe *ProcessError
	if !errors.As(err, &pe) {
		t.Fatalf("ProcessStructAll: want errors.As to find *ProcessError, got %v", err)
	}

	// A clean struct returns nil from both.
	good := Form{A: 1, B: 2, C: 3}
	if err := ProcessStructAll(&good, tag); err != nil {
		t.Fatalf("ProcessStructAll on valid struct: want nil, got %v", err)
	}
}
