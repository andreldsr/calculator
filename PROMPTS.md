# AI Prompts

A record of every prompt used to build this project with Claude Code (model: Claude Opus 5.5).

- **Developer prompts** are quoted verbatim, typos included.
- **Subagent prompts** are the briefs Claude wrote when the developer asked it to delegate to the project's subagents (`.claude/agents/test-writer.md`, `.claude/agents/code-reviewer.md`). They are reproduced in full under each step.
- **Outcome** summarizes what the prompt produced and the resulting commits.

Project-level context loaded into every prompt: [`CLAUDE.md`](CLAUDE.md) (workflow rules and architecture guardrails) and [`TASK_SPEC.md`](TASK_SPEC.md) (requirements and API contract).

---

## 1. Scaffold the repository

> Read @CLAUDE.md and @TASK_SPEC.md. Scaffold the repository skeleton only:
> backend go.mod (module path in CLAUDE.md), cmd/api/main.go with a minimal
> server and /healthz, empty internal packages, Vite React-TS app in frontend/
> with Vitest + RTL configured, strict tsconfig, ESLint with no-explicit-any,
> and a Makefile with the targets from CLAUDE.md. No business logic.
> Verify `go build ./...`, `npm run build` and `make test` succeed.

**Outcome:**
- **Backend:** Go module, a standard-library server with `GET /healthz` and graceful shutdown, a table-driven healthz test, and empty `internal/calculator` and `internal/handler` packages.
- **Frontend:** a Vite React-TS app. The template's oxlint was replaced with ESLint (typescript-eslint `strict`; `no-explicit-any`, `no-non-null-assertion` and `ban-ts-comment` are errors). It has a strict tsconfig, Vitest + React Testing Library with jsdom, and a dev proxy that forwards `/api` to `:8080`.
- **Makefile** with the CLAUDE.md targets. `make test` was verified in a `golang:1.27` Docker container, because the host has no `make` and no C compiler for `-race`.
- **Two problems found in the CLAUDE.md commands:**
  - `npx tsc --noEmit` checks nothing with Vite's project-references tsconfig, so the Makefile uses `tsc -b --noEmit`.
  - `gofmt -l .` exits 0 even when files are unformatted, so the Makefile fails explicitly on any output.

## 2. Initialize git

> Initialize git and commit every meaningful step.

**Outcome:** `git init -b main`, a repo-local git identity, and `.gitattributes` forcing LF line endings. The scaffold was split into four commits:
- `chore: add task spec, project guidelines and repo config`
- `feat(backend): scaffold Go module and API server with /healthz`
- `chore(frontend): scaffold Vite React-TS app with Vitest, RTL and ESLint`
- `chore: add Makefile with test, lint, typecheck and build targets`

## 3. Calculator tests (test-writer subagent)

> Use the test-writer subagent to write table-driven tests for
> internal/calculator covering every operation and error in the spec.

<details>
<summary>Subagent prompt: test-writer</summary>

````text
Write table-driven Go tests for the `internal/calculator` package of this repo (C:\Users\morae\calculator, Go module `github.com/andre/sezzle-calculator/backend`, code in `backend/`). Read TASK_SPEC.md and CLAUDE.md first. Create ONLY `backend/internal/calculator/calculator_test.go` (package `calculator`, internal test). Do not create or edit any other file — the package currently only has `doc.go`; the implementation will be written afterwards against your tests. Standard library only (`testing`, `errors`, `math`); no testify.

## API the tests must target (already decided — do not change it)

```go
type Operation string

const (
    OpAdd        Operation = "add"
    OpSubtract   Operation = "subtract"
    OpMultiply   Operation = "multiply"
    OpDivide     Operation = "divide"
    OpPower      Operation = "power"      // a ^ b
    OpSqrt       Operation = "sqrt"       // unary: √a, b is ignored
    OpPercentage Operation = "percentage" // a percent of b: a / 100 * b
)

var (
    ErrDivisionByZero       = errors.New("division by zero is not allowed") // exact text from the spec contract
    ErrNegativeSqrt         = errors.New("square root of a negative number is not allowed")
    ErrUnsupportedOperation = errors.New("unsupported operation")
    ErrInvalidOperand       = errors.New("operands must be finite numbers")
    ErrNonFiniteResult      = errors.New("result is not a finite number")
)

func Calculate(op Operation, a, b float64) (float64, error)
```

