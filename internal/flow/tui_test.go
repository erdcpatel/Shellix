package flow

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func TestFlowTUIModel(t *testing.T) {
	yamlData := []byte(`
name: tui-test-flow
steps:
  - name: step-first
    run: echo "hello tui"
  - name: step-second
    run: echo "second step"
`)
	f, err := Parse(yamlData)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	ctx := NewContext(".", "main")
	m := newFlowTUIModel(f, ctx)

	// Verify initial state
	if len(m.steps) != 2 {
		t.Fatalf("expected 2 steps, got %d", len(m.steps))
	}
	if m.steps[0].Status != StepStatusPending {
		t.Errorf("expected pending status, got %v", m.steps[0].Status)
	}

	// Test Window resize
	mModel, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 35})
	m = mModel.(flowTUIModel)

	if m.width != 120 || m.height != 35 {
		t.Errorf("expected width 120 height 35, got %dx%d", m.width, m.height)
	}

	// Test Step Start Event
	mModel, _ = m.Update(flowEventMsg(FlowEvent{
		Type:      EventStepStart,
		StepName:  "step-first",
		StepIndex: 0,
	}))
	m = mModel.(flowTUIModel)
	if m.steps[0].Status != StepStatusRunning {
		t.Errorf("expected step 0 to be running, got %v", m.steps[0].Status)
	}

	// Test Step Done Event
	res := ExecutionResult{
		StepName: "step-first",
		Duration: 15 * time.Millisecond,
		Success:  true,
		Output:   "hello tui output",
	}
	mModel, _ = m.Update(flowEventMsg(FlowEvent{
		Type:      EventStepDone,
		StepName:  "step-first",
		StepIndex: 0,
		Result:    &res,
		Duration:  res.Duration,
	}))
	m = mModel.(flowTUIModel)
	if m.steps[0].Status != StepStatusDone {
		t.Errorf("expected step 0 to be done, got %v", m.steps[0].Status)
	}
	if m.steps[0].Output != "hello tui output" {
		t.Errorf("expected output to be stored, got %s", m.steps[0].Output)
	}

	// Test Key Navigation (down / up)
	mModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	m = mModel.(flowTUIModel)
	if m.cursor != 1 {
		t.Errorf("expected cursor to move to 1, got %d", m.cursor)
	}

	mModel, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	m = mModel.(flowTUIModel)
	if m.cursor != 0 {
		t.Errorf("expected cursor to move back to 0, got %d", m.cursor)
	}

	// Test View rendering
	rendered := m.View()
	if !strings.Contains(rendered, "tui-test-flow") {
		t.Errorf("expected flow name in rendered view")
	}
	if !strings.Contains(rendered, "step-first") {
		t.Errorf("expected step-first in rendered view")
	}
}

func TestRenderSnapshot(t *testing.T) {
	yamlData := []byte(`
name: verification-pipeline
steps:
  - name: check-branch
    run: echo "Current branch is main"
  - name: run-go-tests
    parallel: true
    run: go test ./internal/flow/...
  - name: search-check
    parallel: true
    run: ./bin/sx search "type Engine struct" --max 3
  - name: package-binary
    run: make build
`)
	f, err := Parse(yamlData)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}

	ctx := NewContext(".", "main")
	snap := RenderSnapshot(f, ctx)
	if !strings.Contains(snap, "verification-pipeline") {
		t.Errorf("expected flow name in snapshot")
	}
	if os.Getenv("PRINT_SNAPSHOT") == "1" {
		fmt.Print(snap)
	}
}
