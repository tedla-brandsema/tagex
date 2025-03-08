package taggart

import "testing"

type MinMax struct {
	Min int `param:"min"`
	Max int `param:"max"`
}

func TestProcessParams(t *testing.T) {
	args := map[string]string{
		"min": "2",
		"max": "4",
	}
	s := MinMax{}
	_, err := processParams(&s, args)
	if err != nil {
		t.Fatal(err)
		return
	}
	t.Logf("%v", s)
}
