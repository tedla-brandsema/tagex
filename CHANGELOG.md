# Changelog

All notable changes to Tagex are documented here.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and the project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## Stability

Tagex is **pre-1.0 (0.x)**. The public API is still settling: while in 0.x a
breaking change bumps the **minor** version (`0.x.0`) and is called out below
under *Changed* or *Removed* with a migration note; patch releases (`0.x.y`) are
additive or fixes only. Pin a version.

`1.0` will mean the public surface — `NewTag`, `RegisterDirective`,
`ProcessStruct`, `Directive`, the lifecycle hooks, and the error types — is
frozen, and breaking changes thereafter require a major bump.

## [Unreleased]

### Added
- `ProcessStructAll` and `Tag.ProcessStructAll`, which process every field and
  return `errors.Join` of the per-field errors instead of stopping at the first,
  so a caller can report all failures at once (e.g. form validation). Each joined
  error stays a typed `*TagError`/`*ProcessError`, reachable with `errors.As`. A
  structural error such as exceeding the nesting limit still stops processing.

## [0.4.0] - 2026-06-26

Contains breaking changes — see *Changed*. Still pre-1.0; see *Stability*.

### Added
- Recursion into slices, arrays, and maps of structs/pointers. Errors carry
  indexed paths (`Items[2].SKU`, `ByID[key].N`), and `MutMode` directives write
  back through all of them, including map values.
- `DefaultConvert` is now exported, so a `ParamConverter` can fall back to the
  built-in primitive conversion — or map primitive tag args without registering
  a directive.
- Fuzz tests for the tag parser (`kv`, `splitTagValue`).
- A documentation set: rewritten README, a `docs/` guide, and runnable programs
  under `examples/`.
- Documented concurrency guarantee: a `Tag` is safe to share and `ProcessStruct`
  across goroutines once its directives are registered.
- `MustRegisterDirective`, which panics on a registration error — convenient for
  the common case of registering at startup.

### Changed
- **Breaking:** `NewTag` returns `*Tag` instead of `Tag`, removing a
  copy-with-mutex footgun. Pass the result directly:
  `RegisterDirective(tag, …)` / `ProcessStruct(data, tag)` (drop the `&`).
- **Breaking:** `ProcessStruct` and `Tag.ProcessStruct` return only `error`; the
  redundant `bool` (always `err == nil`) is gone. Replace `ok, err :=` with
  `err :=`.
- **Breaking:** `RegisterDirective` now returns an `error` — an
  `*EmptyDirectiveNameError` for a blank directive name, or a
  `*DuplicateDirectiveError` when a name is already registered (previously it
  overwrote silently). Setup-time callers can switch to `MustRegisterDirective`,
  which panics on that error.
- **Breaking:** `InvokePreProcessor` / `InvokeSuccessPostProcessor` /
  `InvokeFailurePostProcessor` no longer require a pointer to a struct — they run
  the hook if the value implements the interface and no-op otherwise, on any
  type. The lifecycle interfaces and these functions now live in `lifecycle.go`.
- `HandleError` is now a first-class, documented error type (pointer receiver
  like the others, non-empty `Error()`). It distinguishes a directive's `Handle`
  rejecting a value from a framework error, both of which occur at
  `StageDirective`.
- Minimum supported Go version is **1.22**, stated in `go.mod`, the README, and
  the package docs.
- Collections that were previously skipped are now processed, so invalid
  elements inside slices/arrays/maps surface validation errors they didn't
  before — review nested data on upgrade.

### Fixed
- Data race and incorrect results under concurrent `ProcessStruct` on a shared
  `Tag`: directives are now cloned per call, so per-invocation parameter state no
  longer mutates the shared registered instance.
- Passing the same `*Tag` twice to `ProcessStruct` no longer takes a recursive
  read lock (a latent deadlock) or runs its directives twice.
- Directive lookup no longer writes the registry map while holding a read lock.
- `splitTagValue` now rejects a whitespace-only directive name with
  `DirectiveParseError` instead of returning an empty id and a nil error.
- Cyclic data (a value that reaches itself through a pointer, slice, or map) now
  returns a `*ProcessError` wrapping a `*MaxDepthError` at a generous nesting
  limit instead of overflowing the stack, so processing data of unknown shape is
  safe and uses the same error handhold as every other failure.
- `ProcessStruct`'s input-validation failures are now typed and wrapped in
  `*ProcessError` (cause `*InvalidTargetError` for a non-pointer-to-struct,
  `*NilTagError` for a nil tag), so `errors.As(err, &ProcessError)` works for
  *every* error it returns — no bare `errors.New` cases remain.

## [0.3.0] - 2026-01-23

### Added
- `required` / `default` parameter semantics, with a conflict error when both are
  set on one parameter.
- `ProcessParams`, a public wrapper exposing parameter application for reuse.

### Changed
- `paramKey` is no longer exported.

## [0.2.0] - 2026-01-21

### Added
- Recursive processing of nested struct and pointer-to-struct fields.
- Success and failure post-processing hooks.
- Structured, typed error types carrying stage, field, directive, and parameter
  context.
- Multi-tag processing via `ProcessStruct(data, …tags)` with tag-scoped errors.
- Benchmarks.

## [0.1.0] - 2026-01-18

### Added
- Initial release: per-`Tag` directive registry, typed `Directive[T]`, eval and
  mutate modes, and an extensible parameter converter. (Renamed from "taggart".)

[Unreleased]: https://github.com/tedla-brandsema/tagex/compare/v0.4.0...HEAD
[0.4.0]: https://github.com/tedla-brandsema/tagex/compare/v0.3.0...v0.4.0
[0.3.0]: https://github.com/tedla-brandsema/tagex/compare/v0.2.0...v0.3.0
[0.2.0]: https://github.com/tedla-brandsema/tagex/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/tedla-brandsema/tagex/releases/tag/v0.1.0
