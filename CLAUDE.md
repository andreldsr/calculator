# Sezzle Calculator — Claude Code Guidelines

Full-stack calculator: Go REST API (`backend/`) + React/TypeScript (`frontend/`), run together with Docker Compose.

Requirements and API contract: @TASK_SPEC.md — that file is the source of truth. If this file and the spec disagree, follow the spec and flag the conflict.

## Commands
| Purpose | Command |
|---|---|
| Backend tests | `cd backend && go test -race -cover ./...` |
| Backend lint | `cd backend && gofmt -l . && go vet ./...` |
| Frontend tests | `cd frontend && npm test -- --run` |
| Frontend typecheck | `cd frontend && npx tsc --noEmit` |
| Frontend lint | `cd frontend && npm run lint` |
| Frontend build | `cd frontend && npm run build` |
| Everything | `make test` |
| Full stack | `docker compose up --build` (UI :3000, API :8080) |

## Workflow rules
- Work one phase at a time: backend → frontend → Docker/docs. Never scaffold everything in one pass.
- Before coding a phase, propose a plan (files to create, approach) and wait for approval.
- Test-first: write table-driven tests, confirm they fail, then implement. Never weaken, skip, or delete a test to make it pass; if a test seems wrong, stop and ask.
- Run the relevant test, lint and typecheck commands before saying a step is done. Do not commit if any of them fails.
- Small commits with conventional messages (`feat(backend): add division with zero check`).
- Ask before adding any dependency (Go module or npm package).
- Do not write or edit `PROMPTS.md`; the developer maintains it.

## Architecture guardrails
- **Backend:** standard library only (`net/http`, `encoding/json`); no Gin/Echo/Fiber. Math lives in `internal/calculator` with no HTTP imports; `internal/handler` only decodes, validates, calls the calculator and encodes.
- **Errors:** domain errors are sentinel values (`ErrDivisionByZero`, `ErrNegativeSqrt`, ...) checked with `errors.Is`. Handler maps them to `400 {"error": "message"}`; wrong method → `405`; Inf/NaN results are errors, never returned as numbers.
- **Frontend:** components are presentational (props in, callbacks out). All calculator and API state lives in `hooks/useCalculator.ts`; all `fetch` calls live in `services/`. Shared types live in `types/` and mirror the API contract exactly.
- **TypeScript:** `strict: true`; no `any`, no non-null `!` assertions, no `@ts-ignore`.

## Repository layout
```
sezzle-calculator/
├── backend/
│   ├── cmd/api/main.go                 # wiring, server, graceful shutdown
│   ├── internal/calculator/            # calculator.go, calculator_test.go
│   ├── internal/handler/               # handler.go, handler_test.go (httptest)
│   ├── Dockerfile
│   └── go.mod                          # module github.com/andre/sezzle-calculator/backend
├── frontend/
│   ├── src/components/                 # Calculator, Display, Keypad, History (+ *.test.tsx)
│   ├── src/hooks/useCalculator.ts      # + useCalculator.test.ts
│   ├── src/services/api.ts
│   ├── src/types/calculator.ts
│   ├── nginx.conf                      # serves build, proxies /api → backend
│   └── Dockerfile
├── docker-compose.yml
├── Makefile
├── README.md
└── PROMPTS.md
```

## Definition of done
- [ ] `make test` passes: Go tests with `-race`, frontend tests, typecheck, lint.
- [ ] Go coverage ≥ 90% for `internal/calculator` and `internal/handler`.
- [ ] `docker compose up --build` starts both services; UI calculates through the `/api` proxy.
- [ ] The cURL examples in README.md run as written, including the error cases.
- [ ] README covers setup, run, API examples, design decisions and assumptions.