package diff

import "sort"

type DiffResult struct {
	OnlyLeft  []string
	OnlyRight []string
	Different map[string][2]string // key → [leftVal, rightVal]
	Identical []string
}

func Compare(left, right map[string]string) DiffResult {
	result := DiffResult{
		Different: make(map[string][2]string),
	}

	for k, lv := range left {
		rv, exists := right[k]
		if !exists {
			result.OnlyLeft = append(result.OnlyLeft, k)
		} else if lv != rv {
			result.Different[k] = [2]string{lv, rv}
		} else {
			result.Identical = append(result.Identical, k)
		}
	}

	for k := range right {
		if _, exists := left[k]; !exists {
			result.OnlyRight = append(result.OnlyRight, k)
		}
	}

	sort.Strings(result.OnlyLeft)
	sort.Strings(result.OnlyRight)
	sort.Strings(result.Identical)

	return result
}
