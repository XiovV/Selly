package screens

import (
	"fmt"
	"github.com/XiovV/selly-client/data"
	"github.com/rivo/tview"
	"strings"
)

type Startup struct {
	app                   *tview.Application
	pages                 *tview.Pages
	generateAccountScreen *GenerateAccount
	db                    *data.Repository
}

func NewStartupScreen(app *tview.Application, db *data.Repository) *Startup {
	pages := tview.NewPages()

	return &Startup{
		app:                   app,
		pages:                 pages,
		generateAccountScreen: NewGenerateAccountScreen(app, db),
		db:                    db,
	}
}

func (s *Startup) showManualAccountRestoreForm() {
	form := tview.NewForm().
		AddInputField("Seed:", "", 0, nil, nil)

	seedInput := form.GetFormItem(0).(*tview.InputField)
	seedInput.SetPlaceholder("enter your seed, words must be comma separated")

	form.AddButton("Restore", func() {
		if len(seedInput.GetText()) == 0 {
			seedInput.SetText("")
			seedInput.SetPlaceholder("input must not be empty")
		}

		if !strings.Contains(seedInput.GetText(), ", ") {
			seedInput.SetText("")
			seedInput.SetPlaceholder("words must be comma separated")
		}

		seed := strings.Split(seedInput.GetText(), ", ")

		if len(seed) != 5 {
			seedInput.SetText("")
			seedInput.SetPlaceholder(fmt.Sprintf("the seed must be 5 words, got: %d", len(seed)))
		}
	})

	form.AddButton("Cancel", func() {
		s.app.SetRoot(s.Render(), true)
	})

	s.app.SetRoot(form, true)
}

func (s *Startup) showRestoreAccountModal() {
	modal := tview.NewModal().
		SetText("Would you like to restore your account manually or by importing it from a file?").
		AddButtons([]string{"Manual", "Import"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			if buttonLabel == "Manual" {
				s.showManualAccountRestoreForm()
			}
		})

	s.app.SetRoot(modal, true)
}

func (s *Startup) Render() tview.Primitive {
	s.pages.AddPage("new-id", s.generateAccountScreen.Render(), false, false)

	return s.pages.AddPage("main-modal", tview.NewModal().
		SetText("Would you like to create a new ID or restore an existing one?").
		AddButtons([]string{"Create New", "Restore"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			if buttonLabel == "Create New" {
				s.pages.SwitchToPage("new-id")
			}

			if buttonLabel == "Restore" {
				s.showRestoreAccountModal()
			}
		}), false, true)
}
