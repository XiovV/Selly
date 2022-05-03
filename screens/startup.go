package screens

import (
	"github.com/XiovV/selly-client/data"
	"github.com/rivo/tview"
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

func (s *Startup) Render() tview.Primitive {
	s.pages.AddPage("new-id", s.generateAccountScreen.Render(), false, false)

	return s.pages.AddPage("main-modal", tview.NewModal().
		SetText("Would you like to create a new ID or restore an existing one?").
		AddButtons([]string{"Create New", "Restore"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			if buttonLabel == "Create New" {
				s.pages.SwitchToPage("new-id")
			}
		}), false, true)
}
