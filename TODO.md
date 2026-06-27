# Tagex backlog

Working notes for tagex. Completed work lives in git history / the CHANGELOG, not
here. Checkbox = actionable; the prose under each item is the *why*, so the
reasoning survives even when the original conversation doesn't.

## In Progress

_(nothing in flight)_

## Backlog

_(empty)_

## Decided / Won't do

- **No support for *unquoted* empty param values (bare `sep=`).**
  `kv` rejects a bare empty value, and that stays. A stray `min=` is far more
  often a typo than an intentional empty string, so failing loud at parse time
  is worth more than silently accepting it. A genuinely-wanted empty string is
  reachable explicitly: quote it (`sep=''`), use `required=false` to leave a
  string field at its `""` zero value, or have a `ParamConverter` produce one.

- **Recursion stops at interface fields and map keys; cycles are depth-capped,
  not precisely detected.**
  Processing descends into structs, pointers, slices, arrays, and maps of
  structs, but not interface-typed fields (the concrete value isn't addressable,
  so `MutMode` couldn't write anyway) or map *keys*. Cyclic data is bounded by a
  depth limit that returns a `*MaxDepthError` rather than crashing — a blunt cap,
  not pointer-identity cycle detection (json.Marshal-style), because Tagex
  targets data/DTO structs where cycles are misuse, not a feature, and a precise
  visited-set is more complexity than the case earns. Upgradeable
  backward-compatibly if a real adopter ever needs deep acyclic graphs past the
  limit.
