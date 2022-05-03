package screens

import (
	"fmt"
	"github.com/XiovV/selly-client/data"
	"github.com/XiovV/selly-client/ws"
	"github.com/gdamore/tcell/v2"
	"github.com/gorilla/websocket"
	"github.com/rivo/tview"
	"log"
	"strings"
)

type Main struct {
	app              *tview.Application
	internalTextView *tview.TextView
	messageInput     *tview.InputField
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
		friendsList:      tview.NewTreeView(),
		db:               db,
	}

	localUser, err := main.db.GetLocalUserInfo()
	if err != nil {
		log.Fatalf("couldn't get local user info: %s", err)
	}

	main.localUser = &localUser

	connection := ws.NewWebsocketClient(localUser.SellyID)
	main.ws = connection

	go main.listenForMessages(connection)

	main.internalTextView.SetDynamicColors(true).
		SetRegions(true).
		SetWordWrap(true).SetBorder(true)

	main.messageInput.SetDoneFunc(main.sendMessage).SetPlaceholder("Message")
	main.messageInput.SetBorder(true)

	main.friendsList.SetRoot(tview.NewTreeNode("")).SetTopLevel(1)
	main.friendsList.SetSelectedFunc(main.onFriendSelect)
	main.friendsList.SetBorder(true)
	main.friendsList.SetTitle("Friends")

	main.loadFriends()

	return main
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
			fmt.Println("error receiving:", err)
		}

		friendData, _ := s.db.GetFriendDataBySellyID(message.Sender)

		err = s.db.StoreMessage(s.selectedFriend.SellyID, message)
		if err != nil {
			log.Fatal("couldn't store message: %s", err)
		}

		message.Sender = friendData.Username
		s.addMessage(message)

		s.app.Draw()
	}
}

func (s *Main) sendMessage(key tcell.Key) {
	if key == tcell.KeyEnter && s.messageInput.GetText() != "" {
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
			AddItem(s.messageInput, 5, 1, false), 0, 2, false)
}
