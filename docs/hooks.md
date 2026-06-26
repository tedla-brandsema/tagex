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
