package calculator

import (
	"errors"
	"math"
	"testing"
)

// relTol is the relative tolerance used for results that are not exactly
// representable in float64 (e.g. 0.1 + 0.2, 1 / 3, √2).
const relTol = 1e-12

// almostEqual reports whether got and want are equal within a relative tolerance.
func almostEqual(got, want float64) bool {
	if got == want {
		return true
	}
	diff := math.Abs(got - want)
	scale := math.Max(math.Abs(got), math.Abs(want))
	return diff <= relTol*scale
}

type calcCase struct {
	name    string
	op      Operation
	a, b    float64
	want    float64
	approx  bool  // compare with almostEqual instead of exact equality
	wantErr error // if set, Calculate must return (0, err) with errors.Is(err, wantErr)
}

func runCases(t *testing.T, cases []calcCase) {
	t.Helper()
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Calculate(tc.op, tc.a, tc.b)

			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("Calculate(%q, %v, %v) error = %v, want errors.Is(err, %v)", tc.op, tc.a, tc.b, err, tc.wantErr)
				}
				if got != 0 {
					t.Errorf("Calculate(%q, %v, %v) result = %v on error, want 0", tc.op, tc.a, tc.b, got)
				}
				return
			}

			if err != nil {
				t.Fatalf("Calculate(%q, %v, %v) unexpected error: %v", tc.op, tc.a, tc.b, err)
			}
			if tc.approx {
				if !almostEqual(got, tc.want) {
					t.Errorf("Calculate(%q, %v, %v) = %v, want ≈ %v", tc.op, tc.a, tc.b, got, tc.want)
				}
				return
			}
			if got != tc.want {
				t.Errorf("Calculate(%q, %v, %v) = %v, want %v", tc.op, tc.a, tc.b, got, tc.want)
			}
		})
	}
}

func TestCalculate_Add(t *testing.T) {
	runCases(t, []calcCase{
		{name: "positive integers", op: OpAdd, a: 2, b: 3, want: 5},
		{name: "negative operands", op: OpAdd, a: -4, b: -6, want: -10},
		{name: "mixed signs", op: OpAdd, a: -4, b: 10, want: 6},
		{name: "zeros", op: OpAdd, a: 0, b: 0, want: 0},
		{name: "exact fractions", op: OpAdd, a: 1.5, b: 2.25, want: 3.75},
		{name: "0.1 + 0.2 within tolerance", op: OpAdd, a: 0.1, b: 0.2, want: 0.3, approx: true},
		{name: "large but finite", op: OpAdd, a: math.MaxFloat64 / 2, b: math.MaxFloat64 / 2, want: math.MaxFloat64},
		{name: "overflow to +Inf", op: OpAdd, a: math.MaxFloat64, b: math.MaxFloat64, wantErr: ErrNonFiniteResult},
		{name: "overflow to -Inf", op: OpAdd, a: -math.MaxFloat64, b: -math.MaxFloat64, wantErr: ErrNonFiniteResult},
	})
}

func TestCalculate_Subtract(t *testing.T) {
	runCases(t, []calcCase{
		{name: "positive integers", op: OpSubtract, a: 10, b: 4, want: 6},
		{name: "result negative", op: OpSubtract, a: 4, b: 10, want: -6},
		{name: "negative operands", op: OpSubtract, a: -4, b: -10, want: 6},
		{name: "zeros", op: OpSubtract, a: 0, b: 0, want: 0},
		{name: "exact fractions", op: OpSubtract, a: 5.75, b: 0.25, want: 5.5},
		{name: "0.3 - 0.1 within tolerance", op: OpSubtract, a: 0.3, b: 0.1, want: 0.2, approx: true},
		{name: "large but finite", op: OpSubtract, a: math.MaxFloat64, b: math.MaxFloat64, want: 0},
		{name: "overflow to +Inf", op: OpSubtract, a: math.MaxFloat64, b: -math.MaxFloat64, wantErr: ErrNonFiniteResult},
		{name: "overflow to -Inf", op: OpSubtract, a: -math.MaxFloat64, b: math.MaxFloat64, wantErr: ErrNonFiniteResult},
	})
}

func TestCalculate_Multiply(t *testing.T) {
	runCases(t, []calcCase{
		{name: "positive integers", op: OpMultiply, a: 6, b: 7, want: 42},
		{name: "negative times positive", op: OpMultiply, a: -6, b: 7, want: -42},
		{name: "negative times negative", op: OpMultiply, a: -6, b: -7, want: 42},
		{name: "by zero", op: OpMultiply, a: 123.456, b: 0, want: 0},
		{name: "exact fractions", op: OpMultiply, a: 2.5, b: 0.5, want: 1.25},
		{name: "inexact fractions within tolerance", op: OpMultiply, a: 0.1, b: 3, want: 0.3, approx: true},
		{name: "large but finite", op: OpMultiply, a: 1e154, b: 1e154, want: 1e308, approx: true},
		{name: "overflow to +Inf", op: OpMultiply, a: math.MaxFloat64, b: 2, wantErr: ErrNonFiniteResult},
		{name: "overflow to -Inf", op: OpMultiply, a: math.MaxFloat64, b: -2, wantErr: ErrNonFiniteResult},
	})
}

