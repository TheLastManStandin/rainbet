package httpapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

func parseDollarsToCents(value json.Number) (int64, error) {
	raw := string(value)
	if raw == "" || strings.HasPrefix(raw, "-") {
		return 0, errors.New("invalid amount")
	}
	parts := strings.Split(raw, ".")
	if len(parts) > 2 || parts[0] == "" {
		return 0, errors.New("invalid amount")
	}
	dollars, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil || dollars < 0 {
		return 0, errors.New("invalid amount")
	}
	cents := int64(0)
	if len(parts) == 2 {
		if len(parts[1]) > 2 {
			return 0, errors.New("amount has more than two decimal places")
		}
		if parts[1] != "" {
			cents, err = strconv.ParseInt(parts[1], 10, 64)
			if err != nil || cents < 0 {
				return 0, errors.New("invalid amount")
			}
			if len(parts[1]) == 1 {
				cents *= 10
			}
		}
	}
	const maxInt64 = int64(1<<63 - 1)
	if dollars > (maxInt64-cents)/100 {
		return 0, errors.New("amount is too large")
	}
	return dollars*100 + cents, nil
}

func formatCents(cents int64) string {
	return fmt.Sprintf("%d.%02d", cents/100, cents%100)
}

func formatHundredths(value int64) string {
	return fmt.Sprintf("%d.%02d", value/100, value%100)
}
