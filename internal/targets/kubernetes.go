package targets

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"regexp"
	"sort"
)

var dns1123LabelRE = regexp.MustCompile(`^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`)

// RenderSecret emits a Kubernetes v1 Secret manifest (type Opaque) with
// base64-encoded values under data:. Namespace is omitted when empty.
//
// Data keys are validated as env var identifiers so `envFrom: secretRef:`
// projects them cleanly into Pods without silent drops.
func RenderSecret(name, namespace string, data map[string]string) ([]byte, error) {
	if !dns1123LabelRE.MatchString(name) || len(name) > 253 {
		return nil, fmt.Errorf("invalid Kubernetes name %q (must be a DNS-1123 label)", name)
	}
	if namespace != "" && (!dns1123LabelRE.MatchString(namespace) || len(namespace) > 253) {
		return nil, fmt.Errorf("invalid Kubernetes namespace %q", namespace)
	}

	keys := make([]string, 0, len(data))
	for k := range data {
		if !envKeyRE.MatchString(k) {
			return nil, fmt.Errorf("invalid Secret data key %q (must be a valid env var identifier)", k)
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var buf bytes.Buffer
	buf.WriteString("apiVersion: v1\n")
	buf.WriteString("kind: Secret\n")
	buf.WriteString("metadata:\n")
	fmt.Fprintf(&buf, "  name: %s\n", name)
	if namespace != "" {
		fmt.Fprintf(&buf, "  namespace: %s\n", namespace)
	}
	buf.WriteString("type: Opaque\n")
	if len(keys) == 0 {
		buf.WriteString("data: {}\n")
		return buf.Bytes(), nil
	}
	buf.WriteString("data:\n")
	for _, k := range keys {
		fmt.Fprintf(&buf, "  %s: %s\n", k, base64.StdEncoding.EncodeToString([]byte(data[k])))
	}
	return buf.Bytes(), nil
}
