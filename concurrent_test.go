package tagex

import (
	"sync"
	"testing"
)

// TestConcurrentProcessStruct runs ProcessStruct on one shared Tag from many
// goroutines using different parameters. Before per-call cloning, the shared
// directive's param fields were written concurrently: a data race (caught by
// -race) and wrong results (one goroutine's bounds clobbering another's).
func TestConcurrentProcessStruct(t *testing.T) {
	tag := NewTag("check")
	RegisterDirective(&tag, &RangeDirective{})

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
			if ok, err := tag.ProcessStruct(&l); !ok || err != nil {
				t.Errorf("low: ok=%v err=%v", ok, err)
			}
		}()
		go func() {
			defer wg.Done()
			h := high{V: 150} // in [100,200]
			if ok, err := tag.ProcessStruct(&h); !ok || err != nil {
				t.Errorf("high: ok=%v err=%v", ok, err)
			}
		}()
	}
	wg.Wait()
}
