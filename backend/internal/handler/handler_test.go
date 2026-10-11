package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/andre/sezzle-calculator/backend/internal/calculator"
)

const (
	endpoint = "/api/v1/calculate"
	maxBody  = 1 << 20 // 1 MiB
)

// Compile-time check that the real calculator satisfies CalculateFunc.
var _ CalculateFunc = calculator.Calculate

// stubCalc records calls and returns a fixed result/error.
type stubCalc struct {
	calls  int
	gotOp  calculator.Operation
	gotA   float64
	gotB   float64
	result float64
	err    error
}

func (s *stubCalc) fn(op calculator.Operation, a, b float64) (float64, error) {
	s.calls++
	s.gotOp, s.gotA, s.gotB = op, a, b
	return s.result, s.err
}

func do(t *testing.T, h http.Handler, method, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, endpoint, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func assertJSONContentType(t *testing.T, rec *httptest.ResponseRecorder) {
	t.Helper()
	ct := rec.Header().Get("Content-Type")
	if !strings.HasPrefix(ct, "application/json") {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}
}

// decodeExact decodes the body and asserts it has exactly one key, wantKey.
func decodeExact(t *testing.T, rec *httptest.ResponseRecorder, wantKey string) json.RawMessage {
	t.Helper()
	var m map[string]json.RawMessage
	if err := json.Unmarshal(rec.Body.Bytes(), &m); err != nil {
		t.Fatalf("response body is not a JSON object: %v; body=%q", err, rec.Body.String())
	}
	if len(m) != 1 {
		t.Fatalf("response body keys = %v, want exactly [%q]; body=%q", keys(m), wantKey, rec.Body.String())
	}
	raw, ok := m[wantKey]
	if !ok {
		t.Fatalf("response body missing key %q; body=%q", wantKey, rec.Body.String())
	}
	return raw
}

func keys(m map[string]json.RawMessage) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

func assertResult(t *testing.T, rec *httptest.ResponseRecorder, want float64) {
	t.Helper()
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body=%q", rec.Code, http.StatusOK, rec.Body.String())
	}
	assertJSONContentType(t, rec)
	raw := decodeExact(t, rec, "result")
	var got float64
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("result is not a number: %v; raw=%s", err, raw)
	}
	if got != want {
		t.Errorf("result = %v, want %v", got, want)
	}
}

func assertError(t *testing.T, rec *httptest.ResponseRecorder, wantStatus int, wantMsg string) {
	t.Helper()
	if rec.Code != wantStatus {
		t.Fatalf("status = %d, want %d; body=%q", rec.Code, wantStatus, rec.Body.String())
	}
	assertJSONContentType(t, rec)
	raw := decodeExact(t, rec, "error")
	var got string
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("error is not a string: %v; raw=%s", err, raw)
	}
	if got != wantMsg {
		t.Errorf("error = %q, want %q", got, wantMsg)
	}
}

func TestCalculate_Success_RealCalculator(t *testing.T) {
	tests := []struct {
		name string
		body string
		want float64
	}{
		{"spec example divide", `{"operation":"divide","a":10,"b":2}`, 5},
		{"add", `{"operation":"add","a":2,"b":3}`, 5},
		{"add with b zero is present", `{"operation":"add","a":5,"b":0}`, 5},
		{"add negatives", `{"operation":"add","a":-2.5,"b":-1.5}`, -4},
		{"subtract", `{"operation":"subtract","a":10,"b":4}`, 6},
		{"subtract to negative", `{"operation":"subtract","a":1,"b":3.5}`, -2.5},
		{"multiply", `{"operation":"multiply","a":6,"b":7}`, 42},
		{"multiply fractional and negative", `{"operation":"multiply","a":-0.5,"b":8}`, -4},
		{"multiply by zero", `{"operation":"multiply","a":123,"b":0}`, 0},
		{"divide fractional result", `{"operation":"divide","a":1,"b":4}`, 0.25},
		{"divide zero numerator", `{"operation":"divide","a":0,"b":5}`, 0},
		{"power", `{"operation":"power","a":2,"b":10}`, 1024},
		{"power zero exponent", `{"operation":"power","a":7,"b":0}`, 1},
		{"power negative exponent", `{"operation":"power","a":2,"b":-2}`, 0.25},
		{"sqrt without b", `{"operation":"sqrt","a":16}`, 4},
		{"sqrt with b ignored", `{"operation":"sqrt","a":16,"b":99}`, 4},
		{"sqrt with b null", `{"operation":"sqrt","a":9,"b":null}`, 3},
		{"sqrt fractional", `{"operation":"sqrt","a":0.25}`, 0.5},
		{"sqrt zero", `{"operation":"sqrt","a":0}`, 0},
		{"percentage 50 of 200", `{"operation":"percentage","a":50,"b":200}`, 100},
		{"field order irrelevant", `{"b":2,"a":10,"operation":"divide"}`, 5},
		{"surrounding whitespace allowed", " \n{\"operation\":\"add\",\"a\":1,\"b\":2}\n", 3},
		{"exponent notation", `{"operation":"multiply","a":1e2,"b":2}`, 200},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := do(t, New(calculator.Calculate), http.MethodPost, tt.body)
			assertResult(t, rec, tt.want)
		})
	}
}

