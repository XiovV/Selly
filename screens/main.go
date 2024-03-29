package screens

import (
	"encoding/json"
	"fmt"
	"github.com/XiovV/selly-client/data"
	"github.com/XiovV/selly-client/friendslist"
	"github.com/XiovV/selly-client/jwt"
	"github.com/XiovV/selly-client/ws"
	"github.com/gdamore/tcell/v2"
	"github.com/gorilla/websocket"
	"github.com/rivo/tview"
	"golang.design/x/clipboard"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"time"
)

const (
	messageType = "message"
)

type Main struct {
	app               *tview.Application
	internalTextView  *tview.TextView
	messageInput      *tview.InputField
	commandBox        *tview.InputField
	friendsList       *friendslist.List
	ws                *websocket.Conn
	db                *data.Repository
	localUser         *data.LocalUser
	selectedFriend    *data.Friend
	addFriendBtn      *tview.Button
	deleteFriendBtn   *tview.Button
	editFriendBtn     *tview.Button
	myDetailsButton   *tview.Button
	isConnectionAlive bool
}

func NewMainScreen(app *tview.Application, db *data.Repository) *Main {
	main := &Main{
		app:              app,
		internalTextView: tview.NewTextView(),
		messageInput:     tview.NewInputField(),
		commandBox:       tview.NewInputField(),
		friendsList:      friendslist.New(),
		addFriendBtn:     tview.NewButton("Add Friend"),
		deleteFriendBtn:  tview.NewButton("Delete Friend"),
		editFriendBtn:    tview.NewButton("Edit Friend"),
		myDetailsButton:  tview.NewButton("My Details"),
		db:               db,
	}

	localUser, err := main.db.GetLocalUserInfo()
	if err != nil {
		log.Fatalf("couldn't get local user info: %s", err)
	}

	main.localUser = &localUser

	main.internalTextView.SetDynamicColors(true).
		SetRegions(true).
		SetWordWrap(true).SetBorder(true)
	main.internalTextView.ScrollToEnd()

	main.messageInput.SetDoneFunc(main.sendMessage).SetPlaceholder("Message")
	main.messageInput.SetBorder(true)

	main.friendsList.SetSelectedFunc(main.onFriendSelect)

	main.addFriendBtn.SetSelectedFunc(main.showAddFriendScreen)
	main.deleteFriendBtn.SetSelectedFunc(main.showDeleteFriendScreen)
	main.editFriendBtn.SetSelectedFunc(main.showEditFriendScreen)
	main.myDetailsButton.SetSelectedFunc(main.showMyDetailsScreen)

	main.addFriendBtn.SetBorder(true)
	main.deleteFriendBtn.SetBorder(true)
	main.editFriendBtn.SetBorder(true)
	main.myDetailsButton.SetBorder(true)

	main.loadFriendsList()
	main.loadFirstFriend()

	err = main.validateJWT()
	if err != nil {
		panic(err)
	}

	main.loadMissedMessages()

	connection, _ := ws.NewWebsocketClient(localUser.JWT)

	main.ws = connection

	if connection != nil {
		main.isConnectionAlive = true
	}

	go main.listenForMessages()

	return main
}

type Payload struct {
	Type string
	Msg  interface{}
}

// TODO: consider optimising this entire method
func (s *Main) loadMissedMessages() {
	messages := s.getMissedMessages()

	for i := len(messages) - 1; i >= 0; i-- {
		err := s.db.StoreMessage(messages[i].Sender, messages[i])
		if err != nil {
			log.Fatalf("couldn't store message: %s", err)
		}

		if messages[i].Sender == s.selectedFriend.SellyID {
			messages[i].Sender = s.selectedFriend.Username
			s.addMessage(messages[i])

			s.db.UpdateLastInteraction(s.selectedFriend.SellyID)
		} else {
			friend, _ := s.db.GetFriendDataBySellyID(messages[i].Sender)

			s.friendsList.IncrementUnreadMessages(friend.Username)

			s.db.UpdateLastInteraction(friend.SellyID)
		}
	}
}

func (s *Main) getMissedMessages() []data.Message {
	req, _ := http.NewRequest(http.MethodGet, "http://localhost:8082/v1/users/missed-messages", nil)

	req.Header.Add("Authorization", "Bearer "+s.localUser.JWT)

	client := &http.Client{}
	r, _ := client.Do(req)

	var response struct {
		Messages []data.Message `json:"messages"`
	}

	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&response)
	if err != nil {
		log.Fatal(err)
	}

	return response.Messages

}

