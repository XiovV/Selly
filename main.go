package main

import (
	"github.com/XiovV/selly-client/data"
	"github.com/XiovV/selly-client/screens"
	_ "github.com/mattn/go-sqlite3"
	"github.com/rivo/tview"
	"os"
)

func main() {
	var fileName string

	if os.Args[1] == "1" {
		fileName = "u1.db"
	} else if os.Args[1] == "2" {
		fileName = "u2.db"
	} else if os.Args[1] == "3" {
		fileName = "u3.db"
	}

	db := data.NewRepository(fileName)

	app := tview.NewApplication()
	root := screens.NewApp(app, db)

	if err := app.SetRoot(root.Start(), true).EnableMouse(true).Run(); err != nil {
		panic(err)
	}
}
