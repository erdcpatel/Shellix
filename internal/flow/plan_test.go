package flow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestComputeExecutionWaves(t *testing.T) {
	// Diamond DAG:
	//       step1
	//      /     \
	//   step2a   step2b
	//      \     /
	//       step3
	flow := &Flow{
		Name: "diamond-pipeline",
		Steps: []Step{
			{Name: "step1", Run: "echo 1"},
			{Name: "step2a", Run: "echo 2a", DependsOn: []string{"step1"}},
			{Name: "step2b", Run: "echo 2b", DependsOn: []string{"step1"}},
			{Name: "step3", Run: "echo 3", DependsOn: []string{"step2a", "step2b"}},
		},
	}

	waves := ComputeExecutionWaves(flow)
	if len(waves) != 3 {
		t.Fatalf("expected 3 waves, got %d", len(waves))
	}

	// Wave 1: step1
	if len(waves[0].Steps) != 1 || waves[0].Steps[0].Name != "step1" {
		t.Errorf("expected wave 1 to have step1, got %+v", waves[0].Steps)
	}
	if waves[0].IsParallel {
		t.Errorf("wave 1 should not be parallel")
	}

	// Wave 2: step2a and step2b (concurrent)
	if len(waves[1].Steps) != 2 {
		t.Errorf("expected wave 2 to have 2 steps, got %d", len(waves[1].Steps))
	}
	if !waves[1].IsParallel {
		t.Errorf("wave 2 should be marked parallel")
	}

	// Wave 3: step3
	if len(waves[2].Steps) != 1 || waves[2].Steps[0].Name != "step3" {
		t.Errorf("expected wave 3 to have step3, got %+v", waves[2].Steps)
	}
}

func TestRenderFlowPlanAndMermaid(t *testing.T) {
	flow := &Flow{
		Name:        "build-and-deploy",
		Description: "Production release pipeline",
		Params: map[string]string{
			"env":     "staging",
			"version": "1.0.0",
		},
		Steps: []Step{
			{Name: "compile", Run: "echo compiling v{{params.version}}"},
			{Name: "test", Run: "echo testing", DependsOn: []string{"compile"}},
			{
				Name:      "deploy",
				Run:       "echo deploying to {{params.env}}",
				DependsOn: []string{"test"},
				Prompt:    "Confirm deployment to production?",
				Artifacts: []string{"build/*.tar.gz"},
			},
		},
	}

	ctx := NewContext(".", "main")
	ctx.ApplyFlowDefaults(flow.Params)
	ctx.SetParam("env", "production")

	planOutput := RenderFlowPlan(flow, ctx)
	if !strings.Contains(planOutput, "Execution Plan: build-and-deploy") {
		t.Errorf("plan output missing title: %s", planOutput)
	}
	if !strings.Contains(planOutput, "env = production") {
		t.Errorf("plan output missing parameter override: %s", planOutput)
	}
	if !strings.Contains(planOutput, "approval_gate") {
		t.Errorf("plan output missing approval gate notice: %s", planOutput)
	}
	if !strings.Contains(planOutput, "artifacts: [build/*.tar.gz]") {
		t.Errorf("plan output missing artifacts notice: %s", planOutput)
	}

	mermaidOutput := RenderMermaid(flow)
	if !strings.Contains(mermaidOutput, "flowchart TD") {
		t.Errorf("mermaid output missing header: %s", mermaidOutput)
	}
	if !strings.Contains(mermaidOutput, "compile --> test") {
		t.Errorf("mermaid output missing dependency link: %s", mermaidOutput)
	}
	if !strings.Contains(mermaidOutput, "⚠️ deploy (Approval)") {
		t.Errorf("mermaid output missing approval gate icon: %s", mermaidOutput)
	}
}

func TestFlowParametersInterpolation(t *testing.T) {
	flow := &Flow{
		Name: "param-test",
		Params: map[string]string{
			"env":     "development",
			"retries": "3",
		},
	}

	ctx := NewContext("/tmp/repo", "feat-auth")
	ctx.ApplyFlowDefaults(flow.Params)

	// Flow defaults should be applied
	if res := ctx.Interpolate("Deploying to {{params.env}}"); res != "Deploying to development" {
		t.Errorf("unexpected default interpolation: %s", res)
	}
	if res := ctx.Interpolate("Retries: {{param.retries}}"); res != "Retries: 3" {
		t.Errorf("unexpected alias interpolation: %s", res)
	}

	// Override via SetParam (e.g. from --set env=prod)
	ctx.SetParam("env", "prod")
	if res := ctx.Interpolate("Deploying to {{params.env}}"); res != "Deploying to prod" {
		t.Errorf("unexpected overridden interpolation: %s", res)
	}
}

func TestFlowArtifactCollection(t *testing.T) {
	tmpDir := t.TempDir()

	// Create dummy build artifacts
	buildDir := filepath.Join(tmpDir, "dist")
	_ = os.MkdirAll(buildDir, 0755)
	appBinary := filepath.Join(buildDir, "shellix-app")
	_ = os.WriteFile(appBinary, []byte("compiled binary content here"), 0644)
	appChecksum := filepath.Join(buildDir, "checksums.txt")
	_ = os.WriteFile(appChecksum, []byte("sha256 abcdef123456"), 0644)

	flowName := "test-artifact-pipeline"
	runID := "run_test_001"

	collected, err := CollectArtifacts(flowName, runID, tmpDir, []string{"dist/*"})
	if err != nil {
		t.Fatalf("CollectArtifacts failed: %v", err)
	}

	if len(collected) != 2 {
		t.Fatalf("expected 2 artifacts collected, got %d: %v", len(collected), collected)
	}

	// Verify manifest in ~/.shellix/artifacts/<flowName>/<runID>/manifest.json
	manifests, err := ListArtifactRuns(flowName)
	if err != nil {
		t.Fatalf("ListArtifactRuns failed: %v", err)
	}

	foundRun := false
	for _, m := range manifests {
		if m.RunID == runID {
			foundRun = true
			if len(m.Files) != 2 {
				t.Errorf("manifest files count mismatch: %d", len(m.Files))
			}
		}
	}
	if !foundRun {
		t.Errorf("expected to find runID %s in manifests", runID)
	}

	// Clean up created artifact dir
	_ = os.RemoveAll(filepath.Join(GetArtifactsBaseDir(), flowName))
}

func TestExecutorWithAutoApprovalAndArtifacts(t *testing.T) {
	tmpDir := t.TempDir()

	flow := &Flow{
		Name: "gated-artifact-flow",
		Steps: []Step{
			{
				Name:   "build-step",
				Run:    "echo \"hello world\" > test-artifact.txt",
				Prompt: "Approve artifact compilation?",
				Artifacts: []string{
					"test-artifact.txt",
				},
				WorkingDir: tmpDir,
			},
		},
	}

	ctx := NewContext(tmpDir, "main")
	executor := NewExecutor(flow, ctx)
	executor.AutoConfirm = true
	executor.Silent = true

	results, err := executor.Execute()
	if err != nil {
		t.Fatalf("Executor.Execute failed: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	res := results[0]
	if !res.Success {
		t.Errorf("step expected to succeed, error: %v", res.Error)
	}
	if !res.PromptApproved {
		t.Errorf("expected step to be prompt approved")
	}
	if len(res.CollectedFiles) != 1 || res.CollectedFiles[0] != "test-artifact.txt" {
		t.Errorf("expected test-artifact.txt to be collected, got: %v", res.CollectedFiles)
	}

	// Clean up artifacts
	_ = os.RemoveAll(filepath.Join(GetArtifactsBaseDir(), flow.Name))
}
