package srs

import "time"

const MaxBox = 6

func NextInterval(box int, correct bool) time.Duration {
	if !correct {
		return 10 * time.Minute
	}
	switch box {
	case 1:
		return 10 * time.Minute
	case 2:
		return 24 * time.Hour
	case 3:
		return 3 * 24 * time.Hour
	case 4:
		return 7 * 24 * time.Hour
	case 5:
		return 14 * 24 * time.Hour
	default:
		return 30 * 24 * time.Hour
	}
}

func NextBox(current int, correct bool) int {
	if !correct {
		return 1
	}
	if current < 1 {
		current = 1
	}
	n := current + 1
	if n > MaxBox {
		n = MaxBox
	}
	return n
}
