// Package tagex provides an extensible system for applying custom struct tags.
//
// Tagex lets you define directives that evaluate or mutate field values based on
// tag metadata. A tag key (such as "check") identifies the tag, the directive
// name selects the handler, and directive parameters are mapped from tag args
// onto the directive's own struct fields via `param` tags.
//
// Typical usage:
//
//  - Create a Tag with NewTag.
//  - Implement a Directive[T] for the field type you want to handle.
//  - Register the directive with RegisterDirective.
//  - Call ProcessStruct on a pointer to a struct to execute directives.
//
// Directive Mode:
//
//  - EvalMode evaluates a field value and returns an error on failure.
//  - MutMode evaluates and writes the returned value back to the field.
//
// Parameter Conversion:
//
//  - Parameters are read from the tag and assigned to `param`-tagged fields
//    on the directive.
//  - The default converters support string, int, float64, and bool.
//  - Custom converters can be registered per Tag via SetConverter.
//
// Pre/Post Processing:
//
//  - If the target value implements PreProcessor or PostProcessor, the Before
//    or After methods are invoked around directive processing.
package tagex
