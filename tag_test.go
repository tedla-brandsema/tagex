package taggart

import (
	"fmt"
	"testing"
)

var valTag = &Tag{
	Key: "val",
}

var NumDirective = NewDirective("num", func(val int, args []string) error {
	fmt.Printf("handling int value: %d\n", val)
	return nil
})

var StrDirective = NewDirective("str", func(val string, args []string) error {
	fmt.Printf("handling string value: %s\n", val)
	return nil
})

type MyStruct struct {
	Number int    `val:"num"`
	Word   string `val:"str"`
}

func init() {
	valTag.Register(NumDirective)
	valTag.Register(StrDirective)
}

func TestProcessStruct(t *testing.T) {
	strct := &MyStruct{
		Number: 12,
		Word:   "Pluk",
	}

	if ok, err := ProcessStruct(valTag, strct); !ok {
		t.Fatal(err)
	}
	t.Log("success!")

}
