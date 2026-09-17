package main

import "testing"

func TestSplitCLIArgsNegativeFlagValue(t *testing.T) {
	paths, flags := splitCLIArgs([]string{"a.funscript", "b.funscript", "--max-lag-ms", "-500"})
	if len(paths) != 2 || paths[0] != "a.funscript" || paths[1] != "b.funscript" {
		t.Fatalf("paths=%v", paths)
	}
	if len(flags) != 2 || flags[0] != "--max-lag-ms" || flags[1] != "-500" {
		t.Fatalf("flags=%v", flags)
	}
}

func TestSplitCLIArgsEqualsForm(t *testing.T) {
	paths, flags := splitCLIArgs([]string{"--max-lag-ms=-200", "ref.funscript", "cand.funscript"})
	if len(paths) != 2 {
		t.Fatalf("paths=%v", paths)
	}
	if len(flags) != 1 || flags[0] != "--max-lag-ms=-200" {
		t.Fatalf("flags=%v", flags)
	}
}

func TestSplitCLIArgsDoesNotSwallowNextOption(t *testing.T) {
	paths, flags := splitCLIArgs([]string{"--lag-step-ms", "--max-lag-ms", "100", "a.funscript", "b.funscript"})
	// --lag-step-ms has no value token; next is another option
	if len(flags) < 2 || flags[0] != "--lag-step-ms" || flags[1] != "--max-lag-ms" {
		t.Fatalf("flags=%v", flags)
	}
	if len(paths) != 2 {
		t.Fatalf("paths=%v", paths)
	}
}

func TestSplitCLIArgsDoubleDash(t *testing.T) {
	paths, flags := splitCLIArgs([]string{"--max-lag-ms", "100", "--", "-odd.funscript", "b.funscript"})
	if len(flags) != 2 {
		t.Fatalf("flags=%v", flags)
	}
	if len(paths) != 2 || paths[0] != "-odd.funscript" {
		t.Fatalf("paths=%v", paths)
	}
}
