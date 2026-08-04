package mcpserver

import (
	"context"
	"strings"
	"testing"
)

func TestServerSetupPlanDoesNotTouchFilesystem(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	_, out, err := serverSetupPlanTool(context.Background(), nil, serverSetupPlanInput{})
	if err != nil {
		t.Fatalf("plan: %v", err)
	}
	if out.PlanID == "" || out.HTTPURL != "http://127.0.0.1:8400" {
		t.Fatalf("unexpected output: %+v", out)
	}
	plans.Lock()
	plan := plans.m[out.PlanID]
	plans.Unlock()
	if plan.ComposeYML == "" {
		t.Fatalf("expected plan to be retained in memory")
	}
	if !strings.Contains(plan.ComposeYML, "FOOSTASH_SSH_ADDR: \"\"") {
		t.Fatalf("default plan should disable SSH listener: %s", plan.ComposeYML)
	}
}

func TestServerSetupPlanWarnsOnPublicBind(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	_, out, err := serverSetupPlanTool(context.Background(), nil, serverSetupPlanInput{BindHost: "0.0.0.0", EnableSSH: true})
	if err != nil {
		t.Fatalf("plan: %v", err)
	}
	if len(out.Warnings) != 2 {
		t.Fatalf("warnings = %v, want public HTTP and SSH warnings", out.Warnings)
	}
}
