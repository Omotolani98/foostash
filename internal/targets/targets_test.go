package targets

import (
	"strings"
	"testing"
)

func TestRenderEnvFile(t *testing.T) {
	t.Run("sorted output", func(t *testing.T) {
		out, err := RenderEnvFile(map[string]string{"B": "2", "A": "1"})
		if err != nil {
			t.Fatal(err)
		}
		if got, want := string(out), "A=1\nB=2\n"; got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("literal value preserved", func(t *testing.T) {
		out, err := RenderEnvFile(map[string]string{"URL": "postgres://u:p@h/d?x=1&y=2"})
		if err != nil {
			t.Fatal(err)
		}
		if got, want := string(out), "URL=postgres://u:p@h/d?x=1&y=2\n"; got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("rejects invalid key", func(t *testing.T) {
		if _, err := RenderEnvFile(map[string]string{"bad-key": "x"}); err == nil {
			t.Fatal("expected error for invalid key")
		}
	})

	t.Run("rejects newline in value", func(t *testing.T) {
		if _, err := RenderEnvFile(map[string]string{"A": "one\ntwo"}); err == nil {
			t.Fatal("expected error for newline value")
		}
	})

	t.Run("empty map", func(t *testing.T) {
		out, err := RenderEnvFile(map[string]string{})
		if err != nil {
			t.Fatal(err)
		}
		if len(out) != 0 {
			t.Errorf("expected empty output, got %q", out)
		}
	})
}

func TestRenderSecret(t *testing.T) {
	t.Run("basic manifest", func(t *testing.T) {
		out, err := RenderSecret("my-secret", "default", map[string]string{"API_KEY": "hello"})
		if err != nil {
			t.Fatal(err)
		}
		s := string(out)
		for _, needle := range []string{
			"apiVersion: v1",
			"kind: Secret",
			"name: my-secret",
			"namespace: default",
			"type: Opaque",
			"API_KEY: aGVsbG8=",
		} {
			if !strings.Contains(s, needle) {
				t.Errorf("missing %q in:\n%s", needle, s)
			}
		}
	})

	t.Run("omits namespace when empty", func(t *testing.T) {
		out, err := RenderSecret("foo", "", map[string]string{"X": "y"})
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(out), "namespace:") {
			t.Errorf("expected no namespace line; got:\n%s", out)
		}
	})

	t.Run("empty data uses {}", func(t *testing.T) {
		out, err := RenderSecret("foo", "", map[string]string{})
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(out), "data: {}") {
			t.Errorf("expected `data: {}`; got:\n%s", out)
		}
	})

	t.Run("keys are sorted", func(t *testing.T) {
		out, err := RenderSecret("foo", "", map[string]string{"B": "2", "A": "1"})
		if err != nil {
			t.Fatal(err)
		}
		s := string(out)
		if strings.Index(s, "A:") >= strings.Index(s, "B:") {
			t.Errorf("expected A before B:\n%s", s)
		}
	})

	t.Run("invalid name", func(t *testing.T) {
		if _, err := RenderSecret("Bad_Name", "", nil); err == nil {
			t.Fatal("expected error for invalid name")
		}
	})

	t.Run("invalid data key", func(t *testing.T) {
		if _, err := RenderSecret("foo", "", map[string]string{"bad-key": "x"}); err == nil {
			t.Fatal("expected error for invalid data key")
		}
	})
}

func TestRenderWorkflowEnv(t *testing.T) {
	t.Run("single-line secret", func(t *testing.T) {
		out, err := RenderWorkflowEnv(map[string]string{"FOO": "bar"})
		if err != nil {
			t.Fatal(err)
		}
		s := string(out)
		for _, needle := range []string{"::add-mask::", "FOO<<__FOOSTASH_EOF__", "$GITHUB_ENV", "bar"} {
			if !strings.Contains(s, needle) {
				t.Errorf("missing %q in:\n%s", needle, s)
			}
		}
	})

	t.Run("multi-line body preserved", func(t *testing.T) {
		out, err := RenderWorkflowEnv(map[string]string{"CERT": "line1\nline2"})
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(out), "line1\nline2") {
			t.Errorf("missing multi-line body:\n%s", out)
		}
	})

	t.Run("invalid key", func(t *testing.T) {
		if _, err := RenderWorkflowEnv(map[string]string{"bad-key": "x"}); err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("value with reserved delim", func(t *testing.T) {
		if _, err := RenderWorkflowEnv(map[string]string{"X": "a\n__FOOSTASH_EOF__\nb"}); err == nil {
			t.Fatal("expected error for reserved delimiter")
		}
	})

	t.Run("empty map", func(t *testing.T) {
		out, err := RenderWorkflowEnv(map[string]string{})
		if err != nil {
			t.Fatal(err)
		}
		if len(out) != 0 {
			t.Errorf("expected empty output, got %q", out)
		}
	})
}