Semantics: on any error Calculate returns (0, err). Errors may be returned wrapped, so tests must check with `errors.Is`, never `==` or string matching — except one dedicated test that asserts `ErrDivisionByZero.Error()` equals the spec string exactly, since it's part of the API contract. NaN/±Inf inputs → ErrInvalidOperand (check operands before anything else, including for sqrt's `a`; for sqrt a non-finite `b` is ignored since b is unused). Any operation whose result would be ±Inf or NaN (overflow, 0^-1, (-8)^(1/3), percentage overflow…) → ErrNonFiniteResult. Unknown or empty operation strings (including wrong case like "ADD" and symbols like "+") → ErrUnsupportedOperation.

## Coverage I expect
- Each of the 7 operations: typical values, negatives, zero, fractional values, and large-but-finite values. Include 0.1+0.2 using an epsilon comparison (write a small `almostEqual` helper with relative tolerance); use exact comparison where results are exactly representable.
- Divide: by zero (positive, negative and zero dividend; also `-0.0` divisor via `math.Copysign(0, -1)`), 0/x = 0.
- Sqrt: 0, perfect squares, non-perfect (√2), negative → ErrNegativeSqrt, b ignored.
- Power: x^0 = 1, 0^0 = 1, negative base with integer exponent, fractional exponent, negative exponent, overflow → ErrNonFiniteResult, 0^-1 → ErrNonFiniteResult, negative base with fractional exponent → ErrNonFiniteResult.
- Percentage: 50 % of 200 = 100, 0 %, 100 %, negative values, fractional.
- Multiply/add/subtract overflow to ±Inf → ErrNonFiniteResult.
- Non-finite inputs for every operation → ErrInvalidOperand.
- Unsupported operations.
Structure: one table per concern (or one big table) with fields like name, op, a, b, want, wantErr; iterate with t.Run; when wantErr is set assert errors.Is and that the result is 0; otherwise assert no error and the value. Keep it readable and idiomatic; no redundant cases.

## Verify
Run `cd backend && gofmt -l ./internal/calculator && go vet ./internal/calculator/` and `go test ./internal/calculator/`. The build must fail ONLY with "undefined:" errors for the API above (missing implementation) — no syntax or type errors in the test file itself. Note: `-race` is unavailable on this Windows machine (no cgo), so run without it. Report the final case list grouped by operation and the exact failure output.
````

</details>

