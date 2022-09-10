package screens

import (
	"github.com/XiovV/selly-client/data"
	"github.com/XiovV/selly-client/ws"
	"github.com/rivo/tview"
)

type App struct {
	app           *tview.Application
	db            *data.Repository
	startupScreen *Startup
	mainScreen    *Main
}

func NewApp(app *tview.Application, db *data.Repository) *App {
	return &App{app: app, db: db}
}

func (a *App) showConnectionFailedMessage() tview.Primitive {
	modal := tview.NewModal().
		SetText("Connection could not be established with the server, please check your internet connection or try again later.").
		AddButtons([]string{"Okay", "Quit"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			if buttonLabel == "Okay" {
				a.app.SetRoot(NewMainScreen(a.app, a.db).Render(), true)
			}

			if buttonLabel == "Quit" {
				a.app.Stop()
			}
		})

	return modal
}

func (a *App) Start() tview.Primitive {
	if a.isAccountSetUp() {
		if !ws.Ping() {
			return a.showConnectionFailedMessage()
		}

		return NewMainScreen(a.app, a.db).Render()
	}

	return NewStartupScreen(a.app, a.db).Render()
}

func (a *App) isAccountSetUp() bool {
	_, err := a.db.GetLocalUserInfo()
	if err != nil {
		return false
	}

	return true
}
