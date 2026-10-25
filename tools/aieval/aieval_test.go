package main

import (
	"strings"
	"testing"
)

func TestTaskFileParsesOrderedOps(t *testing.T) {
	task, err := parseTask("x", "---\ntitle: T\nneeds: modal\ntext: Name\nclick: Save\nnot-text: Name\nsource: OpenWindow\n---\nMake a page.\n")
	if err != nil {
		t.Fatal(err)
	}
	if task.Title != "T" || task.Needs != "modal" || task.Request != "Make a page." || len(task.Source) != 1 {
		t.Fatalf("%+v", task)
	}
	if len(task.Ops) != 3 || task.Ops[1] != (Op{"click", "Save"}) {
		t.Fatalf("ops out of order: %+v", task.Ops)
	}
	if _, err := parseTask("x", "---\ntitle: T\nbogus: 1\n---\nr"); err == nil {
		t.Fatal("an unknown directive must be an error")
	}
}

func TestExtractGoTakesTheLongestBlockAndAnUnterminatedOne(t *testing.T) {
	if got, _ := extractGo("a\n```go\nx\n```\nb\n```go\nlonger\n```"); got != "longer\n" {
		t.Fatalf("got %q", got)
	}
	if got, ok := extractGo("```go\npackage main\nfunc"); !ok || !strings.HasPrefix(got, "package main") {
		t.Fatalf("a reply cut off by the token limit must still be repairable: %q", got)
	}
}

func TestModelConfigAndFlag(t *testing.T) {
	m, err := parseModelFlag(`fast=https://api.example.com/v1,MY_KEY,{"thinking":{"type":"disabled"}}`)
	if err != nil || m.BaseURL != "https://api.example.com/v1" || m.KeyEnv != "MY_KEY" || !strings.Contains(m.Extra, "disabled") {
		t.Fatalf("%+v %v", m, err)
	}
	if _, err := parseModelFlag("nokey=https://x"); err == nil {
		t.Fatal("a model without a key variable must be refused")
	}
}
