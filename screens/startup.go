package screens

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"github.com/XiovV/selly-client/data"
	"github.com/rivo/tview"
	"io/ioutil"
	"strings"
)

const (
	seedLength = 5
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

func (s *Startup) showImportFromFileForm() {
	var acc struct {
		ID   string `json:"id"`
		Seed string `json:"seed"`
	}

	form := tview.NewForm().
		AddInputField("Path:", "", 0, nil, nil)

	pathInput := form.GetFormItem(0).(*tview.InputField)
	pathInput.SetPlaceholder("enter the path to your account json file")

	form.AddButton("Restore", func() {
		content, err := ioutil.ReadFile(pathInput.GetText())
		if err != nil {
			pathInput.SetText("")
			pathInput.SetPlaceholder(err.Error())
			return
		}

		err = json.Unmarshal(content, &acc)
		if err != nil {
			pathInput.SetText("")
			pathInput.SetPlaceholder("json file is of invalid format")
			return
		}

		containsComma := strings.Contains(acc.Seed, ", ")

		if !containsComma {
			pathInput.SetText("")
			pathInput.SetPlaceholder("words must be comma separated")
			return
		}

		seed := strings.Split(acc.Seed, ", ")

		if len(seed) != seedLength && containsComma {
			pathInput.SetText("")
			pathInput.SetPlaceholder(fmt.Sprintf("the seed must be %d words, got: %d", seedLength, len(seed)))
			return
		}

		id := s.generateIDFromSeed(seed)

		s.db.StoreLocalUserInfo(id, acc.Seed)
		s.app.SetRoot(NewMainScreen(s.app, s.db).Render(), true)
	})

	form.AddButton("Cancel", func() {
		s.app.SetRoot(s.Render(), true)
	})

	s.app.SetRoot(form, true)
}

func (s *Startup) showManualAccountRestoreForm() {
	form := tview.NewForm().
		AddInputField("Seed:", "", 0, nil, nil)

	seedInput := form.GetFormItem(0).(*tview.InputField)
	seedInput.SetPlaceholder("enter your seed, words must be comma separated")

	form.AddButton("Restore", func() {
		if seedInput.GetText() == "" {
			seedInput.SetText("")
			seedInput.SetPlaceholder("input must not be empty")
			return
		}

		containsComma := strings.Contains(seedInput.GetText(), ", ")

		if !containsComma {
			seedInput.SetText("")
			seedInput.SetPlaceholder("words must be comma separated")
			return
		}

		seed := strings.Split(seedInput.GetText(), ", ")

		if len(seed) != seedLength && containsComma {
			seedInput.SetText("")
			seedInput.SetPlaceholder(fmt.Sprintf("the seed must be %d words, got: %d", seedLength, len(seed)))
			return
		}

		id := s.generateIDFromSeed(seed)

		s.db.StoreLocalUserInfo(id, seedInput.GetText())
		s.app.SetRoot(NewMainScreen(s.app, s.db).Render(), true)
	})

	form.AddButton("Cancel", func() {
		s.app.SetRoot(s.Render(), true)
	})

	s.app.SetRoot(form, true)
}

func (s *Startup) generateIDFromSeed(seed []string) string {
	hashedSeed := sha256.Sum256([]byte(strings.Join(seed, "")))

	sellyId := sha256.Sum256([]byte(fmt.Sprintf("%x", hashedSeed[:])))

	return fmt.Sprintf("%x", sellyId[:])
}

func (s *Startup) showRestoreAccountModal() {
	modal := tview.NewModal().
		SetText("Would you like to restore your account manually or by importing it from a file?").
		AddButtons([]string{"Manual", "Import"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			if buttonLabel == "Manual" {
				s.showManualAccountRestoreForm()
			}

			if buttonLabel == "Import" {
				s.showImportFromFileForm()
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
