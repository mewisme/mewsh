package tui

// confirmYNPrompt formats the yes/no hint; the capital letter is the Enter default.
func confirmYNPrompt(defaultYes bool) string {
	if defaultYes {
		return "(Y/n)"
	}
	return "(y/N)"
}

// confirmKey resolves a key on a confirmation dialog.
// Returns confirmed, cancelled, and whether the key was handled.
func confirmKey(key string, defaultYes bool) (confirmed, cancelled, handled bool) {
	switch key {
	case "enter":
		if defaultYes {
			return true, false, true
		}
		return false, true, true
	case "y", "Y":
		return true, false, true
	case "n", "N", "esc":
		return false, true, true
	default:
		return false, false, false
	}
}

func confirmFooterBindings(defaultYes bool) []footerBinding {
	if defaultYes {
		return []footerBinding{
			{"Y/enter", "yes"},
			{"n", "no"},
			{"esc", "cancel"},
		}
	}
	return []footerBinding{
		{"y", "yes"},
		{"N/enter", "no"},
		{"esc", "cancel"},
	}
}
