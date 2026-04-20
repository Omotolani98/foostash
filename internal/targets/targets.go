// Package targets renders foostash secrets into formats natively consumed by
// deployment tools: Docker env files, Kubernetes Secret manifests, and
// GitHub Actions workflow step snippets.
package targets

import "regexp"

var envKeyRE = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)
