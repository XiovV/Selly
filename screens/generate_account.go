package screens

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"github.com/XiovV/selly-client/data"
	"github.com/rivo/tview"
	"io/ioutil"
	"math/rand"
	"strings"
	"time"
)

type GenerateAccount struct {
	app       *tview.Application
	pages     *tview.Pages
	seedWords []string
	db        *data.Repository
}

func NewGenerateAccountScreen(app *tview.Application, db *data.Repository) *GenerateAccount {
	return &GenerateAccount{
		app:       app,
		pages:     tview.NewPages(),
		seedWords: []string{"apple", "banana", "car", "orange", "book", "monitor", "computer", "poster", "box", "fan", "card", "desk", "table"},
		db:        db,
	}
}

func (s *GenerateAccount) Render() tview.Primitive {
	id, seed := s.generateNewID()
	seedStr := strings.Join(seed, ", ")

	return s.pages.AddPage("main-modal", tview.NewModal().
		SetText(fmt.Sprintf("Your new ID is: %s\n\n Your seed is: %s\n\nWrite this seed down or export it so you can restore your account later!", id, seedStr)).
		AddButtons([]string{"Next", "Export"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			switch buttonLabel {
			case "Next":
				s.persistID(id, seedStr)
				s.app.SetRoot(NewMainScreen(s.app, s.db).Render(), true)
			case "Export":
				s.persistID(id, seedStr)
				s.exportAccount(id, strings.Join(seed, ", "))
				s.app.SetRoot(NewMainScreen(s.app, s.db).Render(), true)
			}
		}), false, true)
}

func (s *GenerateAccount) exportAccount(id, seed string) {
	var acc struct {
		ID   string `json:"id"`
		Seed string `json:"seed"`
	}

	acc.ID = id
	acc.Seed = seed

	file, _ := json.MarshalIndent(acc, "", " ")
	ioutil.WriteFile("account.json", file, 0644)
}

func (s *GenerateAccount) generateNewID() (string, []string) {
	rand.Seed(time.Now().UnixNano())

	seed := []string{}

	for i := 0; i < 5; i++ {
		randomIndex := rand.Intn(len(s.seedWords) - 1 + 1)
		seed = append(seed, s.seedWords[randomIndex])
	}

	hashedSeed := sha256.Sum256([]byte(strings.Join(seed, "")))

	sellyId := sha256.Sum256([]byte(fmt.Sprintf("%x", hashedSeed[:])))

	return fmt.Sprintf("%x", sellyId[:]), seed
}

func (s *GenerateAccount) generateIDFromSeed(seed []string) string {
	hashedSeed := sha256.Sum256([]byte(strings.Join(seed, "")))

	sellyId := sha256.Sum256([]byte(fmt.Sprintf("%x", hashedSeed[:])))

	return fmt.Sprintf("%x", sellyId[:])
}

func (s *GenerateAccount) persistID(id, seed string) error {
	return s.db.StoreLocalUserInfo(id, seed)
}
