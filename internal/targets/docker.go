package targets

import (
	"bytes"
	"fmt"
	"sort"
	"strings"
)

// RenderEnvFile emits a Docker-compatible env file: one KEY=VALUE per line with
// literal values (no shell escaping or interpolation), sorted by key.
//
// Rejects keys that aren't valid env var identifiers and values containing a
// newline — Docker's env-file format cannot represent either.
func RenderEnvFile(data map[string]string) ([]byte, error) {
	keys := make([]string, 0, len(data))
	for k := range data {
		if !envKeyRE.MatchString(k) {
			return nil, fmt.Errorf("invalid env var name for docker: %q", k)
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var buf bytes.Buffer
	for _, k := range keys {
		v := data[k]
		if strings.ContainsAny(v, "\n\r") {
			return nil, fmt.Errorf("value for %q contains a newline; docker env files cannot carry multi-line values", k)
		}
		fmt.Fprintf(&buf, "%s=%s\n", k, v)
	}
	return buf.Bytes(), nil
}
