package diff

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

func FormatTable(result DiffResult, leftName, rightName string, mask, onlyChanges bool) string {
	var b strings.Builder

	// calculate column widths
	keyWidth := len("Key")
	leftWidth := len(leftName)
	rightWidth := len(rightName)

	allKeys := collectKeys(result, onlyChanges)
	for _, k := range allKeys {
		if len(k) > keyWidth {
			keyWidth = len(k)
		}
	}

	if keyWidth > 30 {
		keyWidth = 30
	}
	if leftWidth < 16 {
		leftWidth = 16
	}
	if rightWidth < 16 {
		rightWidth = 16
	}

	// header
	fmt.Fprintf(&b, "  %-*s  %-*s  %s\n", keyWidth, "Key", leftWidth, leftName, rightName)
	fmt.Fprintf(&b, "  %s\n", strings.Repeat("─", keyWidth+leftWidth+rightWidth+4))

	// only in right (+)
	for _, k := range result.OnlyRight {
		rv := maskVal(result.Different, k, 1, mask)
		// OnlyRight keys aren't in Different, get from the caller context
		if !mask {
			rv = "—" // we don't have the value in DiffResult for OnlyRight
		} else {
			rv = "****"
		}
		fmt.Fprintf(&b, "+ %-*s  %-*s  %s\n", keyWidth, truncate(k, keyWidth), leftWidth, "—", rv)
	}

	// only in left (-)
	for _, k := range result.OnlyLeft {
		lv := "****"
		if !mask {
			lv = "—"
		}
		fmt.Fprintf(&b, "- %-*s  %-*s  %s\n", keyWidth, truncate(k, keyWidth), leftWidth, lv, "—")
	}

	// different (~)
	diffKeys := make([]string, 0, len(result.Different))
	for k := range result.Different {
		diffKeys = append(diffKeys, k)
	}
	sort.Strings(diffKeys)
	for _, k := range diffKeys {
		vals := result.Different[k]
		lv, rv := vals[0], vals[1]
		if mask {
			lv, rv = "****", "****"
		}
		fmt.Fprintf(&b, "~ %-*s  %-*s  %s\n", keyWidth, truncate(k, keyWidth), leftWidth, lv, rv)
	}

	// identical (=)
	if !onlyChanges {
		for _, k := range result.Identical {
			fmt.Fprintf(&b, "= %-*s  %-*s  %s\n", keyWidth, truncate(k, keyWidth), leftWidth, "(same)", "(same)")
		}
	}

	return b.String()
}

type jsonDiff struct {
	Left      string              `json:"left"`
	Right     string              `json:"right"`
	OnlyLeft  []string            `json:"only_left"`
	OnlyRight []string            `json:"only_right"`
	Different map[string]jsonPair `json:"different"`
	Identical []string            `json:"identical"`
}

type jsonPair struct {
	Left  string `json:"left"`
	Right string `json:"right"`
}

func FormatJSON(result DiffResult, leftName, rightName string) ([]byte, error) {
	d := jsonDiff{
		Left:      leftName,
		Right:     rightName,
		OnlyLeft:  result.OnlyLeft,
		OnlyRight: result.OnlyRight,
		Different: make(map[string]jsonPair),
		Identical: result.Identical,
	}
	if d.OnlyLeft == nil {
		d.OnlyLeft = []string{}
	}
	if d.OnlyRight == nil {
		d.OnlyRight = []string{}
	}
	if d.Identical == nil {
		d.Identical = []string{}
	}
	for k, v := range result.Different {
		d.Different[k] = jsonPair{Left: v[0], Right: v[1]}
	}
	return json.MarshalIndent(d, "", "  ")
}

func collectKeys(result DiffResult, onlyChanges bool) []string {
	var keys []string
	keys = append(keys, result.OnlyLeft...)
	keys = append(keys, result.OnlyRight...)
	for k := range result.Different {
		keys = append(keys, k)
	}
	if !onlyChanges {
		keys = append(keys, result.Identical...)
	}
	return keys
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-1] + "…"
}

func maskVal(_ map[string][2]string, _ string, _ int, mask bool) string {
	if mask {
		return "****"
	}
	return ""
}