func (s *Main) deleteFriend(username string) {
	s.friendsList.RemoveFriend(username)

	err := s.db.DeleteFriendByUsername(username)
	if err != nil {
		panic(err)
	}
}

func (s *Main) showDeleteFriendScreen() {
	modal := tview.NewModal().
		SetText(fmt.Sprintf("Are you sure that you would like to remove %s from your friend's list?", s.selectedFriend.Username)).
		AddButtons([]string{"Yes", "No"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			if buttonLabel == "Yes" {
				s.deleteFriend(s.selectedFriend.Username)
			}

			s.app.SetRoot(s.Render(), true)
		})

	s.app.SetRoot(modal, true)
}

func (s *Main) showMyDetailsScreen() {
	modal := tview.NewModal().SetText(fmt.Sprintf("Your SellyID is: %s\n\n Your seed is: %s", s.localUser.SellyID, s.localUser.Seed)).
		AddButtons([]string{"Copy SellyID", "Copy Seed", "Export Account", "Back"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			if buttonLabel == "Back" {
				s.app.SetRoot(s.Render(), true)
			}

			err := clipboard.Init()
			if err != nil {
				//TODO: handle this panic gracefully
				panic(err)
			}

			if buttonLabel == "Copy SellyID" {
				clipboard.Write(clipboard.FmtText, []byte(s.localUser.SellyID))
			}

			if buttonLabel == "Copy Seed" {
				clipboard.Write(clipboard.FmtText, []byte(s.localUser.Seed))
			}

			if buttonLabel == "Export Account" {
				s.exportAccount()
			}

			s.app.SetRoot(s.Render(), true)
		})

	s.app.SetRoot(modal, true)
}

func (s *Main) exportAccount() {
	var acc struct {
		ID   string `json:"id"`
		Seed string `json:"seed"`
	}

	acc.ID = s.localUser.SellyID
	acc.Seed = s.localUser.Seed

	file, _ := json.MarshalIndent(acc, "", " ")
	ioutil.WriteFile("account.json", file, 0644)
}

func (s *Main) showEditFriendScreen() {
	if s.selectedFriend != nil {
		form := tview.NewForm().
			AddInputField("Custom username", s.selectedFriend.Username, 0, nil, nil).
			AddInputField("SellyID", s.selectedFriend.SellyID, 64, nil, nil)

		form.AddButton("Save", func() {
			usernameField := form.GetFormItem(0).(*tview.InputField)
			sellyIDField := form.GetFormItem(1).(*tview.InputField)

			isUsernameValid, err := validateUsernameInput(usernameField)
			if !isUsernameValid {
				usernameField.SetText("")
				usernameField.SetPlaceholder(err)
			}

			isSellyIDValid, err := validateSellyID(sellyIDField)
			if !isSellyIDValid {
				sellyIDField.SetText("")
				sellyIDField.SetPlaceholder(err)
			}

			if isUsernameValid && isSellyIDValid {
				s.editFriend(usernameField.GetText(), sellyIDField.GetText())

				s.app.SetRoot(s.Render(), true)
			}
		})

		form.AddButton("Cancel", func() {
			s.app.SetRoot(s.Render(), true)
		})

		form.SetBorder(true).SetTitle("Edit Friend").SetTitleAlign(tview.AlignLeft)
		s.app.SetRoot(form, true)
	}
}

func (s *Main) editFriend(username, sellyID string) {
	s.friendsList.EditFriendText(s.selectedFriend.Username, username, sellyID)

	err := s.db.EditFriend(s.selectedFriend.SellyID, sellyID, username)
	if err != nil {
		panic(err)
	}
}

