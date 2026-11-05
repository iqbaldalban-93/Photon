package protox

import (
	"sort"
	"strconv"
)

func FromMap(m map[string]any) []byte {
	return encodeMap(m)
}

func encodeMap(m map[string]any) []byte {
	nums := make([]int, 0, len(m))
	byNum := make(map[int]string, len(m))
	for k := range m {
		n, err := strconv.Atoi(k)
		if err != nil {
			continue
		}
		nums = append(nums, n)
		byNum[n] = k
	}
	sort.Ints(nums)

	var out []byte
	for _, n := range nums {
		out = encodeValue(out, n, m[byNum[n]])
	}
	return out
}

func encodeValue(out []byte, num int, v any) []byte {
	switch val := v.(type) {
	case nil:
		return out
	case map[string]any:
		out = appendTag(out, num, 2)
		sub := encodeMap(val)
		out = appendVarint(out, uint64(len(sub)))
		return append(out, sub...)
	case string:
		out = appendTag(out, num, 2)
		out = appendVarint(out, uint64(len(val)))
		return append(out, val...)
	case []byte:
		out = appendTag(out, num, 2)
		out = appendVarint(out, uint64(len(val)))
		return append(out, val...)
	case bool:
		out = appendTag(out, num, 0)
		if val {
			return appendVarint(out, 1)
		}
		return appendVarint(out, 0)
	case int:
		out = appendTag(out, num, 0)
		return appendVarint(out, uint64(val))
	case int64:
		out = appendTag(out, num, 0)
		return appendVarint(out, uint64(val))
	case uint64:
		out = appendTag(out, num, 0)
		return appendVarint(out, val)
	case []int:
		for _, item := range val {
			out = appendTag(out, num, 0)
			out = appendVarint(out, uint64(item))
		}
		return out
	case []string:
		for _, item := range val {
			out = appendTag(out, num, 2)
			out = appendVarint(out, uint64(len(item)))
			out = append(out, item...)
		}
		return out
	case []any:
		for _, item := range val {
			out = encodeValue(out, num, item)
		}
		return out
	default:
		return out
	}
}
