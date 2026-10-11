# Sezzle Calculator

A full-stack calculator: a Go REST API that does the math and a React + TypeScript UI that drives it. Every operation is calculated by the backend, and the frontend renders results, errors and a history of the last 10 calculations.

- **Backend:** Go 1.27, standard library only (`net/http`, `encoding/json`).
- **Frontend:** React 19, TypeScript (strict), Vite, Vitest + React Testing Library.
- **Packaging:** multi-stage Dockerfiles, nginx reverse proxy, Docker Compose.

Operations: addition, subtraction, multiplication, division, exponentiation, square root and percentage.

## Contents

- [Quick start (Docker)](#quick-start-docker)
- [Local setup](#local-setup)
- [API](#api)
- [Using the calculator](#using-the-calculator)
- [Tests and coverage](#tests-and-coverage)
- [Design decisions](#design-decisions)
- [Assumptions](#assumptions)
- [Project layout](#project-layout)

## Quick start (Docker)

Requires Docker with Compose v2.

```bash
docker compose up --build
```

| Service | URL | Notes |
|---|---|---|
| UI | http://localhost:3000 | nginx serves the app and proxies `/api` to the backend |
| API | http://localhost:8080 | published directly so you can call it with cURL |

The UI container starts once the backend reports healthy. Stop everything with `docker compose down`.

## Local setup

### Prerequisites

- Go 1.27 or newer
- Node.js 24 (with npm)
- Optional: GNU `make`, for the shortcuts below

### Backend

```bash
cd backend
go run ./cmd/api          # listens on :8080; set PORT to change it
```

### Frontend

```bash
cd frontend
npm ci
npm run dev               # http://localhost:3000
```

The Vite dev server proxies `/api` to `http://localhost:8080`. Point it somewhere else with `VITE_API_PROXY_TARGET`, e.g. `VITE_API_PROXY_TARGET=http://localhost:9090 npm run dev`.

Production build: `npm run build` (typechecks, then writes `frontend/dist`).

## API

### `POST /api/v1/calculate`

Request body:

```json
{"operation": "divide", "a": 10, "b": 2}
```

| `operation` | Result | Needs `b` |
|---|---|---|
| `add` | a + b | yes |
| `subtract` | a − b | yes |
| `multiply` | a × b | yes |
| `divide` | a ÷ b | yes |
| `power` | a ^ b | yes |
| `sqrt` | √a | no (ignored if sent) |
| `percentage` | a% of b, i.e. a × b ÷ 100 | yes |

Responses always have `Content-Type: application/json`.

| Status | Body | When |
|---|---|---|
| 200 | `{"result": 5}` | success |
| 400 | `{"error": "..."}` | invalid JSON, missing or unknown fields, unsupported operation, domain errors (division by zero, negative square root, non-finite result) |
| 405 | `{"error": "method not allowed"}` | any method other than `POST` (with an `Allow: POST` header) |
| 413 | `{"error": "request body too large"}` | body over 1 MiB |
| 500 | `{"error": "internal server error"}` | unexpected failure; details are only logged |

Error messages:

| Message | Cause |
|---|---|
| `division by zero is not allowed` | `divide` with `b` = 0 |
| `square root of a negative number is not allowed` | `sqrt` with `a` < 0 |
| `result is not a finite number` | the result overflows, e.g. `10 ^ 400` |
| `unsupported operation` | `operation` is not in the table above |
| `operation is required` / `a is required` / `b is required` | a required field is missing or `null` |
| `invalid request body` | malformed JSON, wrong field types, unknown fields, or extra data after the object |

### `GET /healthz`

Returns `{"status":"ok"}`. The backend binary can also check itself, which is what the Docker health check uses, because the runtime image has no shell or curl:

```bash
docker compose exec backend /api -healthcheck   # exit code 0 = healthy, 1 = unhealthy
```

In Git Bash on Windows, prefix it with `MSYS_NO_PATHCONV=1`, which stops Git Bash from rewriting `/api` into a Windows path.

### cURL examples

These are written for a POSIX shell (bash, zsh, Git Bash). The backend must be running on `:8080`, either with Docker Compose or `go run`.

Success:

```bash
curl -s -X POST http://localhost:8080/api/v1/calculate -H 'Content-Type: application/json' -d '{"operation": "divide", "a": 10, "b": 2}'
# 200 {"result":5}

curl -s -X POST http://localhost:8080/api/v1/calculate -H 'Content-Type: application/json' -d '{"operation": "add", "a": 2.5, "b": -0.5}'
# 200 {"result":2}

curl -s -X POST http://localhost:8080/api/v1/calculate -H 'Content-Type: application/json' -d '{"operation": "power", "a": 2, "b": 10}'
# 200 {"result":1024}

curl -s -X POST http://localhost:8080/api/v1/calculate -H 'Content-Type: application/json' -d '{"operation": "sqrt", "a": 16}'
# 200 {"result":4}

curl -s -X POST http://localhost:8080/api/v1/calculate -H 'Content-Type: application/json' -d '{"operation": "percentage", "a": 15, "b": 200}'
# 200 {"result":30}
```

Errors:

```bash
curl -s -X POST http://localhost:8080/api/v1/calculate -H 'Content-Type: application/json' -d '{"operation": "divide", "a": 10, "b": 0}'
# 400 {"error":"division by zero is not allowed"}

curl -s -X POST http://localhost:8080/api/v1/calculate -H 'Content-Type: application/json' -d '{"operation": "sqrt", "a": -4}'
# 400 {"error":"square root of a negative number is not allowed"}

curl -s -X POST http://localhost:8080/api/v1/calculate -H 'Content-Type: application/json' -d '{"operation": "power", "a": 10, "b": 400}'
# 400 {"error":"result is not a finite number"}

curl -s -X POST http://localhost:8080/api/v1/calculate -H 'Content-Type: application/json' -d '{"operation": "modulo", "a": 10, "b": 3}'
# 400 {"error":"unsupported operation"}

curl -s -X POST http://localhost:8080/api/v1/calculate -H 'Content-Type: application/json' -d '{"operation": "add", "a": 1}'
# 400 {"error":"b is required"}

curl -s -X POST http://localhost:8080/api/v1/calculate -H 'Content-Type: application/json' -d '{"operation": "add", "a": "1", "b": 2}'
# 400 {"error":"invalid request body"}

curl -s -X POST http://localhost:8080/api/v1/calculate -H 'Content-Type: application/json' -d '{"operation": "add", "a": 1, "b": 2, "c": 3}'
# 400 {"error":"invalid request body"}

curl -s -X GET http://localhost:8080/api/v1/calculate
# 405 {"error":"method not allowed"}
```

The same API through the UI's nginx proxy (Docker Compose only):

```bash
curl -s -X POST http://localhost:3000/api/v1/calculate -H 'Content-Type: application/json' -d '{"operation": "multiply", "a": 6, "b": 7}'
# 200 {"result":42}
```

Add `-i` to any command to see the status line and headers.

## Using the calculator

The calculator uses **immediate execution, left to right**, like a basic handheld calculator. Each operation is one backend call. `2 + 3 × 4 =` gives 20, not 14: pressing `×` first calculates `2 + 3`.

| Key | Keyboard | Behaviour |
|---|---|---|
| `0`–`9`, `.` | `0`–`9`, `.` or `,` | Type a number (up to 15 digits) |
| `+ − × ÷` | `+ - * /` (`x` also works for ×) | Calculate any pending operation, then wait for the next number |
| `xʸ` | `^` | Power |
| `=` | `Enter` or `=` | Calculate the pending operation |
| `√` | `r` | Square root of the number shown, right away |
| `%` | `%` | Percentage of the number shown, right away (see below) |
| `±` | `n` | Toggle the sign of the number being typed (no backend call) |
| `⌫` | `Backspace` | Delete the last typed digit |
| `AC` | `Escape` or `Delete` | Clear everything except history; also cancels a request in flight |

Details:

- **`%` depends on context.** With an operator pending, `x%` means x percent of the left operand, so `200 + 10 % =` gives 220. With nothing pending, `x%` means x ÷ 100.
- **`√` and `%` keep a pending operator.** `9 + 16 √ =` gives 13. Pressed directly after an operator, they act on the number still on the display: `9 + √ =` is `9 + √9 = 12`.
- **Errors** returned by the backend show in an alert under the display. The next key press clears the error and starts fresh; history is kept.
- **History** shows the last 10 successful calculations, newest first. It lives in memory, so a page reload clears it.
- **Enter** presses `=`, except when a key has keyboard focus (via Tab), where it presses that key, as buttons normally do.

## Tests and coverage

| What | Command |
|---|---|
| Everything (lint, typecheck, tests) | `make test` |
| Backend tests | `cd backend && go test -race -cover ./...` |
| Backend lint | `cd backend && gofmt -l . && go vet ./...` |
| Backend coverage report | `cd backend && go test -coverprofile=coverage.out ./... && go tool cover -func=coverage.out` |
| Backend race tests in Docker | `make test-backend-docker` (or `docker build --target test backend`) |
| Frontend tests | `cd frontend && npm test -- --run` |
| Frontend coverage report | `cd frontend && npm run test:coverage` |
| Frontend typecheck | `cd frontend && npx tsc -b --noEmit` |
| Frontend lint | `cd frontend && npm run lint` |
| Both coverage reports | `make coverage` |

`go test -race` needs cgo and a C compiler. On a machine without one, such as a plain Windows install, drop `-race` or use the Docker target, which runs the race detector inside the `golang` image.

Current coverage:

| Package | Statements |
|---|---|
| `backend/internal/calculator` | 100% |
| `backend/internal/handler` | 98.2% |
| `frontend/src` (hooks, services, components) | 100% statements and branches |

Backend tests are table-driven: the calculator in isolation, the handler through `httptest`, and the real router in `cmd/api`. Frontend tests cover the reducer through key-sequence tables, the API service against a stubbed `fetch`, each component with React Testing Library, and an integration test that runs the real service end to end.

## Design decisions

### Backend

- **Standard library only.** Go's `net/http` method-aware routing and `encoding/json` cover everything this API needs, with no framework.
- **Layers.**
  - `internal/calculator` is pure math with no HTTP imports.
  - `internal/handler` only decodes, validates, calls the calculator and encodes.
  - `cmd/api` wires the routes and runs the server.
  - The handler takes the calculate function as a dependency, so tests can stub it.
- **Sentinel errors.**
  - Domain errors (`ErrDivisionByZero`, `ErrNegativeSqrt`, ...) are package-level values matched with `errors.Is`, so wrapped errors still map to the right response.
  - The handler maps them to 400 and their message.
  - Anything unexpected becomes a 500 with a generic message, and the details are only logged.
- **Strict input.**
  - Fields are decoded into pointers, so a missing `b` is distinguishable from `b: 0`.
  - Unknown fields and trailing data after the JSON object are rejected.
  - Bodies are capped at 1 MiB with `http.MaxBytesReader`.
- **No Inf or NaN in responses.** JSON can't represent them, so a non-finite result is a 400 (`result is not a finite number`), never a number. A `-0` result is returned as `0`.
- **Percentage precision.** It is computed as `a * b / 100` so that common cases are exact (7% of 100 is 7, not 7.000000000000001). It falls back to `a / 100 * b` only when `a * b` would overflow.
- **JSON 405.** The route pattern deliberately has no method, so the handler answers wrong methods itself with a JSON body and an `Allow` header. Go's mux would otherwise answer in plain text.
- **Operational details.**
  - Server timeouts are set, and the server shuts down gracefully on SIGTERM or Ctrl-C (10 s drain).
  - `PORT` is configurable.
  - The `-healthcheck` flag lets the shell-less image check itself.

### Frontend

- **State in one hook.**
  - `useCalculator` holds all calculator and API state in a `useReducer` with a pure, exported reducer, so the transitions are unit-testable without React.
  - Components are presentational: props in, callbacks out.
  - `fetch` lives only in `services/api.ts`.
- **Requests are state.**
  - A key that needs the backend stores the request in state, and an effect sends it.
  - AC or unmounting aborts it with an `AbortController`.
  - Responses for a request that is no longer current are ignored by id.
  - While a request is in flight only AC is accepted, so there are no double submissions.
- **The service never throws for HTTP errors.**
  - `calculate()` returns `{ok: true, result} | {ok: false, error}`.
  - It narrows the `unknown` response body with type guards, with no casts.
  - It falls back to a readable message for non-JSON responses (e.g. a proxy error page) and network failures.
- **Display precision.**
  - Results are shown with 15 significant digits, so `0.1 + 0.2` shows `0.3`.
  - The next operation uses the displayed value, so what you see is what gets calculated.
  - Typed input is capped at 15 digits, the most a double holds exactly.
- **Types mirror the contract.** `types/calculator.ts` defines `Operation`, `CalculateRequest`, and the success and error bodies exactly as the API does.
- **Accessibility.**
  - Every key is a real `<button>` with a spoken label ("divided by", "square root").
  - The pending operator has `aria-pressed`.
  - The display is an `aria-live` region, and errors use `role="alert"`.
  - Full keyboard support.
  - Keys are `aria-disabled` rather than `disabled` while loading, so focus isn't lost.
  - Mouse clicks don't move focus onto keys, so Enter keeps meaning `=`.
- **Responsive.** The layout is mobile-first and puts history beside the keypad from 768 px. It has dark mode via `prefers-color-scheme`, and keys are at least 44 px tall.

### Docker

- **Multi-stage builds.**
  - The backend builds a static binary and runs on `distroless/static:nonroot`, a 16.5 MB image with no shell.
  - The frontend builds with Node and is served by `nginx-unprivileged`.
  - Both containers run as non-root users.
- **nginx as the single entry point for the UI.**
  - It proxies `/api/` with the path unchanged, falls back to the app for client-side routes, caches hashed assets as immutable, and sends `no-cache` for `index.html`.
  - It returns the same JSON error envelope for errors it raises itself (413, and 503 when the backend is down).
  - It re-resolves the backend through Docker's DNS, so a recreated backend container is picked up.
- **Health-gated startup.** Compose starts the UI only after the backend's health check passes.

## Assumptions

- **Percentage** in the API is binary: "a percent of b". The UI's `%` key maps onto it as described in [Using the calculator](#using-the-calculator).
- **Square root** uses only `a`. A `b` sent with it is ignored rather than rejected.
- **Numbers** are IEEE-754 doubles on both sides, so results carry normal floating-point limits. The UI hides representation noise beyond 15 significant digits.
- **JSON parsing** follows Go's `encoding/json`. Field names match case-insensitively (`"A"` works as `"a"`), and with duplicate keys the last one wins.
- **Unknown paths** get Go's default plain-text 404. Only `/api/v1/calculate` and `/healthz` are part of the API.
- **History** is client-side and in memory, per browser tab. It is not persisted or shared.
- **Repeated `=`** does not repeat the last operation, and `5 + =` does nothing until a second number is entered.
- **Out of scope:** authentication, rate limiting and CORS (the UI reaches the API through the same-origin proxy).

## Project layout

```
.
├── backend/
│   ├── cmd/api/                 # main.go: routes, server, graceful shutdown, -healthcheck
│   ├── internal/calculator/     # pure math + sentinel errors
│   ├── internal/handler/        # HTTP: decode, validate, call, encode
│   └── Dockerfile               # build → test (optional, -race) → distroless runtime
├── frontend/
│   ├── src/components/          # Calculator, Display, Keypad, History (+ tests)
│   ├── src/hooks/               # useCalculator (reducer + effect), keyboard mapping
│   ├── src/services/api.ts      # the only place that calls fetch
│   ├── src/types/calculator.ts  # API contract and UI types
│   ├── src/utils/format.ts      # number and expression formatting
│   ├── nginx.conf               # static files + /api proxy
│   └── Dockerfile               # node build → nginx-unprivileged runtime
├── docker-compose.yml
├── Makefile
└── PROMPTS.md                   # every AI prompt used to build this project
```

## AI usage

This project was built with Claude Code. [`PROMPTS.md`](PROMPTS.md) records every prompt, the subagent briefs, and what each step produced.
