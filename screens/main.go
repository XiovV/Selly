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
}

func NewMainScreen(app *tview.Application, db *data.Repository) *Main {
	main := &Main{
		app:              app,
		internalTextView: tview.NewTextView(),
		messageInput:     tview.NewInputField(),
		commandBox:       tview.NewInputField(),
		friendsList:      tview.NewTreeView(),
		db:               db,
	}

	localUser, err := main.db.GetLocalUserInfo()
	if err != nil {
		log.Fatalf("couldn't get local user info: %s", err)
	}

	main.localUser = &localUser

	main.validateJWT(localUser)

	connection := ws.NewWebsocketClient(localUser.JWT)
	main.ws = connection

	go main.listenForMessages(connection)

	main.internalTextView.SetDynamicColors(true).
		SetRegions(true).
		SetWordWrap(true).SetBorder(true)
	main.internalTextView.ScrollToEnd()

	main.messageInput.SetDoneFunc(main.sendMessage).SetPlaceholder("Message")
	main.messageInput.SetBorder(false)

	main.commandBox.SetDoneFunc(main.handleCommand).SetPlaceholder("Enter a command")
	main.commandBox.SetBorder(false)

	main.friendsList.SetRoot(tview.NewTreeNode("")).SetTopLevel(1)
	main.friendsList.SetSelectedFunc(main.onFriendSelect)
	main.friendsList.SetBorder(true)
	main.friendsList.SetTitle("Friends")

	main.loadFriends()

	return main
}

func (s *Main) handleCommand(key tcell.Key) {
	if key == tcell.KeyEnter && s.commandBox.GetText() != "" {
		command := strings.Split(s.commandBox.GetText(), " ")

		if len(command) < 2 {
			s.commandBox.SetText("")
			s.commandBox.SetPlaceholder("invalid amount of arguments")
			return
		}

		switch command[0] {
		case "/add":
			err := s.addFriend(command)
			if err != nil {
				s.commandBox.SetText("")
				s.commandBox.SetPlaceholder(err.Error())
				return
			}
		case "/remove":
			err := s.removeFriend(command)
			if err != nil {
				s.commandBox.SetText("")
				s.commandBox.SetPlaceholder(err.Error())
				return
			}
		}

		s.commandBox.SetPlaceholder("Enter a command")
		s.commandBox.SetText("")
	}
}

func (s *Main) removeFriend(command []string) error {
	if len(command) > 2 {
		return errors.New(fmt.Sprintf(" remove command needs exactly 1 argument, got: %d", len(command)-1))
	}

	username := command[1]

	if len(username) < 1 {
		return errors.New(" username needs to be at least one character long")
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

func (s *Main) addFriend(command []string) error {
	if len(command) > 3 {
		return errors.New(fmt.Sprintf(" add command needs exactly 2 arguments, got: %d", len(command)-1))
	}

	sellyID := command[1]
	username := command[2]

	if len(sellyID) < 64 {
		return errors.New(fmt.Sprintf(" Selly ID needs to be 64 characters long, got: %d", len(sellyID)-1))
	}

	if len(username) < 1 {
		return errors.New(" username needs to be at least one character long")
	}

	node := tview.NewTreeNode(fmt.Sprintf("%s (%s)", username, s.truncateId(sellyID)))
	s.friendsList.GetRoot().AddChild(node)

	err := s.db.AddFriend(sellyID, username)
	if err != nil {
		panic(err)
	}

	return nil
}

func (s *Main) validateJWT(localUser data.LocalUser) {
	if localUser.JWT == "" {
		token := s.getNewToken(localUser.SellyID)

		err := s.db.UpdateJWT(token)
		if err != nil {
			panic(err)
		}

		s.localUser.JWT = token

		return
	}

	if jwt.IsTokenExpired(localUser.JWT) {
		token := s.refreshToken(localUser.JWT)

		err := s.db.UpdateJWT(token)
		if err != nil {
			panic(err)
		}

		s.localUser.JWT = token
	}

}

func (s *Main) getNewToken(sellyId string) string {
	req, err := http.NewRequest(http.MethodGet, "http://localhost:8082/v1/users/token?id="+sellyId, nil)
	if err != nil {
		panic(err)
	}

	client := &http.Client{}
	r, err := client.Do(req)

	var response struct {
		AccessToken string `json:"access_token"`
	}

	decoder := json.NewDecoder(r.Body)
	err = decoder.Decode(&response)
	if err != nil {
		panic(err)
	}

	return response.AccessToken
}

func (s *Main) refreshToken(jwt string) string {
	req, err := http.NewRequest(http.MethodGet, "http://localhost:8082/v1/users/refresh-token", nil)
	if err != nil {
		panic(err)
	}

	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", jwt))
	client := &http.Client{}
	r, err := client.Do(req)

	var response struct {
		AccessToken string `json:"access_token"`
	}

	decoder := json.NewDecoder(r.Body)
	err = decoder.Decode(&response)
	if err != nil {
		panic(err)
	}

	return response.AccessToken
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
			AddItem(s.internalTextView, 0, 3, false).
			AddItem(s.messageInput, 2, 1, false).
			AddItem(s.commandBox, 2, 1, false), 0, 2, false)
}
