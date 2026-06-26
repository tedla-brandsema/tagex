// Package tagex provides an extensible system for applying custom struct tags.
//
// Requires Go 1.22 or later.
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
//  - Register the directive with MustRegisterDirective (or RegisterDirective,
//    which returns an error instead of panicking).
//  - Call ProcessStruct on a pointer to a struct to execute directives.
//  - To apply multiple tags in one pass, call tagex.ProcessStruct(data, tag1, tag2, ...).
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
//  - The default converters support string, int, int64, float64, and bool.
//  - Directives can override conversion by implementing ParamConverter.
//  - DefaultConvert exposes the built-in conversion for reuse as a fallback.
//  - ProcessParams exposes this parameter application logic for reuse.
//
// Lifecycle Hooks:
//
//  - If the target value implements PreProcessor, Before is invoked before processing.
//  - If it implements SuccessPostProcessor, Success is invoked after successful processing.
//  - If it implements FailurePostProcessor, Failure is invoked when processing fails.
//  - The hooks are independent of tag processing: invoke them on their own with
//    InvokePreProcessor, InvokeSuccessPostProcessor, and InvokeFailurePostProcessor.
//
// Errors:
//
//  - Failures are wrapped with ProcessError to capture stage, field path, directive,
//    and parameter context when available.
//  - Hook failures are wrapped in HookError.
//
// Concurrency:
//
//  - Once its directives are registered, a Tag is safe to share and to call
//    ProcessStruct on from multiple goroutines; per-call state is kept off the
//    shared directive instances.
//  - RegisterDirective is the only mutating operation; register during setup,
//    before processing concurrently.
package tagex
