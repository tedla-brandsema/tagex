# Tagex backlog

Working notes for tagex. Completed work lives in git history / the CHANGELOG, not
here. Checkbox = actionable; the prose under each item is the *why*, so the
reasoning survives even when the original conversation doesn't.

## In Progress

_(nothing in flight)_

## Backlog

- [ ] **Stop unbounded recursion on cyclic data (depth cap → error).**
  `processValue` recurses through pointers, slices, and maps without a visited
  set, so a self-referential graph (a struct that reaches itself via a pointer,
  slice, or map) recurses until the stack overflows — a hard process crash. The
  *limitation* (no graph support) is intended; the *failure mode* (crash vs.
  error) is not. Add a depth cap that returns a typed error when nesting exceeds
  a generous constant, converting the one "unsafe under arbitrary input" path
  into a clean error. Matters for anything that feeds Tagex structs whose shape
  it doesn't control (e.g. generic validation middleware).

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

- **Recursion stops at interface fields and map keys; no cycle detection.**
  Processing now descends into structs, pointers, slices, arrays, and maps of
  structs, but not interface-typed fields (the concrete value isn't addressable,
  so `MutMode` couldn't write anyway) or map *keys*. It also does not guard
  against cycles: a self-referential pointer/slice/map graph recurses without
  bound. Tagex targets data/DTO structs, where cycles are unusual, and a visited
  set is real complexity for a rare case. Documented as "don't process cyclic
  data"; revisit (backward-compatibly) if a real adopter needs either.