func (s *Main) showAddFriendScreen() {
	form := tview.NewForm().
		AddInputField("Custom username", "", 0, nil, nil).
		AddInputField("SellyID", "", 64, nil, nil)

	form.AddButton("Save", func() {
		usernameField := form.GetFormItem(0).(*tview.InputField)
		sellyIDField := form.GetFormItem(1).(*tview.InputField)

		isUsernameValid, err := validateUsernameInput(usernameField)
		if !isUsernameValid {
			usernameField.SetText("")
			usernameField.SetPlaceholder(err)
		}

		isSellyIDValid, err := validateSellyID(sellyIDField)
		if !isSellyIDValid {
			sellyIDField.SetText("")
			sellyIDField.SetPlaceholder(err)
		}

		if isUsernameValid && isSellyIDValid {
			s.addFriend(usernameField.GetText(), sellyIDField.GetText())

			s.app.SetRoot(s.Render(), true)

			s.db.UpdateLastInteraction(sellyIDField.GetText())
		}
	})

	form.AddButton("Cancel", func() {
		s.app.SetRoot(s.Render(), true)
	})

	form.SetBorder(true).SetTitle("Add Friend").SetTitleAlign(tview.AlignLeft)
	s.app.SetRoot(form, true)
}

func (s *Main) addFriend(username, sellyID string) {
	s.friendsList.AddFriend(username, sellyID)

	err := s.db.AddFriend(sellyID, username)
	if err != nil {
		panic(err)
	}
}

func (s *Main) validateJWT() error {
	if s.localUser.JWT == "" {
		hashedSeed := s.localUser.GetHashedSeed()

		token, err := s.getNewToken(hashedSeed)
		if err != nil {
			return err
		}

		err = s.db.UpdateJWT(token)
		if err != nil {
			return err
		}

		s.localUser.JWT = token

		return nil
	}

	if jwt.IsExpired(s.localUser.JWT) {
		token, err := s.refreshToken(s.localUser.JWT)
		if err != nil {
			return err
		}

		err = s.db.UpdateJWT(token)
		if err != nil {
			return err
		}

		s.localUser.JWT = token
	}

	return nil
}

func (s *Main) getNewToken(sellyId string) (string, error) {
	req, err := http.NewRequest(http.MethodGet, "http://localhost:8082/v1/users/token?id="+sellyId, nil)
	if err != nil {
		return "", err
	}

	client := &http.Client{}
	r, err := client.Do(req)
	if err != nil {
		return "", err
	}

	var response struct {
		AccessToken string `json:"access_token"`
	}

	decoder := json.NewDecoder(r.Body)
	err = decoder.Decode(&response)
	if err != nil {
		return "", err
	}

	return response.AccessToken, nil
}

func (s *Main) refreshToken(jwt string) (string, error) {
	req, err := http.NewRequest(http.MethodGet, "http://localhost:8082/v1/users/refresh-token", nil)
	if err != nil {
		panic(err)
	}

	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", jwt))
	client := &http.Client{}
	r, err := client.Do(req)

	if err != nil {
		return "", err
	}

	var response struct {
		AccessToken string `json:"access_token"`
	}

	decoder := json.NewDecoder(r.Body)
	err = decoder.Decode(&response)
	if err != nil {
		return "", err
	}

	return response.AccessToken, nil
}

func (s *Main) loadFirstFriend() {
	firstFriend := s.friendsList.GetFirst()

	if firstFriend == nil {
		return
	}

	s.friendsList.SetCurrentFriend(firstFriend)

	s.onFriendSelect(firstFriend)
}

func (s *Main) loadFriendsList() {
	friends, err := s.db.GetFriendsSorted()
	if err != nil {
		log.Fatalf("couldn't fetch friends: %s", err)
	}

	for _, friend := range friends {
		s.friendsList.AddFriend(friend.Username, friend.SellyID)

		unreadMessagesCount := s.db.GetCountOfUnreadMessages(friend.SellyID)

		s.friendsList.SetUnreadCounter(friend.Username, unreadMessagesCount)
	}
}

func (s *Main) onFriendSelect(node *tview.TreeNode) {
	s.friendsList.SanitizeNode(node)

	userParts := strings.Split(node.GetText(), " ")

	friendData, err := s.db.GetFriendDataByUsername(userParts[0])
	if err != nil {
		log.Fatalf("couldn't get friend info: %s", err)
	}

	s.selectedFriend = &friendData

	s.internalTextView.SetTitle(friendData.Username)
	s.internalTextView.SetText("")

	s.db.SetRead(friendData.SellyID)

	s.loadMessages()
}

