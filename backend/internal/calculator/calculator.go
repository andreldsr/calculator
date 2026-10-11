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

var operations = map[Operation]func(a, b float64) (float64, error){
	OpAdd:      func(a, b float64) (float64, error) { return a + b, nil },
	OpSubtract: func(a, b float64) (float64, error) { return a - b, nil },
	OpMultiply: func(a, b float64) (float64, error) { return a * b, nil },
	OpDivide: func(a, b float64) (float64, error) {
		if b == 0 {
			return 0, ErrDivisionByZero
		}
		return a / b, nil
	},
	OpPower: func(a, b float64) (float64, error) { return math.Pow(a, b), nil },
	OpSqrt: func(a, _ float64) (float64, error) {
		if a < 0 {
			return 0, ErrNegativeSqrt
		}
		return math.Sqrt(a), nil
	},
	OpPercentage: func(a, b float64) (float64, error) {
		// Multiplying first keeps common cases exact (7% of 100 is 7, not
		// 7.000000000000001); dividing first is the fallback when the
		// intermediate a*b overflows but the result is representable.
		if p := a * b; !math.IsInf(p, 0) {
			return p / 100, nil
		}
		return a / 100 * b, nil
	},
}

// Valid reports whether op is a supported operation.
func (op Operation) Valid() bool {
	_, ok := operations[op]
	return ok
}

// Unary reports whether op uses only the a operand.
func (op Operation) Unary() bool {
	return op == OpSqrt
}

// Calculate applies op to a and b. For unary operations only a is used.
// Operands must be finite, and a result that would be ±Inf or NaN is
// reported as ErrNonFiniteResult. On error the result is always 0.
func Calculate(op Operation, a, b float64) (float64, error) {
	if !isFinite(a) || (!op.Unary() && !isFinite(b)) {
		return 0, ErrInvalidOperand
	}
	if !op.Valid() {
		return 0, ErrUnsupportedOperation
	}

	result, err := operations[op](a, b)
	if err != nil {
		return 0, err
	}
	if !isFinite(result) {
		return 0, ErrNonFiniteResult
	}
	if result == 0 {
		return 0, nil // normalize -0, which JSON would encode as "-0"
	}
	return result, nil
}

func isFinite(f float64) bool {
	return !math.IsInf(f, 0) && !math.IsNaN(f)
}
