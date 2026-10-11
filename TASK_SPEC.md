# Full Task Specification: Sezzle Calculator Assessment

## 1. Functional Requirements
- **Core Operations:** Addition (`+`), Subtraction (`-`), Multiplication (`*`), Division (`/`)
- **Advanced Operations:** Exponentiation (`^`), Square Root (`sqrt`), Percentage (`%`)
- **Frontend UI (React + TypeScript):**
    - Intuitive calculator display and keypad
    - Clear rendering of calculation results and active state
    - Display error alerts returned by backend
    - Calculation history log (last 10 operations)
    - Responsive design (basic mobile support)
- **Backend Service (Go REST API):**
    - Endpoint: `POST /api/v1/calculate`
    - Input validation for JSON payloads
    - Graceful edge-case handling (Division by Zero, negative square root, invalid parameters)
    - Structured error envelopes with HTTP 400 status (`{"error": "message"}`)
- **JSON Contract**
  ```// request
       {"operation": "divide", "a": 10, "b": 2}
       // 200
       {"result": 5}
       // 400
       {"error": "division by zero is not allowed"}
  
## 2. Non-Functional & Code Quality Criteria
- Idiomatic Go using standard library `net/http` without heavy web frameworks
- Strict TypeScript types (zero `any` usage)
- Decoupled React state (all calculation state encapsulated inside `useCalculator.ts`)
- High test coverage with Go table-driven unit tests and React component unit tests

## 3. Explicit Deliverables Required by Sezzle
1. **Git Repository:** Clean commit history with source code for frontend and backend
2. **`README.md`:** Must include:
    - Local setup instructions (Go and Node)
    - How to run frontend and backend
    - Example REST API calls (cURL / JSON requests)
    - Architectural design decisions and assumptions
3. **`PROMPTS.md`:** Detailed record of all AI prompts used during development
4. **Unit Tests & Coverage:** Runnable test suite with coverage reporting commands
5. **Docker:** Multi-stage Dockerfiles and `docker-compose.yml` for single-command launch