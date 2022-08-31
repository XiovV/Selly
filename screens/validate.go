package screens

import (
	"github.com/rivo/tview"
	"strings"
)

func validateUsernameInput(usernameField *tview.InputField) (bool, string) {
	username := usernameField.GetText()

	if len(username) < 1 {
		return false, "username must be at least 1 character long"
	}

	if len(username) > 64 {
		return false, "username must be at most 64 characters long"
	}

	if strings.Contains(username, " ") {
		return false, "username must be a single word"
	}

	return true, ""
}

func validateSellyID(sellyIDField *tview.InputField) (bool, string) {
	sellyID := sellyIDField.GetText()

	if len(sellyID) != 64 {
		return false, "Selly ID must be exactly 64 characters long"
	}

	return true, ""
}