func TestCalculate_Divide(t *testing.T) {
	negZero := math.Copysign(0, -1)
	runCases(t, []calcCase{
		{name: "spec example 10 / 2", op: OpDivide, a: 10, b: 2, want: 5},
		{name: "negative dividend", op: OpDivide, a: -10, b: 2, want: -5},
		{name: "negative divisor", op: OpDivide, a: 10, b: -4, want: -2.5},
		{name: "both negative", op: OpDivide, a: -10, b: -4, want: 2.5},
		{name: "zero dividend", op: OpDivide, a: 0, b: 5, want: 0},
		{name: "exact fractions", op: OpDivide, a: 7.5, b: 2.5, want: 3},
		{name: "repeating result within tolerance", op: OpDivide, a: 1, b: 3, want: 1.0 / 3.0, approx: true},
		{name: "large but finite", op: OpDivide, a: math.MaxFloat64, b: 2, want: math.MaxFloat64 / 2},
		{name: "overflow to +Inf", op: OpDivide, a: math.MaxFloat64, b: 0.5, wantErr: ErrNonFiniteResult},
		{name: "positive by zero", op: OpDivide, a: 10, b: 0, wantErr: ErrDivisionByZero},
		{name: "negative by zero", op: OpDivide, a: -10, b: 0, wantErr: ErrDivisionByZero},
		{name: "zero by zero", op: OpDivide, a: 0, b: 0, wantErr: ErrDivisionByZero},
		{name: "positive by negative zero", op: OpDivide, a: 10, b: negZero, wantErr: ErrDivisionByZero},
	})
}

func TestCalculate_Power(t *testing.T) {
	runCases(t, []calcCase{
		{name: "positive integer exponent", op: OpPower, a: 2, b: 10, want: 1024},
		{name: "x ^ 0 is 1", op: OpPower, a: 7.5, b: 0, want: 1},
		{name: "0 ^ 0 is 1", op: OpPower, a: 0, b: 0, want: 1},
		{name: "0 ^ positive is 0", op: OpPower, a: 0, b: 5, want: 0},
		{name: "negative base even exponent", op: OpPower, a: -2, b: 2, want: 4},
		{name: "negative base odd exponent", op: OpPower, a: -2, b: 3, want: -8},
		{name: "negative exponent", op: OpPower, a: 2, b: -2, want: 0.25},
		{name: "fractional exponent square root", op: OpPower, a: 4, b: 0.5, want: 2},
		{name: "fractional exponent cube root within tolerance", op: OpPower, a: 27, b: 1.0 / 3.0, want: 3, approx: true},
		{name: "fractional base", op: OpPower, a: 0.5, b: 3, want: 0.125},
		{name: "large but finite", op: OpPower, a: 10, b: 308, want: 1e308, approx: true},
		{name: "overflow to +Inf", op: OpPower, a: 10, b: 309, wantErr: ErrNonFiniteResult},
		{name: "overflow to -Inf", op: OpPower, a: -10, b: 309, wantErr: ErrNonFiniteResult},
		{name: "0 ^ -1 is Inf", op: OpPower, a: 0, b: -1, wantErr: ErrNonFiniteResult},
		{name: "negative base fractional exponent is NaN", op: OpPower, a: -8, b: 1.0 / 3.0, wantErr: ErrNonFiniteResult},
	})
}

func TestCalculate_Sqrt(t *testing.T) {
	runCases(t, []calcCase{
		{name: "zero", op: OpSqrt, a: 0, want: 0},
		{name: "one", op: OpSqrt, a: 1, want: 1},
		{name: "perfect square", op: OpSqrt, a: 144, want: 12},
		{name: "fractional perfect square", op: OpSqrt, a: 2.25, want: 1.5},
		{name: "non-perfect square within tolerance", op: OpSqrt, a: 2, want: math.Sqrt2, approx: true},
		{name: "large but finite", op: OpSqrt, a: 1e300, want: 1e150, approx: true},
		{name: "b is ignored", op: OpSqrt, a: 9, b: 12345, want: 3},
		{name: "non-finite b is ignored", op: OpSqrt, a: 9, b: math.NaN(), want: 3},
		{name: "infinite b is ignored", op: OpSqrt, a: 16, b: math.Inf(-1), want: 4},
		{name: "negative", op: OpSqrt, a: -4, wantErr: ErrNegativeSqrt},
		{name: "tiny negative", op: OpSqrt, a: -1e-300, wantErr: ErrNegativeSqrt},
		{name: "negative with b set", op: OpSqrt, a: -9, b: 9, wantErr: ErrNegativeSqrt},
	})
}

