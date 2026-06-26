package tagex

import (
	"errors"
	"testing"
)

var (
	_ PreProcessor         = (*lifecycleRecorder)(nil)
	_ SuccessPostProcessor = (*lifecycleRecorder)(nil)
	_ FailurePostProcessor = (*lifecycleRecorder)(nil)
)

type lifecycleRecorder struct {
	beforeCalled  bool
	successCalled bool
	failureCalled bool
	failureCause  error
}

func (r *lifecycleRecorder) Before() error  { r.beforeCalled = true; return nil }
func (r *lifecycleRecorder) Success() error { r.successCalled = true; return nil }
func (r *lifecycleRecorder) Failure(cause error) error {
	r.failureCalled = true
	r.failureCause = cause
	return nil
}

// TestInvokeHooks_Standalone drives the hooks directly, without any Tag,
// directive, or ProcessStruct call — the independent-use case.
func TestInvokeHooks_Standalone(t *testing.T) {
	r := &lifecycleRecorder{}

	if err := InvokePreProcessor(r); err != nil {
		t.Fatalf("InvokePreProcessor: %v", err)
	}
	if !r.beforeCalled {
		t.Error("Before was not called")
	}

	if err := InvokeSuccessPostProcessor(r); err != nil {
		t.Fatalf("InvokeSuccessPostProcessor: %v", err)
	}
	if !r.successCalled {
		t.Error("Success was not called")
	}

	cause := errors.New("boom")
	if err := InvokeFailurePostProcessor(r, cause); err != nil {
		t.Fatalf("InvokeFailurePostProcessor: %v", err)
	}
	if !r.failureCalled || !errors.Is(r.failureCause, cause) {
		t.Errorf("Failure not called with cause: called=%v cause=%v", r.failureCalled, r.failureCause)
	}
}

type lifecycleNoHooks struct{}

// TestInvokeHooks_NotImplemented confirms the Invoke* functions are no-ops when
// the value implements none of the interfaces.
func TestInvokeHooks_NotImplemented(t *testing.T) {
	v := &lifecycleNoHooks{}
	if err := InvokePreProcessor(v); err != nil {
		t.Errorf("InvokePreProcessor: expected nil, got %v", err)
	}
	if err := InvokeSuccessPostProcessor(v); err != nil {
		t.Errorf("InvokeSuccessPostProcessor: expected nil, got %v", err)
	}
	if err := InvokeFailurePostProcessor(v, nil); err != nil {
		t.Errorf("InvokeFailurePostProcessor: expected nil, got %v", err)
	}
}

type lifecycleFailing struct{}

func (*lifecycleFailing) Before() error { return errors.New("before failed") }

func TestInvokePreProcessor_PropagatesError(t *testing.T) {
	err := InvokePreProcessor(&lifecycleFailing{})
	if err == nil || err.Error() != "before failed" {
		t.Fatalf("expected \"before failed\", got %v", err)
	}
}

type lifecycleStatus int

func (s lifecycleStatus) Before() error {
	if s < 0 {
		return errors.New("negative status")
	}
	return nil
}

// TestInvokeHooks_NonStructType confirms the hooks run on any type that
// implements the interface — not only pointers to structs — and no-op on a type
// that implements nothing.
func TestInvokeHooks_NonStructType(t *testing.T) {
	if err := InvokePreProcessor(lifecycleStatus(1)); err != nil {
		t.Fatalf("valid status: expected nil, got %v", err)
	}
	if err := InvokePreProcessor(lifecycleStatus(-1)); err == nil {
		t.Fatal("negative status: expected error from Before")
	}
	if err := InvokePreProcessor("implements nothing"); err != nil {
		t.Errorf("non-implementer: expected no-op nil, got %v", err)
	}
}
