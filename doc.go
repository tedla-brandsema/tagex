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
//  - Directives can override conversion by implementing ParamConverter.
//
// Lifecycle Hooks:
//
//  - If the target value implements PreProcessor, Before is invoked before processing.
//  - If it implements SuccessPostProcessor, Success is invoked after successful processing.
//  - If it implements FailurePostProcessor, Failure is invoked when processing fails.
//
// Errors:
//
//  - Failures are wrapped with ProcessError to capture stage, field path, directive,
//    and parameter context when available.
//  - Hook failures are wrapped in HookError.
package tagex
