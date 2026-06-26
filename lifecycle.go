package tagex

// Lifecycle hooks let a value run custom logic before and after it is processed.
//
// The interfaces are independent of tag processing: any value that implements
// one can be driven directly through the Invoke* functions in this file, without
// registering directives or calling ProcessStruct. ProcessStruct invokes the
// same hooks automatically around directive processing, so a type that
// implements them works in both contexts.

// PreProcessor runs before processing begins.
type PreProcessor interface {
	Before() error
}

// SuccessPostProcessor runs after processing completes successfully.
type SuccessPostProcessor interface {
	Success() error
}

// FailurePostProcessor runs after processing fails, receiving the cause.
type FailurePostProcessor interface {
	Failure(cause error) error
}

// InvokePreProcessor calls Before on v if it implements PreProcessor, and is a
// no-op (returning nil) otherwise. Use it to run the pre-processing hook on its
// own, outside of ProcessStruct.
func InvokePreProcessor(v any) error {
	if p, ok := v.(PreProcessor); ok {
		return p.Before()
	}
	return nil
}

// InvokeSuccessPostProcessor calls Success on v if it implements
// SuccessPostProcessor, and is a no-op (returning nil) otherwise. Use it to run
// the success hook on its own, outside of ProcessStruct.
func InvokeSuccessPostProcessor(v any) error {
	if p, ok := v.(SuccessPostProcessor); ok {
		return p.Success()
	}
	return nil
}

// InvokeFailurePostProcessor calls Failure(cause) on v if it implements
// FailurePostProcessor, and is a no-op (returning nil) otherwise. Use it to run
// the failure hook on its own, outside of ProcessStruct.
func InvokeFailurePostProcessor(v any, cause error) error {
	if p, ok := v.(FailurePostProcessor); ok {
		return p.Failure(cause)
	}
	return nil
}