func TestCalculate_DomainErrors_RealCalculator(t *testing.T) {
	tests := []struct {
		name    string
		body    string
		wantMsg string
	}{
		{"divide by zero (spec contract)", `{"operation":"divide","a":10,"b":0}`, "division by zero is not allowed"},
		{"divide zero by zero", `{"operation":"divide","a":0,"b":0}`, "division by zero is not allowed"},
		{"sqrt negative", `{"operation":"sqrt","a":-4}`, calculator.ErrNegativeSqrt.Error()},
		{"power overflow", `{"operation":"power","a":10,"b":309}`, calculator.ErrNonFiniteResult.Error()},
		{"multiply overflow", `{"operation":"multiply","a":1e308,"b":10}`, calculator.ErrNonFiniteResult.Error()},
		{"power NaN (negative base fractional exponent)", `{"operation":"power","a":-8,"b":0.5}`, calculator.ErrNonFiniteResult.Error()},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := do(t, New(calculator.Calculate), http.MethodPost, tt.body)
			assertError(t, rec, http.StatusBadRequest, tt.wantMsg)
		})
	}
}

func TestCalculate_ValidationErrors(t *testing.T) {
	const (
		invalidBody = "invalid request body"
		opRequired  = "operation is required"
		unsupported = "unsupported operation"
		aRequired   = "a is required"
		bRequired   = "b is required"
	)
	tests := []struct {
		name    string
		body    string
		wantMsg string
	}{
		// Body decoding.
		{"empty body", ``, invalidBody},
		{"whitespace only body", "   \n", invalidBody},
		{"malformed JSON", `{"operation":"add","a":1,`, invalidBody},
		{"not JSON", `hello`, invalidBody},
		{"a as string", `{"operation":"add","a":"10","b":2}`, invalidBody},
		{"b as string", `{"operation":"add","a":10,"b":"2"}`, invalidBody},
		{"a as bool", `{"operation":"add","a":true,"b":2}`, invalidBody},
		{"operation as number", `{"operation":5,"a":1,"b":2}`, invalidBody},
		{"unknown field", `{"operation":"add","a":1,"b":2,"c":3}`, invalidBody},
		{"trailing object", `{"operation":"add","a":1,"b":2}{}`, invalidBody},
		{"trailing garbage", `{"operation":"add","a":1,"b":2} x`, invalidBody},
		{"a out of float64 range", `{"operation":"add","a":1e400,"b":1}`, invalidBody},
		{"b out of float64 range", `{"operation":"add","a":1,"b":-1e400}`, invalidBody},
		{"top-level array", `[{"operation":"add","a":1,"b":2}]`, invalidBody},
		{"top-level string", `"add"`, invalidBody},
		{"top-level number", `42`, invalidBody},

		// Operation.
		{"operation missing", `{"a":1,"b":2}`, opRequired},
		{"operation empty", `{"operation":"","a":1,"b":2}`, opRequired},
		{"operation null", `{"operation":null,"a":1,"b":2}`, opRequired},
		{"operation missing and a missing", `{"b":2}`, opRequired},
		{"unsupported modulo", `{"operation":"modulo","a":1,"b":2}`, unsupported},
		{"unsupported uppercase ADD", `{"operation":"ADD","a":1,"b":2}`, unsupported},
		{"unsupported symbol +", `{"operation":"+","a":1,"b":2}`, unsupported},
		{"unsupported with padding", `{"operation":" add","a":1,"b":2}`, unsupported},
		{"unsupported op with b missing", `{"operation":"modulo","a":1}`, unsupported},
		{"unsupported op with a missing", `{"operation":"modulo","b":1}`, unsupported},

		// Operands.
		{"a missing", `{"operation":"add","b":2}`, aRequired},
		{"a null", `{"operation":"add","a":null,"b":2}`, aRequired},
		{"a and b missing", `{"operation":"add"}`, aRequired},
		{"sqrt a missing", `{"operation":"sqrt"}`, aRequired},
		{"sqrt a null", `{"operation":"sqrt","a":null,"b":4}`, aRequired},
		{"b missing add", `{"operation":"add","a":1}`, bRequired},
		{"b null add", `{"operation":"add","a":1,"b":null}`, bRequired},
		{"b missing subtract", `{"operation":"subtract","a":1}`, bRequired},
		{"b missing multiply", `{"operation":"multiply","a":1}`, bRequired},
		{"b missing divide", `{"operation":"divide","a":1}`, bRequired},
		{"b missing power", `{"operation":"power","a":1}`, bRequired},
		{"b missing percentage", `{"operation":"percentage","a":1}`, bRequired},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stub := &stubCalc{result: 999}
			rec := do(t, New(stub.fn), http.MethodPost, tt.body)
			assertError(t, rec, http.StatusBadRequest, tt.wantMsg)
			if stub.calls != 0 {
				t.Errorf("calculator called %d times, want 0 on validation failure", stub.calls)
			}
		})
	}
}

