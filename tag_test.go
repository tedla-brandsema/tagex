package taggart

import (
	"fmt"
	"testing"
)

var valTag = Tag{
	Key: "val",
}

var NumDirective = NewDirective("num", func(val int) error {
	fmt.Printf("handling int value: %d\n", val)
	return nil
})

//		Directive[int]{
//	Name: "num",
//	Handler: DirectiveHandleFunc[int](func(val int) error {
//		fmt.Println("handling int value")
//		return nil
//	}),
//}

var StrDirective = NewDirective("str", func(val string) error {
	fmt.Printf("handling string value: %s\n", val)
	return nil
})

//		Directive[string]{
//	Name: "str",
//	Handler: DirectiveHandleFunc[string](func(val string) error {
//		fmt.Println("handling string value")
//		return nil
//	}),
//}

type MyStruct struct {
	Number int    `val:"num"`
	Word   string `val:"str"`
}

func init() {

	RegisterDirective(NumDirective)
	RegisterDirective(StrDirective)
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