func (s *Main) loadMessages() {
	messages, err := s.db.GetMessages(s.selectedFriend.SellyID)
	if err != nil {
		log.Fatalf("couldn't get messages: %s", err)
	}

	for _, message := range messages {
		if message.Sender == s.localUser.SellyID {
			message.Sender = "You"
		} else {
			friend, err := s.db.GetFriendDataBySellyID(message.Sender)
			if err != nil {
				log.Fatalf("couldn't find friend: %s", err)
			}

			message.Sender = friend.Username
		}

		s.addMessage(message)
	}
}

func (s *Main) retryConnection() {
	time.Sleep(1 * time.Second)

	s.validateJWT()
	conn, _ := ws.NewWebsocketClient(s.localUser.JWT)
	if conn == nil {
		s.listenForMessages()
		return
	}

	s.ws = conn
	s.isConnectionAlive = true
	s.addSuccessMessage("connection restored")
	s.app.Draw()
}

func (s *Main) listenForMessages() {
	var msg json.RawMessage
	payload := Payload{Msg: &msg}

	if !s.isConnectionAlive {
		s.retryConnection()
	}

	for {
		err := s.ws.ReadJSON(&payload)
		if err != nil {
			if s.isConnectionAlive {
				s.addErrorMessage("connection lost")
				s.app.Draw()
			}

			s.isConnectionAlive = false
			s.listenForMessages()
			return
		}

		if s.isConnectionAlive {
			switch payload.Type {
			case messageType:
				s.readIncomingMessage(msg)
			default:
				log.Fatal("unknown message type")
			}
		}
	}
}

func (s *Main) readIncomingMessage(msg json.RawMessage) {
	var message data.Message

	if err := json.Unmarshal(msg, &message); err != nil {
		log.Fatal(err)
	}

	friendData, _ := s.db.GetFriendDataBySellyID(message.Sender)

	if message.Sender == s.selectedFriend.SellyID {
		message.Read = 1

		err := s.db.StoreMessage(message.Sender, message)
		if err != nil {
			log.Fatalf("couldn't store message: %s", err)
		}

		message.Sender = friendData.Username

		s.addMessage(message)

		s.app.Draw()
	} else {
		message.Read = 0

		err := s.db.StoreMessage(message.Sender, message)
		if err != nil {
			log.Fatalf("couldn't store message: %s", err)
		}

		s.friendsList.IncrementUnreadMessages(friendData.Username)

		s.app.Draw()
	}

	s.db.UpdateLastInteraction(friendData.SellyID)
}

func (s *Main) sendMessage(key tcell.Key) {
	if key == tcell.KeyEnter && s.messageInput.GetText() != "" && s.isConnectionAlive {
		s.validateJWT()

		message := data.Message{
			Sender:   s.localUser.SellyID,
			Receiver: s.selectedFriend.SellyID,
			Message:  s.messageInput.GetText(),
		}

		payload := Payload{
			Type: messageType,
			Msg:  message,
		}

		s.ws.WriteJSON(payload)

		message.DateCrated = time.Now().Unix()
		message.Read = 1

		err := s.db.StoreMessage(s.selectedFriend.SellyID, message)
		if err != nil {
			log.Fatalf("couldn't store message: %s", err)
		}

		message.Sender = "You"
		s.addMessage(message)
		s.messageInput.SetText("")

		s.db.UpdateLastInteraction(s.selectedFriend.SellyID)
		s.friendsList.MoveToTop(s.selectedFriend.Username)
	}
}

func (s *Main) addErrorMessage(message string) {
	fmt.Fprintf(s.internalTextView, "[#ffffff]Error: [#ff0000]%s\n", message)
}

func (s *Main) addSuccessMessage(message string) {
	fmt.Fprintf(s.internalTextView, "[#ffffff]Success:[#00ff00] %s\n", message)
}

func (s *Main) addMessage(message data.Message) {
	fmt.Fprintf(s.internalTextView, "[#ffffff]%s: %s\n", message.Sender, message.Message)
}

func (s *Main) Render() tview.Primitive {
	return tview.NewFlex().
		AddItem(s.friendsList.GetTreeView(), 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(s.internalTextView, 0, 8, false).
			AddItem(s.messageInput, 3, 1, false).
			AddItem(tview.NewFlex().SetDirection(tview.FlexColumn).
				AddItem(s.addFriendBtn, 0, 1, false).
				AddItem(s.deleteFriendBtn, 0, 1, false).
				AddItem(s.editFriendBtn, 0, 1, false).
				AddItem(s.myDetailsButton, 0, 1, false), 0, 1, false), 0, 2, false)
}
