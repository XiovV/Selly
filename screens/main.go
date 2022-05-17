package screens

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/XiovV/selly-client/data"
	"github.com/XiovV/selly-client/jwt"
	"github.com/XiovV/selly-client/ws"
	"github.com/gdamore/tcell/v2"
	"github.com/gorilla/websocket"
	"github.com/rivo/tview"
	"log"
	"net/http"
	"strings"
)

type Main struct {
	app              *tview.Application
	internalTextView *tview.TextView
	messageInput     *tview.InputField
	commandBox       *tview.InputField
	friendsList      *tview.TreeView
	ws               *websocket.Conn
	db               *data.Repository
	localUser        *data.LocalUser
	selectedFriend   *data.Friend
	addFriendBtn     *tview.Button
	deleteFriendBtn  *tview.Button
	editFriendBtn    *tview.Button
	myInformationBtn *tview.Button
}

func NewMainScreen(app *tview.Application, db *data.Repository) *Main {
	main := &Main{
		app:              app,
		internalTextView: tview.NewTextView(),
		messageInput:     tview.NewInputField(),
		commandBox:       tview.NewInputField(),
		friendsList:      tview.NewTreeView(),
		addFriendBtn:     tview.NewButton("Add Friend"),
		deleteFriendBtn:  tview.NewButton("Delete Friend"),
		editFriendBtn:    tview.NewButton("Edit Friend"),
		myInformationBtn: tview.NewButton("My Information"),
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

	main.friendsList.SetRoot(tview.NewTreeNode("")).SetTopLevel(1)
	main.friendsList.SetSelectedFunc(main.onFriendSelect)
	main.friendsList.SetBorder(true)
	main.friendsList.SetTitle("Friends")

	main.addFriendBtn.SetSelectedFunc(main.handleAddFriend)

	main.addFriendBtn.SetBorder(true)
	main.deleteFriendBtn.SetBorder(true)
	main.editFriendBtn.SetBorder(true)
	main.myInformationBtn.SetBorder(true)

	main.loadFriends()

	err = main.validateJWT(localUser)
	if err != nil {
		panic(err)
	}

	connection, err := ws.NewWebsocketClient(localUser.JWT)
	if err != nil {
		panic(err)
	}

	main.ws = connection

	go main.listenForMessages(connection)

	return main
}

func (s *Main) handleAddFriend() {
	s.showAddFriendScreen()
}

func (s *Main) removeFriend(command []string) error {
	if len(command) > 2 {
		return errors.New(fmt.Sprintf("remove command needs exactly 1 argument, got: %d", len(command)-1))
	}

	username := command[1]

	if len(username) < 1 {
		return errors.New("username needs to be at least one character long")
	}

	for _, friend := range s.friendsList.GetRoot().GetChildren() {
		if strings.Contains(friend.GetText(), username) {
			s.friendsList.GetRoot().RemoveChild(friend)

			err := s.db.DeleteFriendByUsername(username)
			if err != nil {
				panic(err)
			}

			return nil
		}
	}

	return errors.New(" couldn't find that user")
}

func (s *Main) showAddFriendScreen() {
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
			s.addFriend(usernameField.GetText(), sellyIDField.GetText())

			s.app.SetRoot(s.Render(), true)
		}
	})

	form.AddButton("Cancel", func() {
		s.app.SetRoot(s.Render(), true)
	})

	form.SetBorder(true).SetTitle("Add Friend").SetTitleAlign(tview.AlignLeft)
	s.app.SetRoot(form, true)
}

func (s *Main) addFriend(username, sellyID string) {
	node := tview.NewTreeNode(fmt.Sprintf("%s (%s)", username, s.truncateId(sellyID)))
	s.friendsList.GetRoot().AddChild(node)

	err := s.db.AddFriend(sellyID, username)
	if err != nil {
		panic(err)
	}
}

func (s *Main) validateJWT(localUser data.LocalUser) error {
	if localUser.JWT == "" {
		token, err := s.getNewToken(localUser.SellyID)
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

	if jwt.IsTokenExpired(localUser.JWT) {
		token, err := s.refreshToken(localUser.JWT)
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

func (s *Main) loadFriends() {
	friends, err := s.db.GetFriends()
	if err != nil {
		log.Fatalf("couldn't fetch friends: %s", err)
	}

	for _, friend := range friends {
		node := tview.NewTreeNode(fmt.Sprintf("%s (%s)", friend.Username, s.truncateId(friend.SellyID)))
		s.friendsList.GetRoot().AddChild(node)
	}
}

func (s *Main) truncateId(id string) string {
	return fmt.Sprintf("%s...%s", id[:7], id[len(id)-7:])
}

func (s *Main) onFriendSelect(node *tview.TreeNode) {
	selectedUsername := node.GetText()

	userParts := strings.Split(selectedUsername, " ")

	friendData, err := s.db.GetFriendDataByUsername(userParts[0])
	if err != nil {
		log.Fatalf("couldn't get friend info: %s", err)
	}

	s.selectedFriend = &friendData

	s.internalTextView.SetTitle(friendData.Username)
	s.internalTextView.SetText("")

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

func (s *Main) listenForMessages(conn *websocket.Conn) {
	var message data.Message

	for {
		err := conn.ReadJSON(&message)
		if err != nil {
			log.Fatalf("websocket error: %s", err)
		}

		friendData, _ := s.db.GetFriendDataBySellyID(message.Sender)

		err = s.db.StoreMessage(s.selectedFriend.SellyID, message)
		if err != nil {
			log.Fatalf("couldn't store message: %s", err)
		}

		message.Sender = friendData.Username
		s.addMessage(message)

		s.app.Draw()
	}
}

func (s *Main) sendMessage(key tcell.Key) {
	if key == tcell.KeyEnter && s.messageInput.GetText() != "" {
		s.validateJWT(*s.localUser)

		message := data.Message{
			Sender:   s.localUser.SellyID,
			Receiver: s.selectedFriend.SellyID,
			Message:  s.messageInput.GetText(),
		}

		s.ws.WriteJSON(message)

		err := s.db.StoreMessage(s.selectedFriend.SellyID, message)
		if err != nil {
			log.Fatalf("couldn't store message: %s", err)
		}

		message.Sender = "You"
		s.addMessage(message)
		s.messageInput.SetText("")
	}
}

func (s *Main) addMessage(message data.Message) {
	fmt.Fprintf(s.internalTextView, "%s: %s\n", message.Sender, message.Message)
}

func (s *Main) Render() tview.Primitive {
	return tview.NewFlex().
		AddItem(s.friendsList, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(s.internalTextView, 0, 8, false).
			AddItem(s.messageInput, 3, 1, false).
			AddItem(tview.NewFlex().SetDirection(tview.FlexColumn).
				AddItem(s.addFriendBtn, 0, 1, false).
				AddItem(s.deleteFriendBtn, 0, 1, false).
				AddItem(s.editFriendBtn, 0, 1, false).
				AddItem(s.myInformationBtn, 0, 1, false), 0, 1, false), 0, 2, false)
}