func TestCalculate_StubErrorMapping(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantStatus int
		wantMsg    string
	}{
		{"division by zero", calculator.ErrDivisionByZero, http.StatusBadRequest, "division by zero is not allowed"},
		{"negative sqrt", calculator.ErrNegativeSqrt, http.StatusBadRequest, calculator.ErrNegativeSqrt.Error()},
		{"non-finite result", calculator.ErrNonFiniteResult, http.StatusBadRequest, calculator.ErrNonFiniteResult.Error()},
		{"unsupported operation", calculator.ErrUnsupportedOperation, http.StatusBadRequest, calculator.ErrUnsupportedOperation.Error()},
		{"invalid operand", calculator.ErrInvalidOperand, http.StatusBadRequest, calculator.ErrInvalidOperand.Error()},
		{"wrapped division by zero", fmt.Errorf("calc: %w", calculator.ErrDivisionByZero), http.StatusBadRequest, "division by zero is not allowed"},
		{"wrapped negative sqrt", fmt.Errorf("layer1: %w", fmt.Errorf("layer2: %w", calculator.ErrNegativeSqrt)), http.StatusBadRequest, calculator.ErrNegativeSqrt.Error()},
		{"wrapped invalid operand", fmt.Errorf("ctx: %w", calculator.ErrInvalidOperand), http.StatusBadRequest, calculator.ErrInvalidOperand.Error()},
		{"unknown error", errors.New("secret db failure"), http.StatusInternalServerError, "internal server error"},
		{"wrapped unknown error", fmt.Errorf("outer: %w", errors.New("secret db failure")), http.StatusInternalServerError, "internal server error"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stub := &stubCalc{err: tt.err}
			rec := do(t, New(stub.fn), http.MethodPost, `{"operation":"divide","a":10,"b":2}`)
			assertError(t, rec, tt.wantStatus, tt.wantMsg)
			if stub.calls != 1 {
				t.Errorf("calculator called %d times, want 1", stub.calls)
			}
			if strings.Contains(rec.Body.String(), "secret") {
				t.Errorf("response leaks underlying error: %q", rec.Body.String())
			}
		})
	}
}

