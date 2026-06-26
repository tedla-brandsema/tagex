# Tagex backlog

Working notes for tagex. Completed work lives in git history / the CHANGELOG, not
here. Checkbox = actionable; the prose under each item is the *why*, so the
reasoning survives even when the original conversation doesn't.

## In Progress

_(nothing in flight)_

## Backlog

- [ ] **Recurse into slices, arrays, and maps of structs.**
  `processStructFields` (tag.go) only descends into `struct` and `*struct`
  fields. A `[]LineItem` or `map[string]Address` whose elements carry tags is
  silently skipped — its directives never run. For a validation library that is
  a real correctness gap, not a missing nicety: a user reasonably expects
  `Items []LineItem` to be validated. Known debt since day one (it was "obvious"
  and never written down — hence this file).
  Implementation notes: walk slice/array elements and map values; preserve
  dotted field paths in errors (e.g. `Items[2].SKU`, `ByID[abc].Name`); map
  *keys* are not addressable, so leave them out of scope.

- [ ] **Resolve the `HandleError` footgun.**
  `HandleError` is exported, but `Error()` returns `""` when `Nested == nil`. An
  error that stringifies to empty is a trap for logging and `errors.Is`/`As`.
  Decide one of: unexport it (it looks like internal surface), guarantee
  `Nested` is always non-nil at construction, or return a non-empty fallback
  string. Public type, so settle it before 1.0.

- [ ] **Fuzz `kv` / `splitTagValue`.**
  ~15 lines of `go test -fuzz` asserting the tag parser never panics and that a
  parsed pair round-trips. Worth doing regardless of whether the grammar is ever
  widened — it hardens the parser we have and turns "string parsing is scary"
  into a proof. Cheap, high confidence.

- [ ] **Add a CHANGELOG and a stability statement before tagging 1.0.**
  The API is small and coherent but still 0.x. Document the pre-1.0 status and
  what "stable" will mean, so adopters know what they're signing up for.

- [ ] **(Deferred, on demand) Allow `=` inside param values via `SplitN`.**
  `kv` uses `strings.Split(pair, "=")` and requires exactly two parts, so a
  value can't contain `=` (`pattern=^a=b$`, base64 padding, query strings all
  fail). The fix is one word — `strings.SplitN(pair, "=", 2)` — and is purely
  additive: every currently-valid tag parses identically, only previously-
  rejected `2+ =` input becomes accepted, and SplitN is cheaper on pathological
  input. **Deliberately not done yet:** no third-party users exist to demand it,
  it slightly weakens loud-fail typo detection for string params, and partial
  support (`=` works but `,` still doesn't) can confuse more than a clean
  reserved-delimiter rule. Because it is backward-compatible, it can land the
  day a real adopter needs it with zero migration. Until then, wait.

## Decided / Won't do

- **No escaping/quoting for `,` or `=` in param values.**
  The tag grammar reserves `,` (separates args) and `=` (separates key/value).
  Supporting them inside values means a real escaping/quoting grammar — the
  permutations and failure modes (and parser-CVE surface) explode for the least
  rewarding part of the library. Tags are assumed compile-time, developer-
  authored constants, so the threat model doesn't justify it. Workaround:
  implement `ParamConverter` and use your own delimiter for structured values
  (e.g. `addends=1|2|3`). If tags ever come from a non-static source, revisit.
  (See the `=`-only relaxation in the backlog — that one stays on the table.)

- **No support for empty param values (`sep=`).**
  `kv` rejects an empty value, and that stays. A stray `min=` is far more often
  a typo than an intentional empty string, so failing loud at parse time is
  worth more than supporting the niche case. A genuinely-wanted empty string is
  already reachable: `required=false` leaves a string field at its `""` zero
  value, or a `ParamConverter` can produce one explicitly.
