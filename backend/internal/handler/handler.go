package handler

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"

	"github.com/andre/sezzle-calculator/backend/internal/calculator"
)

// maxBodyBytes caps the request body; a calculation needs only a few bytes.
const maxBodyBytes = 1 << 20

// Request errors. Their messages are returned to the client verbatim.
var (
	errMethodNotAllowed  = errors.New("method not allowed")
	errInvalidBody       = errors.New("invalid request body")
	errBodyTooLarge      = errors.New("request body too large")
	errOperationRequired = errors.New("operation is required")
	errARequired         = errors.New("a is required")
	errBRequired         = errors.New("b is required")
	errInternal          = errors.New("internal server error")
)

// domainErrors are calculator errors caused by the client's input.
var domainErrors = []error{
	calculator.ErrDivisionByZero,
	calculator.ErrNegativeSqrt,
	calculator.ErrUnsupportedOperation,
	calculator.ErrInvalidOperand,
	calculator.ErrNonFiniteResult,
}

// CalculateFunc matches calculator.Calculate; injected so tests can stub it.
type CalculateFunc func(op calculator.Operation, a, b float64) (float64, error)

type calculateRequest struct {
	Operation *string  `json:"operation"`
	A         *float64 `json:"a"`
	B         *float64 `json:"b"`
}

type resultResponse struct {
	Result float64 `json:"result"`
}

type errorResponse struct {
	Error string `json:"error"`
}

type calculateHandler struct {
	calc CalculateFunc
}

// New returns the handler for POST /api/v1/calculate.
func New(calc CalculateFunc) http.Handler {
	return &calculateHandler{calc: calc}
}

func (h *calculateHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeError(w, http.StatusMethodNotAllowed, errMethodNotAllowed)
		return
	}

	req, err := decodeRequest(w, r)
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, errBodyTooLarge) {
			status = http.StatusRequestEntityTooLarge
		}
		writeError(w, status, err)
		return
	}

	op, a, b, err := validate(req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	result, err := h.calc(op, a, b)
	if err != nil {
		for _, domainErr := range domainErrors {
			if errors.Is(err, domainErr) {
				writeError(w, http.StatusBadRequest, domainErr)
				return
			}
		}
		log.Printf("calculate %s: %v", op, err)
		writeError(w, http.StatusInternalServerError, errInternal)
		return
	}

	writeJSON(w, http.StatusOK, resultResponse{Result: result})
}

// decodeRequest reads exactly one JSON object with known fields from the body.
func decodeRequest(w http.ResponseWriter, r *http.Request) (calculateRequest, error) {
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBodyBytes))
	dec.DisallowUnknownFields()

	var req calculateRequest
	if err := dec.Decode(&req); err != nil {
		return calculateRequest{}, decodeError(err)
	}
	// Anything after the object, other than whitespace, is rejected.
	if err := dec.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return calculateRequest{}, decodeError(err)
	}
	return req, nil
}

func decodeError(err error) error {
	var tooLarge *http.MaxBytesError
	if errors.As(err, &tooLarge) {
		return errBodyTooLarge
	}
	return errInvalidBody
}

func validate(req calculateRequest) (op calculator.Operation, a, b float64, err error) {
	if req.Operation == nil || *req.Operation == "" {
		return "", 0, 0, errOperationRequired
	}
	op = calculator.Operation(*req.Operation)
	if !op.Valid() {
		return "", 0, 0, calculator.ErrUnsupportedOperation
	}
	if req.A == nil {
		return "", 0, 0, errARequired
	}
	if req.B == nil && !op.Unary() {
		return "", 0, 0, errBRequired
	}
	if req.B != nil {
		b = *req.B
	}
	return op, *req.A, b, nil
}

func writeError(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, errorResponse{Error: err.Error()})
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		log.Printf("encode response: %v", err)
	}
}
