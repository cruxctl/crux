package managedops

import (
	"reflect"
	"testing"

	"github.com/cruxctl/crux/pkg/cruxapi"
)

func TestResumeAgentBuildsProviderCommands(t *testing.T) {
	tests := []struct {
		name    string
		session string
		want    []string
	}{
		{name: "claude", session: "abc", want: []string{"--resume", "abc", "-p", "{prompt}"}},
		{name: "claude", session: "last", want: []string{"--continue", "-p", "{prompt}"}},
		{name: "codex", session: "abc", want: []string{"exec", "resume", "--all", "abc", "--skip-git-repo-check", "{prompt}"}},
		{name: "codex", session: "last", want: []string{"exec", "resume", "--all", "--last", "--skip-git-repo-check", "{prompt}"}},
		{name: "gemini", session: "abc", want: []string{"--skip-trust", "--resume", "abc", "-p", "{prompt}"}},
		{name: "gemini", session: "last", want: []string{"--skip-trust", "--resume", "latest", "-p", "{prompt}"}},
		{name: "kimi", session: "abc", want: []string{"--quiet", "--session", "abc", "--prompt", "{prompt}"}},
		{name: "kimi", session: "last", want: []string{"--quiet", "--continue", "--prompt", "{prompt}"}},
	}
	for _, tt := range tests {
		t.Run(tt.name+"-"+tt.session, func(t *testing.T) {
			agent, err := ResumeAgent(cruxapi.Agent{Name: tt.name}, tt.session)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(agent.Command.Args, tt.want) {
				t.Fatalf("expected args %#v, got %#v", tt.want, agent.Command.Args)
			}
		})
	}
}

func TestExecAgentBuildsInteractiveProviderCommands(t *testing.T) {
	tests := []struct {
		name    string
		session string
		args    []string
		want    []string
	}{
		{name: "claude", want: nil},
		{name: "claude", session: "last", want: []string{"--continue"}},
		{name: "claude", session: "abc", want: []string{"--resume", "abc"}},
		{name: "codex", want: nil},
		{name: "codex", session: "abc", want: []string{"resume", "--all", "abc"}},
		{name: "codex", session: "last", want: []string{"resume", "--all", "--last"}},
		{name: "gemini", want: []string{"--skip-trust"}},
		{name: "gemini", session: "last", want: []string{"--skip-trust", "--resume", "latest"}},
		{name: "kimi", session: "abc", args: []string{"--model", "moon"}, want: []string{"--session", "abc", "--model", "moon"}},
	}
	for _, tt := range tests {
		t.Run(tt.name+"-"+tt.session, func(t *testing.T) {
			agent, err := ExecAgent(cruxapi.Agent{Name: tt.name}, cruxapi.AgentExecPlanRequest{ResumeSession: tt.session, Args: tt.args})
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(agent.Command.Args, tt.want) {
				t.Fatalf("expected args %#v, got %#v", tt.want, agent.Command.Args)
			}
		})
	}
}

func TestExecAgentSubstitutesCustomPrompt(t *testing.T) {
	agent, err := ExecAgent(cruxapi.Agent{
		Name: "echo",
		Command: cruxapi.CommandSpec{
			Path: "/usr/bin/printf",
			Args: []string{"%s\n", "{prompt}"},
		},
	}, cruxapi.AgentExecPlanRequest{Prompt: "hello"})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(agent.Command.Args, []string{"%s\n", "hello"}) {
		t.Fatalf("unexpected args: %#v", agent.Command.Args)
	}
	input := ExecInput(cruxapi.Agent{Name: "echo", Command: cruxapi.CommandSpec{Args: []string{"{prompt}"}}}, cruxapi.AgentExecPlanRequest{Prompt: "hello"}, agent)
	if len(input) != 0 {
		t.Fatalf("expected no stdin input when prompt placeholder was substituted, got %#v", input)
	}
}

func TestHistoryBuildsExecutionPreviews(t *testing.T) {
	items := History([]cruxapi.Execution{
		{
			ID:        "exec_1",
			AgentName: "gemini",
			Prompt:    "line one\nline two",
			Stdout:    "answer",
			Status:    cruxapi.ExecutionSucceeded,
			QueuedAt:  cruxapi.Now(),
		},
		{
			ID:        "exec_2",
			AgentName: "codex",
			Prompt:    "other",
			Status:    cruxapi.ExecutionSucceeded,
			QueuedAt:  cruxapi.Now(),
		},
	}, "gemini")
	if len(items) != 1 {
		t.Fatalf("expected one item, got %d", len(items))
	}
	if items[0].PromptPreview != "line one | line two" {
		t.Fatalf("unexpected prompt preview: %q", items[0].PromptPreview)
	}
	if items[0].StdoutPreview != "answer" {
		t.Fatalf("unexpected stdout preview: %q", items[0].StdoutPreview)
	}
}
