package tagex

import (
	"sync"
	"testing"
)

type doubleDirective struct{}

func (d *doubleDirective) Name() string                { return "double" }
func (d *doubleDirective) Mode() DirectiveMode         { return MutMode }
func (d *doubleDirective) Handle(val int) (int, error) { return val * 2, nil }

// TestProcessStructDuplicateTag ensures passing the same Tag twice neither
// deadlocks (recursive read lock) nor runs its directives twice — a MutMode
// directive must mutate once.
func TestProcessStructDuplicateTag(t *testing.T) {
	tag := NewTag("m")
	RegisterDirective(tag, &doubleDirective{})

	type S struct {
		V int `m:"double"`
	}
	s := S{V: 5}
	err := ProcessStruct(&s, tag, tag)
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if s.V != 10 {
		t.Fatalf("V = %d, want 10 (processed once, not twice)", s.V)
	}
}

// TestConcurrentProcessStruct runs ProcessStruct on one shared Tag from many
// goroutines using different parameters. Before per-call cloning, the shared
// directive's param fields were written concurrently: a data race (caught by
// -race) and wrong results (one goroutine's bounds clobbering another's).
func TestConcurrentProcessStruct(t *testing.T) {
	tag := NewTag("check")
	RegisterDirective(tag, &RangeDirective{})

	type low struct {
		V int `check:"range, min=0, max=10"`
	}
	type high struct {
		V int `check:"range, min=100, max=200"`
	}

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			l := low{V: 5} // in [0,10]
			if err := tag.ProcessStruct(&l); err != nil {
				t.Errorf("low: err=%v", err)
			}
		}()
		go func() {
			defer wg.Done()
			h := high{V: 150} // in [100,200]
			if err := tag.ProcessStruct(&h); err != nil {
				t.Errorf("high: err=%v", err)
			}
		}()
	}
	wg.Wait()
}
