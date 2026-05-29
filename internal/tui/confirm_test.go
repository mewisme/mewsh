package tui

import "testing"

func TestConfirmYNPrompt(t *testing.T) {
	t.Parallel()
	if got := confirmYNPrompt(true); got != "(Y/n)" {
		t.Fatalf("default yes: got %q", got)
	}
	if got := confirmYNPrompt(false); got != "(y/N)" {
		t.Fatalf("default no: got %q", got)
	}
}

func TestConfirmKeyEnterDefault(t *testing.T) {
	t.Parallel()
	yes, cancel, handled := confirmKey("enter", true)
	if !handled || !yes || cancel {
		t.Fatalf("enter default yes: yes=%v cancel=%v handled=%v", yes, cancel, handled)
	}
	yes, cancel, handled = confirmKey("enter", false)
	if !handled || yes || !cancel {
		t.Fatalf("enter default no: yes=%v cancel=%v handled=%v", yes, cancel, handled)
	}
}
