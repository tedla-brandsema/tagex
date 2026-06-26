package tagex

import "testing"

type benchRangeDirective struct {
	Min int `param:"min"`
	Max int `param:"max"`
}

func (d *benchRangeDirective) Name() string {
	return "range"
}

func (d *benchRangeDirective) Mode() DirectiveMode {
	return EvalMode
}

func (d *benchRangeDirective) Handle(val int) (int, error) {
	return val, nil
}

type benchLengthDirective struct {
	Min int `param:"min"`
	Max int `param:"max"`
}

func (d *benchLengthDirective) Name() string {
	return "length"
}

func (d *benchLengthDirective) Mode() DirectiveMode {
	return EvalMode
}

func (d *benchLengthDirective) Handle(val string) (string, error) {
	return val, nil
}

type benchMultiplyDirective struct {
	Factor int `param:"factor"`
}

func (d *benchMultiplyDirective) Name() string {
	return "mul"
}

func (d *benchMultiplyDirective) Mode() DirectiveMode {
	return MutMode
}

func (d *benchMultiplyDirective) Handle(val int) (int, error) {
	return val * d.Factor, nil
}

type benchInner struct {
	Count int    `val:"range, min=0, max=10"`
	Label string `val:"length, min=1, max=10"`
}

type benchOuter struct {
	benchInner
	Inner benchInner
	Count int `mul:"mul, factor=2"`
}

func setupBenchTags() (*Tag, *Tag) {
	valTag := NewTag("val")
	RegisterDirective(valTag, &benchRangeDirective{})
	RegisterDirective(valTag, &benchLengthDirective{})

	mulTag := NewTag("mul")
	RegisterDirective(mulTag, &benchMultiplyDirective{})

	return valTag, mulTag
}

func BenchmarkProcessStruct_SingleTag(b *testing.B) {
	valTag, _ := setupBenchTags()
	data := benchOuter{
		benchInner: benchInner{Count: 5, Label: "ok"},
		Inner:      benchInner{Count: 5, Label: "ok"},
		Count:      3,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = valTag.ProcessStruct(&data)
	}
}

func BenchmarkProcessStruct_MultiTag(b *testing.B) {
	valTag, mulTag := setupBenchTags()
	data := benchOuter{
		benchInner: benchInner{Count: 5, Label: "ok"},
		Inner:      benchInner{Count: 5, Label: "ok"},
		Count:      3,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ProcessStruct(&data, valTag, mulTag)
	}
}

type benchFail struct {
	Count int `val:"range, min=bad, max=10"`
}

func BenchmarkProcessStruct_Failure(b *testing.B) {
	valTag, _ := setupBenchTags()
	data := benchFail{Count: 5}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = valTag.ProcessStruct(&data)
	}
}
