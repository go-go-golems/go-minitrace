package serve

import "math"

func clampIntToUint32(value int) uint32 {
	if value <= 0 {
		return 0
	}
	if value > math.MaxUint32 {
		return math.MaxUint32
	}
	return uint32(value)
}

func clampIntToInt32(value int) int32 {
	if value < math.MinInt32 {
		return math.MinInt32
	}
	if value > math.MaxInt32 {
		return math.MaxInt32
	}
	return int32(value)
}

func clampOptionalIntToUint32(value *int) *uint32 {
	if value == nil {
		return nil
	}
	v := clampIntToUint32(*value)
	return &v
}
