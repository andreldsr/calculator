package calculator

import (
	"errors"
	"math"
)

// Operation identifies a calculator operation by its API name.
type Operation string

// Supported operations.
const (
	OpAdd        Operation = "add"
	OpSubtract   Operation = "subtract"
	OpMultiply   Operation = "multiply"
	OpDivide     Operation = "divide"
	OpPower      Operation = "power"      // a ^ b
	OpSqrt       Operation = "sqrt"       // √a; unary, b is ignored
	OpPercentage Operation = "percentage" // a percent of b
)

// Domain errors returned by Calculate. Compare them with errors.Is.
var (
	ErrDivisionByZero       = errors.New("division by zero is not allowed")
	ErrNegativeSqrt         = errors.New("square root of a negative number is not allowed")
	ErrUnsupportedOperation = errors.New("unsupported operation")
	ErrInvalidOperand       = errors.New("operands must be finite numbers")
	ErrNonFiniteResult      = errors.New("result is not a finite number")
)

// Calculate applies op to a and b. For OpSqrt only a is used.
// Operands must be finite, and a result that would be ±Inf or NaN is
// reported as ErrNonFiniteResult. On error the result is always 0.
func Calculate(op Operation, a, b float64) (float64, error) {
	if !isFinite(a) || (op != OpSqrt && !isFinite(b)) {
		return 0, ErrInvalidOperand
	}

	var result float64
	switch op {
	case OpAdd:
		result = a + b
	case OpSubtract:
		result = a - b
	case OpMultiply:
		result = a * b
	case OpDivide:
		if b == 0 {
			return 0, ErrDivisionByZero
		}
		result = a / b
	case OpPower:
		result = math.Pow(a, b)
	case OpSqrt:
		if a < 0 {
			return 0, ErrNegativeSqrt
		}
		result = math.Sqrt(a)
	case OpPercentage:
		// Divide first so large-but-representable results don't overflow
		// in an intermediate a*b.
		result = a / 100 * b
	default:
		return 0, ErrUnsupportedOperation
	}

	if !isFinite(result) {
		return 0, ErrNonFiniteResult
	}
	return result, nil
}

func isFinite(f float64) bool {
	return !math.IsInf(f, 0) && !math.IsNaN(f)
}
