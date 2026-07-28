package money

import (
	"fmt"
	"math"
	"math/bits"
	"strconv"
	"strings"
)

const (
	FenPerYuan = int64(100)
)

type Fen int64

func NewFen(v int64) Fen {
	return Fen(v)
}

func (f Fen) Int64() int64 {
	return int64(f)
}

func (f Fen) YuanString() string {
	value := int64(f)
	sign := ""
	if value < 0 {
		sign = "-"
	}
	magnitude := int64Magnitude(value)
	yuan := magnitude / uint64(FenPerYuan)
	fen := magnitude % uint64(FenPerYuan)
	return fmt.Sprintf("%s%d.%02d", sign, yuan, fen)
}

func (f Fen) Add(other Fen) (Fen, error) {
	left := int64(f)
	right := int64(other)
	if (right > 0 && left > math.MaxInt64-right) || (right < 0 && left < math.MinInt64-right) {
		return 0, fmt.Errorf("money add overflow: %d + %d", left, right)
	}
	return Fen(left + right), nil
}

func (f Fen) Multiply(qty int64) (Fen, error) {
	if qty == 0 || f == 0 {
		return 0, nil
	}

	overflow, product := bits.Mul64(int64Magnitude(int64(f)), int64Magnitude(qty))
	negative := (f < 0) != (qty < 0)
	limit := uint64(math.MaxInt64)
	if negative {
		limit++
	}
	if overflow != 0 || product > limit {
		return 0, fmt.Errorf("money multiply overflow: %d * %d", f, qty)
	}

	result := int64(product)
	if negative && product != uint64(1)<<63 {
		result = -result
	}
	return Fen(result), nil
}

func ParseYuan(value string) (Fen, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, fmt.Errorf("money string is empty")
	}

	sign := int64(1)
	if strings.HasPrefix(value, "-") {
		sign = -1
		value = strings.TrimPrefix(value, "-")
	} else if strings.HasPrefix(value, "+") {
		value = strings.TrimPrefix(value, "+")
	}

	if value == "" {
		return 0, fmt.Errorf("money string is invalid")
	}

	parts := strings.Split(value, ".")
	if len(parts) > 2 {
		return 0, fmt.Errorf("money string %q is invalid", value)
	}

	intPart := parts[0]
	if intPart == "" {
		intPart = "0"
	}

	fracPart := ""
	if len(parts) == 2 {
		fracPart = parts[1]
	}
	if len(fracPart) > 2 {
		return 0, fmt.Errorf("money string %q has more than 2 decimal places", value)
	}
	if fracPart == "" {
		fracPart = "00"
	} else if len(fracPart) == 1 {
		fracPart += "0"
	}

	if !allDigits(intPart) || !allDigits(fracPart) {
		return 0, fmt.Errorf("money string %q is invalid", value)
	}

	normalized := strings.TrimLeft(intPart+fracPart, "0")
	if normalized == "" {
		return 0, nil
	}

	fenValue, err := strconv.ParseUint(normalized, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("parse money %q: %w", value, err)
	}

	limit := uint64(math.MaxInt64)
	if sign < 0 {
		limit++
	}
	if fenValue > limit {
		return 0, fmt.Errorf("parse money %q: value out of range", value)
	}

	result := int64(fenValue)
	if sign < 0 && fenValue != uint64(1)<<63 {
		result = -result
	}
	return Fen(result), nil
}

func MustParseYuan(value string) Fen {
	fen, err := ParseYuan(value)
	if err != nil {
		panic(err)
	}
	return fen
}

func allDigits(value string) bool {
	for _, ch := range value {
		if ch < '0' || ch > '9' {
			return false
		}
	}
	return true
}

func int64Magnitude(v int64) uint64 {
	if v < 0 {
		return uint64(-(v + 1)) + 1
	}
	return uint64(v)
}
