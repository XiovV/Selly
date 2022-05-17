package screens

import (
	"github.com/rivo/tview"
)

func ShowAddFriendScreen(app *tview.Application, main *Main) {
	form := tview.NewForm().
		AddInputField("Custom username", "", 0, nil, nil).
		AddInputField("SellyID", "", 64, nil, nil)

	form.AddButton("Save", func() {
		usernameField := form.GetFormItem(0).(*tview.InputField)
		sellyIDField := form.GetFormItem(1).(*tview.InputField)

		if len(usernameField.GetText()) < 1 {
			usernameField.SetText("")
			usernameField.SetPlaceholder("username must be at least 1 character long")
		}

		if len(usernameField.GetText()) > 64 {
			usernameField.SetText("")
			usernameField.SetPlaceholder("username must be at most 64 characters long")
		}

		if len(sellyIDField.GetText()) != 64 {
			sellyIDField.SetText("")
			sellyIDField.SetPlaceholder("Selly ID must be exactly 64 characters long")
		}

		if len(usernameField.GetText()) < 64 && len(sellyIDField.GetText()) == 64 {
			main.addFriend(usernameField.GetText(), sellyIDField.GetText())

			app.SetRoot(main.Render(), true)
		}
	})

	form.AddButton("Cancel", func() {
		app.SetRoot(main.Render(), true)
	})

	form.SetBorder(true).SetTitle("Add Friend").SetTitleAlign(tview.AlignLeft)
	app.SetRoot(form, true)
}