**Outcome:** `calculator_test.go` was written (261 lines, one table per operation, plus cases for non-finite operands, unsupported operations, the spec's error message, and that the errors are distinct from each other). It failed only with `undefined:` errors. It wasn't committed on its own, because CLAUDE.md forbids committing failing tests.

## 4. Implement the calculator

> Implement internal/calculator so all tests in calculator_test.go pass.
> Don't modify the tests. Use sentinel errors. Run go test -race -cover.

**Outcome:** `calculator.go` was written. The operand check comes first, then the operation, then the Inf/NaN result check. All tests passed with 100% coverage, including under `-race` in Docker.
Commit: `feat(backend): add calculator operations with domain errors`.

## 5. First backend review (code-reviewer subagent)

> Use the code-reviewer subagent on the backend changes.

<details>
<summary>Subagent prompt: code-reviewer</summary>

```text
Review the backend changes in this repo (C:\Users\morae\calculator). They are already committed, so the working-tree diff is empty — review this range instead: `git diff 22d29aa..HEAD -- backend/` (commits 966bcff "scaffold Go module and API server with /healthz" and 1387f40 "add calculator operations with domain errors"; also use `git show` on each). Check against TASK_SPEC.md and CLAUDE.md.

Context:
- The project is phased: the HTTP handler (`internal/handler`, POST /api/v1/calculate, 400/405 mapping, JSON validation) is intentionally NOT implemented yet — don't report its absence. Do assess whether the calculator API is a good foundation for it (e.g. how the handler will know `b` is optional for sqrt).
- Intentional design decisions, review them critically but don't flag them merely for existing: percentage = "a percent of b" computed as a/100*b; sqrt ignores b; NaN/Inf operands → ErrInvalidOperand checked before other errors; Inf/NaN results → ErrNonFiniteResult; sentinels returned unwrapped.
- Commands: `cd backend && gofmt -l . && go vet ./... && go test -cover ./...`. `-race` cannot run on this Windows host (no cgo/gcc); if you want race coverage run `MSYS_NO_PATHCONV=1 docker run --rm -v "$(pwd -W):/src:ro" -w /src -e GOFLAGS=-buildvcs=false golang:1.27 go test -race -cover ./...` from `backend/`.

Report findings ordered by severity with file:line. Don't invent issues; if it's fine, say so briefly.
```

</details>

**Outcome:** the review found one real bug, which was then confirmed independently: computing percentage as `a/100*b` gave `7% of 100 = 7.000000000000001`. It also raised:
- the handler would need the calculator to say whether an operation is valid and whether it is unary;
- results of `-0` are encoded as `{"result":-0}`;
- `log.Fatalf` in `main.go` skips deferred cleanup.

## 6. Handler tests (test-writer subagent)

> Use the test-writer subagent to write table-driven tests for
> internal/handler covering every operation and error in the spec.

<details>
<summary>Subagent prompt: test-writer</summary>

````text
Write table-driven Go tests for `internal/handler` in this repo (C:\Users\morae\calculator, Go module `github.com/andre/sezzle-calculator/backend`, code in `backend/`). Read TASK_SPEC.md, CLAUDE.md and `backend/internal/calculator/calculator.go` (already implemented; its sentinels, operation names and semantics are fixed) first. Create ONLY `backend/internal/handler/handler_test.go` (package `handler`). Do not create or edit any other file; the handler package currently only has `doc.go`. Standard library only (`net/http/httptest`, `encoding/json`, `strings`, `errors`, …); no testify.

## API the tests must target (already decided — do not change it)

```go
package handler

// CalculateFunc matches calculator.Calculate; injected so tests can stub it.
type CalculateFunc func(op calculator.Operation, a, b float64) (float64, error)

// New returns the handler for POST /api/v1/calculate.
func New(calc CalculateFunc) http.Handler
```
main.go will later do `mux.Handle("/api/v1/calculate", handler.New(calculator.Calculate))` — the handler does its own method check (the mux pattern has no method), so tests call `New(...).ServeHTTP` directly via httptest.

## Contract
Request JSON: `{"operation": string, "a": number, "b": number}`; `b` is optional for `sqrt` (ignored if present) and required for every other operation. Operation names: add, subtract, multiply, divide, power, sqrt, percentage (percentage = a percent of b).

Every response has `Content-Type: application/json`. Success: 200 with body exactly `{"result": <number>}` (no other keys). Errors: body exactly `{"error": "<message>"}` (no other keys). Status and exact messages:

| Situation | Status | error message |
|---|---|---|
| method other than POST (GET, PUT, DELETE, PATCH…) | 405, plus header `Allow: POST` | `method not allowed` |
| empty body, malformed JSON, wrong JSON types (e.g. `"a": "10"`), unknown fields, trailing data after the object, number out of float64 range (`1e400`), top-level non-object (array, string) | 400 | `invalid request body` |
| body larger than 1 MiB | 413 | `request body too large` |
| `operation` missing or `""` | 400 | `operation is required` |
| operation not supported (e.g. `modulo`, `ADD`, `+`) | 400 | `unsupported operation` |
| `a` missing or `null` | 400 | `a is required` |
| `b` missing or `null` for a non-sqrt operation | 400 | `b is required` |
| calculator returns ErrDivisionByZero | 400 | `division by zero is not allowed` (spec contract) |
| ErrNegativeSqrt / ErrNonFiniteResult / ErrUnsupportedOperation / ErrInvalidOperand | 400 | the sentinel's `.Error()` text |
| calculator returns any other (non-domain) error, incl. wrapped-unknown | 500 | `internal server error` (must not leak the underlying error text) |

Validation order: body decode → operation required → operation supported → a required → b required. So an unsupported op with `b` missing reports `unsupported operation`, and missing `a` and `b` reports `a is required`. A domain error wrapped with `fmt.Errorf("...: %w", calculator.ErrDivisionByZero)` returned by a stub must still map to 400 with the sentinel's message (handler uses errors.Is).

## Coverage I expect
- Happy path for all 7 operations through the real `calculator.Calculate`, including the spec example `{"operation":"divide","a":10,"b":2}` → `{"result":5}`. Use values with exactly representable results; for percentage use 50 % of 200 = 100 (avoid cases like 7 % of 100 — that calculator path has a known rounding issue being fixed separately). Include `b: 0` being accepted as present (e.g. add 5 + 0 = 5), negative and fractional operands, sqrt with and without `b`.
- Every error row in the table above, using the real calculator for domain errors (10/0, sqrt -4, power 10^309, …) and a stub CalculateFunc for the 500 and wrapped-error cases. Also assert the stub is NOT called when validation fails (e.g. a counter).
- 405 for several methods, checking the Allow header.
- 413 with a >1 MiB body (e.g. valid JSON padded with whitespace, or a long string field).
Assert status, Content-Type, and the decoded body (decode into `map[string]any` or `map[string]json.RawMessage` to check the exact key set). Keep tables readable: name, method, body, want status, want result or want error; small helpers are fine.

## Verify
Run `cd backend && gofmt -l ./internal/handler && go vet ./internal/handler/` and `go test ./internal/handler/`. The build must fail ONLY with "undefined:" errors for `New` / `CalculateFunc` (missing implementation) — use `go test -gcflags=-e` to see all errors. No syntax or type errors in the test file itself. `-race` is unavailable on this host (no cgo). Optionally, prove the tests are consistent by writing a throwaway implementation in your scratchpad directory (outside the repo, never in it) and confirming they pass there. Report the case list grouped by concern and the exact failure output.
````

</details>

**Outcome:** `handler_test.go` was written (380 lines). It covers:
- success for all 7 operations, with `b: 0` counted as present;
- every validation message, and the order fields are checked in;
- calculator errors mapped to 400, including wrapped ones;
- 500 without leaking details;
- 405 with `Allow: POST`;
- the 1 MiB limit at the exact boundary.

It failed only with `undefined: New` and `undefined: CalculateFunc`.

## 7. Implement the handler

> Implement internal/handler so all tests in calculator_test.go pass.
> Don't modify the tests. Use sentinel errors. Run go test -race -cover.

(The prompt names `calculator_test.go` but asks for `internal/handler`, so it was taken to mean `handler_test.go`.)

**Outcome:**
- **`handler.go`:**
  - request fields are pointers, so a missing `b` is different from `b: 0`;
  - `MaxBytesReader` limits the body to 1 MiB;
  - `DisallowUnknownFields`, plus a check for trailing data;
  - unexported sentinel errors for validation messages;
  - calculator errors mapped with `errors.Is`.
- **Calculator:** gained `Operation.Valid()` and `Operation.Unary()`, with a map-based dispatch, so the handler doesn't duplicate the operation list.
- **`main.go`:** wires `/api/v1/calculate` with no method in the pattern, so the handler can answer 405 with a JSON body.
- **Verification:** coverage is 100% for the calculator and 98.2% for the handler under `-race`. The running server was smoke-tested with curl.

Commit: `feat(backend): add POST /api/v1/calculate endpoint`.

## 8. Full backend review (code-reviewer subagent)

> Now lets review everything Use the code-reviewer subagent on the backend changes.

<details>
<summary>Subagent prompt: code-reviewer</summary>

```text
Review all backend work in this repo (C:\Users\morae\calculator). Everything is committed, so the working-tree diff is empty — review the range `git diff 22d29aa..HEAD -- backend/` (commits 966bcff server+/healthz, 1387f40 calculator, 7eae8ae POST /api/v1/calculate handler + Operation.Valid/Unary + main.go wiring; use `git show` on each), and read the final files in full. Check against TASK_SPEC.md and CLAUDE.md. The backend phase should now be complete per the spec (frontend, Docker and README are later phases — don't report their absence).

Context:
- An earlier review of the first two commits already raised these; they're known and still open, so list them in one line each under "Known, still open" only if you agree they still apply — don't re-argue them: (a) percentage computed as a/100*b gives 7%·100 = 7.000000000000001 (fix planned: a*b/100 with fallback on overflow); (b) -0 results encoded as `{"result":-0}`; (c) log.Fatalf in main.go skips deferred cleanup.
- Intentional decisions (critique on merits, don't flag merely for existing): 413 for bodies > 1 MiB though the spec only mentions 400; unknown JSON fields rejected; Content-Type not required; handler does its own method check (mux pattern has no method) so 405 has a JSON body and Allow: POST; non-domain errors → 500 "internal server error" with details only logged; sqrt ignores b.
- Focus on: correctness of handler.go decode/validate/error mapping, the calculator map-based refactor, test quality and gaps (e.g. main.go route wiring is untested — is that worth a test?), CLAUDE.md guardrails (stdlib only, calculator has no HTTP imports, sentinels with errors.Is, Inf/NaN never returned), coverage ≥ 90% for calculator and handler, and anything that would break the README cURL examples or a frontend calling through an nginx /api proxy.
- Commands: `cd backend && gofmt -l . && go vet ./... && go test -count=1 -cover ./...`. `-race` needs cgo which this Windows host lacks; run it via `MSYS_NO_PATHCONV=1 docker run --rm -v "$(pwd -W):/src:ro" -w /src -e GOFLAGS=-buildvcs=false golang:1.27 go test -race -cover ./...` from `backend/`.

Report findings ordered by severity with file:line, and say for each whether you verified it by running code. Don't invent issues; if it's fine, say so briefly.
```

</details>

**Outcome:** no spec violations, and the three known issues were confirmed as still open. New low-severity findings, each confirmed against a running server:
- the route wiring in `main.go` was untested;
- wrong paths return a plain-text 404. That matters for the nginx config: `proxy_pass` must have no trailing slash;
- JSON field names match case-insensitively and duplicate keys are accepted. This is standard `encoding/json` behaviour, to be noted in the README.

## 9. Apply the review fixes

> lets do all the fixes and add the necessary tests for that

**Outcome:** each fix was done test-first. The new cases were added to the existing test files, confirmed failing, then fixed. Commits:
- `fix(backend): avoid rounding error in percentage and normalize -0`
  - Percentage is now `a*b/100`, falling back to `a/100*b` only if `a*b` overflows.
  - Zero results are always returned as `+0`.
  - Tests check exact results for 7%, 29% and 57% of 100, and that the sign of a zero result is positive for every operation.
- `test(backend): cover /api/v1/calculate route wiring`
  - A table test runs the real router (`newMux()`).
  - It was checked by temporarily adding `POST` to the route pattern: the test failed as intended.
- `refactor(backend): return errors from run instead of log.Fatalf`
  - `main` now calls `run(ctx, addr) error`.
  - Tests cover graceful shutdown on cancel and an address already in use.
  - SIGTERM handling was checked with `docker stop`: the server logged "shutting down" and exited with code 0.


## 10. Frontend design and implementation

> Read @TASK_SPEC.md and the handler in @backend/internal/handler to confirm the exact request/response contract.
>
> Propose:
> 1. TypeScript types mirroring the contract
> 2. useCalculator: state shape, return type, and every transition (digit, decimal, operator, equals, sqrt, percent, clear, backspace, error → next input clears it). Input model: immediate execution, left to right, one backend call per operation.
> 3. Keypad layout and how unary ops (√, %) behave
> 4. History: last 10 successful calculations, newest first, client-side
> 5. Accessibility: keyboard input, aria-live on the display, labeled buttons

**Clarifying questions answered:**
- `%` is contextual. With an operator pending, `x%` is x percent of the left operand (`200 + 10 % =` → 220). Otherwise it is `x / 100`.
- Add a local `±` key, so negative operands (and the negative-sqrt error) can be entered directly.

**Outcome:** the plan was approved and implemented test-first, one commit per step:
- `feat(frontend): add API contract types and calculate service`
  - `calculate()` never throws for HTTP or network errors; it narrows the `unknown` body with type guards.
  - Only an abort is rethrown.
- `feat(frontend): add number formatting and keyboard mapping helpers`
- `feat(frontend): add useCalculator hook with immediate-execution state machine`
  - A pure reducer with entry modes `typing | result | awaiting`. A single `overwrite` flag couldn't tell "operator just pressed" apart from "unary result showing".
  - The in-flight request lives in state and an effect sends it, so AC and unmount abort it.
  - Every transition is covered by table-driven key-sequence tests.
- `feat(frontend): add Display, Keypad and History components`
- `feat(frontend): wire Calculator with keyboard input into App`
- `style(frontend): add responsive calculator layout with focus and dark mode styles`

## 11. Frontend review (code-reviewer subagent)

> Its all working correctly on my local test. Use the code-reviewer subagent on the frontend changes

The subagent got the commit range, the backend contract and the intentional design decisions. It was asked to focus on reducer edge cases, StrictMode, the CLAUDE.md guardrails, accessibility and weak tests.

**Outcome:** no spec or guardrail violations, and all checks pass. Findings:
- **Medium:** a chained operation uses the raw result (`0.30000000000000004`) where `=` uses the displayed value (`0.3`).
- **Medium (a11y):** disabling the keypad while a request is in flight drops keyboard focus to `<body>`.
- **Medium (test):** the "Enter on a focused key" test still passes without `preventDefault`, so it doesn't test what it claims.
- **Low–medium (a11y):** Enter always means `=`, so it can never activate the focused button.
- **Low:** the 16-digit entry cap is beyond double precision; it should be 15.
- **Low:** √ and % right after an operator act on the left operand, and no test covers this.
- **Missing tests:**
  - the hook→service seam, tested with a stubbed `fetch`;
  - operator → ± → ⌫.

## 12. Apply the frontend review fixes

> Lets to all the fixes, update tests that needed

**Outcome:** each fix was done test-first: the test was written or updated, seen failing, then the fix applied. Commits:
- `fix(frontend): chain operations on the displayed result`
  - The chained operand is now `Number(formatResult(result))`.
  - Test: `0.1 + 0.2 + 1 =` sends `add(0.3, 1)`.
- `fix(frontend): cap entry at 15 digits so typed values are sent exactly`
  - The old 16-digit expectation was changed to 15.
  - New test: a 15-digit entry is sent unchanged.
- `fix(frontend): use aria-disabled on keys so focus survives a calculation`
  - The Keypad ignores clicks on inactive keys, and the CSS selectors now target `[aria-disabled='true']`.
  - A planned "keeps focus" test was dropped: jsdom never moves focus off disabled buttons, so that test could never fail.
- `fix(frontend): let Enter activate the focused key instead of always pressing equals`
  - The old Enter test was replaced, because it encoded the old behaviour and passed without `preventDefault`.
  - The new `fireEvent` return-value test was checked by deleting `preventDefault`: it failed as intended.
- `fix(frontend): detect aborts by name rather than DOMException instance`
- `test(frontend): cover unary keys after an operator, sign-then-backspace and the service seam`
  - A new `Calculator.integration.test.tsx` runs the real service against a stubbed `fetch`.
  - It was checked by temporarily breaking the endpoint URL: it failed as intended.

## 13. Add coverage tooling and commit pending files

> Please add the coverage tool and then commit the files

**Outcome:**
- Added `@vitest/coverage-v8@5.0.3` as a dev dependency, pinned to the installed vitest version, which it requires as a peer.
- `npm run test:coverage` gives 98.7% statements, 98.5% branches, and 100% of functions and lines for `hooks/` and `services/`.
- The pending root `package.json`, `CLAUDE.md` and `PROMPTS.md` edits were committed separately.

## 14. Second frontend review and final adjustments (code-reviewer subagent)

> Now lets review everything again using the sub agent and do the final adjustments

The subagent was asked to verify the first round of fixes, look for regressions, check that aria-disabled keys leak nothing (mouse or keyboard), check the Enter-on-button rule, look for tests that pass for the wrong reason, and confirm the coverage provider's version matches vitest.

**Review outcome:** no spec or guardrail violations, all first-round fixes confirmed correct, no wrong-reason tests, and the coverage provider matches vitest. Findings:
- **Medium:** Chrome and Firefox focus a button on mouse click, so clicking `2 + 3` and then pressing Enter gave 33 instead of 5.
- **Low:** two abort branches had no test.

**Fixes:**
- `fix(frontend): keep mouse clicks from focusing keys so Enter means equals`
  - Keys call `preventDefault` on `mousedown`; Tab focus is unaffected.
  - The new test failed first. The old focused-digit test now focuses the key the way the keyboard would.
- `refactor(frontend): drop unreachable abort guard and test the stale-request path`
  - The hook's `signal.aborted` guard was unreachable: after AC the reducer drops the result as stale, and after unmount `dispatch` does nothing. Any test of it would have passed for the wrong reason, so it was removed.
  - A new test covers a rejected, aborted request after AC. It was checked by disabling the reducer's stale-id check: the test failed as intended.
- `test(frontend): cover an abort while reading the response body`
  - Checked by deleting the rethrow: the test failed as intended.

Final state: 206 tests, coverage of `hooks/` and `services/` at 100% statements, branches, functions and lines, and typecheck, lint and build clean.

## 15. Docker multi-stage setup

> So lets move to docker multi stage config

A plan was proposed: a Go build stage, an optional `-race` test stage and a distroless runtime; a Node build stage and an unprivileged nginx runtime; an nginx `/api` proxy; compose. It came with one question, how to health-check a backend image that has no shell.

> lets move with the -healthcheck flag approach so we have an easy way to check the service health

**Outcome:**
- `feat(backend): add -healthcheck flag for container health checks`
  - `checkHealth(ctx, url)` and `healthcheckURL()` were written test-first: the real `/healthz`, 500, 404, an unreachable server, a timeout, and the `PORT` override.
  - `/api -healthcheck` exits 0 or 1.
- `feat(docker): add multi-stage Dockerfiles, nginx proxy and compose`
  - **Backend:** `golang:1.27` builds the binary, an optional `test` stage runs `go test -race`, and the runtime is `distroless/static:nonroot`. The image is 16.5 MB, with a `HEALTHCHECK` that uses the flag.
  - **Frontend:** `node:24-alpine` runs `npm ci` and the build, and `nginx-unprivileged:1.30-alpine` serves it as uid 101 on port 8080.
  - **nginx:** proxies `/api/` with the path unchanged and returns JSON envelopes for its own 413 and 502/503/504 errors. It re-resolves `backend` through Docker DNS, sends `no-cache` for `index.html`, caches hashed assets as immutable, falls back to the app for client-side routes, uses gzip, and sets `server_tokens off`.
  - **Compose:** the backend is on `8080:8080`, the frontend on `3000:8080`, and the frontend waits for `service_healthy`.
- `build: add make target for race tests in Docker`

**Verified** with the stack on ports 18080/13000 through a scratch override file, because the developer's own dev servers were using 8080 and 3000:
- **API:** 200, 400 (divide by zero, negative sqrt, invalid body) and 405 all match, both directly and through the proxy.
- **Body limit:** a 1.1 MB body gets a JSON 413 at both layers.
- **Backend down:** returns a JSON 503.
- **Backend recreated:** after being forced onto a new IP, the proxy still reaches it.
- **Health:** the health check works, and with a wrong `PORT` the flag exits 1.
- **Users:** both containers run as non-root.
- **Shutdown:** a SIGTERM stop logs "shutting down" and exits 0.
- **Tests:** `docker build --target test backend` passes with `-race`.

Two changes came out of verification:
- the JSON error pages, because nginx's own 413 was an HTML page;
- the DNS re-resolution, because nginx caches the backend IP from startup.

## 16. README and final checks

> Lets work on the readme now and do last checkings

**Outcome:** `README.md` was written from the actual code and configuration. It covers:
- quick start with Docker;
- local Go and Node setup;
- the API reference, with every status code and error message;
- 14 cURL examples with their expected output;
- the calculator's keys and keyboard shortcuts;
- test and coverage commands;
- design decisions for the backend, frontend and Docker;
- assumptions and the project layout.

**Checks:**
- **cURL examples:** a script ran every README example exactly as written, with only the ports swapped for the Docker stack on 18080/13000. It compared status and body against the README's `# <status> <body>` lines: 14/14 match.
- **Health check:** `docker compose exec backend /api -healthcheck` fails in Git Bash, which rewrites `/api` into a Windows path. The README now documents the `MSYS_NO_PATHCONV=1` workaround, which was verified.
- **Coverage claim:** checked before stating it. 9 frontend files are fully covered (100% statements and branches), and the 1 skipped file is type-only.
- **Definition of done:**
  - Backend: gofmt and go vet are clean. `go test` passes, with `internal/calculator` at 100% and `internal/handler` at 98.2%. `-race` passes in the Docker test stage, forced with `--no-cache-filter`.
  - Frontend: tsc, ESLint and the build are clean, and 206 tests pass.
  - Docker: the compose stack is healthy, and the UI and `/api` proxy work.