func TestCalculate_Percentage(t *testing.T) {
	runCases(t, []calcCase{
		{name: "50 percent of 200", op: OpPercentage, a: 50, b: 200, want: 100},
		{name: "0 percent", op: OpPercentage, a: 0, b: 200, want: 0},
		{name: "100 percent", op: OpPercentage, a: 100, b: 200, want: 200},
		{name: "percent of zero", op: OpPercentage, a: 50, b: 0, want: 0},
		{name: "over 100 percent", op: OpPercentage, a: 250, b: 40, want: 100},
		{name: "negative percent", op: OpPercentage, a: -50, b: 200, want: -100},
		{name: "percent of negative", op: OpPercentage, a: 50, b: -200, want: -100},
		{name: "fractional percent", op: OpPercentage, a: 12.5, b: 80, want: 10},
		{name: "inexact percent within tolerance", op: OpPercentage, a: 15, b: 19.99, want: 2.9985, approx: true},
		{name: "large but finite", op: OpPercentage, a: 200, b: 1e307, want: 2e307, approx: true},
		{name: "overflow to +Inf", op: OpPercentage, a: math.MaxFloat64, b: math.MaxFloat64, wantErr: ErrNonFiniteResult},
		{name: "overflow to -Inf", op: OpPercentage, a: -math.MaxFloat64, b: math.MaxFloat64, wantErr: ErrNonFiniteResult},
	})
}

func TestCalculate_NonFiniteOperands(t *testing.T) {
	binaryOps := []Operation{OpAdd, OpSubtract, OpMultiply, OpDivide, OpPower, OpPercentage}
	nonFinite := []struct {
		name string
		v    float64
	}{
		{"NaN", math.NaN()},
		{"+Inf", math.Inf(1)},
		{"-Inf", math.Inf(-1)},
	}

	var cases []calcCase
	for _, op := range binaryOps {
		for _, nf := range nonFinite {
			cases = append(cases,
				calcCase{name: string(op) + "/a=" + nf.name, op: op, a: nf.v, b: 2, wantErr: ErrInvalidOperand},
				calcCase{name: string(op) + "/b=" + nf.name, op: op, a: 2, b: nf.v, wantErr: ErrInvalidOperand},
			)
		}
	}
	for _, nf := range nonFinite {
		cases = append(cases,
			calcCase{name: "sqrt/a=" + nf.name, op: OpSqrt, a: nf.v, wantErr: ErrInvalidOperand},
		)
	}
	// Operand validation runs before any operation-specific check.
	cases = append(cases,
		calcCase{name: "divide/NaN by zero is invalid operand", op: OpDivide, a: math.NaN(), b: 0, wantErr: ErrInvalidOperand},
		calcCase{name: "divide/+Inf by zero is invalid operand", op: OpDivide, a: math.Inf(1), b: 0, wantErr: ErrInvalidOperand},
		calcCase{name: "sqrt/-Inf is invalid operand not negative sqrt", op: OpSqrt, a: math.Inf(-1), wantErr: ErrInvalidOperand},
	)

	runCases(t, cases)
}

func TestCalculate_UnsupportedOperation(t *testing.T) {
	runCases(t, []calcCase{
		{name: "empty", op: "", a: 1, b: 2, wantErr: ErrUnsupportedOperation},
		{name: "unknown word", op: "modulo", a: 1, b: 2, wantErr: ErrUnsupportedOperation},
		{name: "upper case", op: "ADD", a: 1, b: 2, wantErr: ErrUnsupportedOperation},
		{name: "mixed case", op: "Divide", a: 1, b: 2, wantErr: ErrUnsupportedOperation},
		{name: "surrounding whitespace", op: " add ", a: 1, b: 2, wantErr: ErrUnsupportedOperation},
		{name: "plus symbol", op: "+", a: 1, b: 2, wantErr: ErrUnsupportedOperation},
		{name: "slash symbol", op: "/", a: 1, b: 2, wantErr: ErrUnsupportedOperation},
		{name: "caret symbol", op: "^", a: 1, b: 2, wantErr: ErrUnsupportedOperation},
		{name: "percent symbol", op: "%", a: 1, b: 2, wantErr: ErrUnsupportedOperation},
	})
}

// The division-by-zero message is part of the API contract in TASK_SPEC.md,
// so its exact text is asserted here (and only here).
func TestErrDivisionByZero_MessageMatchesSpec(t *testing.T) {
	const want = "division by zero is not allowed"
	if got := ErrDivisionByZero.Error(); got != want {
		t.Errorf("ErrDivisionByZero.Error() = %q, want %q", got, want)
	}
}

// Sentinel errors must be distinct so errors.Is can tell them apart.
func TestSentinelErrors_AreDistinct(t *testing.T) {
	sentinels := []struct {
		name string
		err  error
	}{
		{"ErrDivisionByZero", ErrDivisionByZero},
		{"ErrNegativeSqrt", ErrNegativeSqrt},
		{"ErrUnsupportedOperation", ErrUnsupportedOperation},
		{"ErrInvalidOperand", ErrInvalidOperand},
		{"ErrNonFiniteResult", ErrNonFiniteResult},
	}
	for i, x := range sentinels {
		for j, y := range sentinels {
			if i == j {
				continue
			}
			t.Run(x.name+"_vs_"+y.name, func(t *testing.T) {
				if errors.Is(x.err, y.err) {
					t.Errorf("errors.Is(%s, %s) = true, want false", x.name, y.name)
				}
			})
		}
	}
}
