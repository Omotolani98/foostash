package targets

import (
	"bytes"
	"fmt"
	"sort"
	"strings"
)

const ghDelim = "__FOOSTASH_EOF__"

// RenderWorkflowEnv emits a POSIX shell script body intended for a GitHub
// Actions `run:` step. Per secret it:
//  1. Masks each non-empty line via `::add-mask::` so values never appear in logs.
//  2. Appends KEY<<DELIM / value / DELIM to $GITHUB_ENV so later steps see it as
//     an env var. Heredoc form preserves multi-line values.
func RenderWorkflowEnv(data map[string]string) ([]byte, error) {
	keys := make([]string, 0, len(data))
	for k := range data {
		if !envKeyRE.MatchString(k) {
			return nil, fmt.Errorf("invalid env var name %q for GitHub Actions", k)
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var buf bytes.Buffer
	for _, k := range keys {
		v := data[k]
		if strings.Contains(v, ghDelim) {
			return nil, fmt.Errorf("value for %q contains reserved heredoc delimiter %q", k, ghDelim)
		}
		body := strings.TrimRight(v, "\n")
		fmt.Fprintf(&buf,
			"while IFS= read -r __fs_line || [ -n \"$__fs_line\" ]; do [ -n \"$__fs_line\" ] && echo \"::add-mask::$__fs_line\"; done <<'%s'\n%s\n%s\n",
			ghDelim, body, ghDelim)
		fmt.Fprintf(&buf,
			"{ echo \"%s<<%s\"; cat <<'%s'\n%s\n%s\n } >> \"$GITHUB_ENV\"\n",
			k, ghDelim, ghDelim, body, ghDelim)
	}
	return buf.Bytes(), nil
}