func TestCalculate_PassesArgumentsToCalculator(t *testing.T) {
	tests := []struct {
		name   string
		body   string
		wantOp calculator.Operation
		wantA  float64
		wantB  float64
		checkB bool
	}{
		{"add", `{"operation":"add","a":1.5,"b":-2}`, calculator.OpAdd, 1.5, -2, true},
		{"subtract", `{"operation":"subtract","a":3,"b":4}`, calculator.OpSubtract, 3, 4, true},
		{"multiply", `{"operation":"multiply","a":5,"b":6}`, calculator.OpMultiply, 5, 6, true},
		{"divide", `{"operation":"divide","a":10,"b":2}`, calculator.OpDivide, 10, 2, true},
		{"power", `{"operation":"power","a":2,"b":8}`, calculator.OpPower, 2, 8, true},
		{"percentage", `{"operation":"percentage","a":50,"b":200}`, calculator.OpPercentage, 50, 200, true},
		{"b zero passed through", `{"operation":"divide","a":7,"b":0}`, calculator.OpDivide, 7, 0, true},
		{"sqrt without b", `{"operation":"sqrt","a":9}`, calculator.OpSqrt, 9, 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stub := &stubCalc{result: 42.5}
			rec := do(t, New(stub.fn), http.MethodPost, tt.body)
			assertResult(t, rec, 42.5)
			if stub.calls != 1 {
				t.Fatalf("calculator called %d times, want 1", stub.calls)
			}
			if stub.gotOp != tt.wantOp {
				t.Errorf("op = %q, want %q", stub.gotOp, tt.wantOp)
			}
			if stub.gotA != tt.wantA {
				t.Errorf("a = %v, want %v", stub.gotA, tt.wantA)
			}
			if tt.checkB && stub.gotB != tt.wantB {
				t.Errorf("b = %v, want %v", stub.gotB, tt.wantB)
			}
		})
	}
}

func TestCalculate_MethodNotAllowed(t *testing.T) {
	tests := []struct {
		name   string
		method string
		body   string
	}{
		{"GET", http.MethodGet, ``},
		{"GET with valid body", http.MethodGet, `{"operation":"add","a":1,"b":2}`},
		{"PUT", http.MethodPut, `{"operation":"add","a":1,"b":2}`},
		{"DELETE", http.MethodDelete, ``},
		{"PATCH", http.MethodPatch, `{"operation":"add","a":1,"b":2}`},
		{"OPTIONS", http.MethodOptions, ``},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stub := &stubCalc{}
			rec := do(t, New(stub.fn), tt.method, tt.body)
			assertError(t, rec, http.StatusMethodNotAllowed, "method not allowed")
			if got := rec.Header().Get("Allow"); got != http.MethodPost {
				t.Errorf("Allow = %q, want %q", got, http.MethodPost)
			}
			if stub.calls != 0 {
				t.Errorf("calculator called %d times, want 0", stub.calls)
			}
		})
	}
}

func TestCalculate_BodySize(t *testing.T) {
	const valid = `{"operation":"add","a":1,"b":2}`
	padTo := func(n int, leading bool) string {
		pad := strings.Repeat(" ", n-len(valid))
		if leading {
			return pad + valid
		}
		return valid + pad
	}
	longString := `{"operation":"` + strings.Repeat("a", maxBody+10) + `","a":1,"b":2}`

	tests := []struct {
		name       string
		body       string
		wantStatus int
		wantResult float64
		wantMsg    string
	}{
		{"exactly 1 MiB leading whitespace accepted", padTo(maxBody, true), http.StatusOK, 3, ""},
		{"exactly 1 MiB trailing whitespace accepted", padTo(maxBody, false), http.StatusOK, 3, ""},
		{"1 MiB + 1 leading whitespace", padTo(maxBody+1, true), http.StatusRequestEntityTooLarge, 0, "request body too large"},
		{"2 MiB leading whitespace", padTo(2*maxBody, true), http.StatusRequestEntityTooLarge, 0, "request body too large"},
		{"1 MiB + 1 trailing whitespace", padTo(maxBody+1, false), http.StatusRequestEntityTooLarge, 0, "request body too large"},
		{"long string field", longString, http.StatusRequestEntityTooLarge, 0, "request body too large"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stub := &stubCalc{result: 3}
			rec := do(t, New(stub.fn), http.MethodPost, tt.body)
			if tt.wantStatus == http.StatusOK {
				assertResult(t, rec, tt.wantResult)
				return
			}
			assertError(t, rec, tt.wantStatus, tt.wantMsg)
			if stub.calls != 0 {
				t.Errorf("calculator called %d times, want 0", stub.calls)
			}
		})
	}
}
