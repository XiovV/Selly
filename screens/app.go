package screens

import (
	"github.com/XiovV/selly-client/data"
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

func (a *App) Start() tview.Primitive {
	if a.isAccountSetUp() {
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
