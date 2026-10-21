# Architecture review — 2026-10-21

## Scope and constraints

Second source review of the images/pull vertical slice. Keep Go, the existing
engine/service/UI boundaries and the current feature scope. No build, test,
lint, gallery or daemon runs are authorized. Test sources below are regression
specifications, not evidence of passing behavior.

## Work checklist

- [x] Inspect package boundaries and resource ownership using the graph and source.
- [x] Separate Docker configuration selection from credential-helper execution.
- [x] Bound each removal attempt and classify adapter failures at the service boundary.
- [x] Keep Docker-context restrictions out of offline fake mode.
- [x] Add focused regression specifications and update architecture documentation.
- [x] Format changed Go files and inspect the diff.

## Source review outcome

A separate source reviewer found no concrete critical or important issues in
the bounded diff. Its feedback led to clearer review status and an additional
regression specification for cancellation after reference checking. Formatting
and `git diff --check` completed; executable correctness remains unverified.
The commit and remote push are reported in the task hand-off rather than
predicted in this document.

## Initial findings and implemented approach

The existing engine interface and service facade are sufficient for this slice;
adding repositories, generic job frameworks or more abstraction would add cost
without an additional consumer. Improve the existing boundaries instead.

Before this review, credential lookup mixed configuration I/O, credential selection,
decoding and helper execution. Legacy-key lookup iterates a map, making ambiguous
configurations nondeterministic. Extract those responsibilities and give exact
keys precedence, followed by sorted legacy keys.

Removal checks and deletion previously used an unbounded session context. Give
each image its own deadline, stop adapter calls after cancellation, and classify
adapter errors before returning them. Keep non-forced deletion and per-image
results; no automatic retries for destructive operations.

The entrypoint previously rejected Docker-context settings before selecting an engine. Move
that validation to Docker construction so fake mode stays offline and usable.

## Remaining review risks

Dependency/API integration, event reconnect readiness and replay duplication,
long-session job history growth, and inspection cache invalidation remain future
work. This bounded review does not establish release readiness or exhaustive
correctness.
