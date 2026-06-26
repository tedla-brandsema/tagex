# Lifecycle hooks

A struct being processed can opt into callbacks that run around its directives.
Implement any subset of these interfaces on the struct (pointer receiver, since
`ProcessStruct` takes a pointer):

```go
type PreProcessor interface {
	Before() error
}

type SuccessPostProcessor interface {
	Success() error
}

type FailurePostProcessor interface {
	Failure(cause error) error
}
```

The order during `ProcessStruct`:

1. `Before()` runs first. If it returns an error, processing stops and no
   directives run.
2. Directives run over the struct's fields.
3. On success, `Success()` runs.
4. On the first directive failure, `Failure(cause)` runs with the underlying
   error, and `ProcessStruct` returns that cause.

A hook returning an error is wrapped in a `*HookError` (carrying the hook name
and, for `Failure`, the original `cause`). Example:

```go
type Record struct {
	Name string `check:"audit"`
}

func (r *Record) Before() error  { log.Println("validating", r.Name); return nil }
func (r *Record) Success() error { log.Println("ok"); return nil }
func (r *Record) Failure(cause error) error {
	log.Println("rejected:", cause)
	return nil
}
```

Returning `nil` from `Failure` lets the original directive error propagate as the
result of `ProcessStruct`; returning a non-nil error from a hook replaces the
result with a `*HookError`.

## Using hooks without tag processing

The three interfaces are independent of directives and tags — they're just a
before/success/failure lifecycle. If a value implements them, you can run the
hooks directly, without any `Tag`, directive, or `ProcessStruct` call, via the
`Invoke*` functions:

```go
order := &Order{ /* implements PreProcessor, SuccessPostProcessor, ... */ }

if err := tagex.InvokePreProcessor(order); err != nil {
	return err
}

// ... do work ...

if err := doWork(order); err != nil {
	return tagex.InvokeFailurePostProcessor(order, err)
}
return tagex.InvokeSuccessPostProcessor(order)
```

Each `Invoke*` function calls the corresponding method if the value implements
the interface and is a no-op otherwise. The value can be any type that
implements the interface — not just a struct. `ProcessStruct` runs these same
hooks automatically, so a type that implements them works both standalone and
during tag processing.
