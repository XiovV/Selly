package screens

import (
	"encoding/json"
	"fmt"
	"github.com/XiovV/selly-client/data"
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
)

const (
	unreadMessageColor = "[#fccb00]"
)

type Main struct {
	app               *tview.Application
	internalTextView  *tview.TextView
	messageInput      *tview.InputField
	commandBox        *tview.InputField
	friendsList       *tview.TreeView
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
		friendsList:      tview.NewTreeView(),
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

	main.friendsList.SetRoot(tview.NewTreeNode("")).SetTopLevel(1)
	main.friendsList.SetSelectedFunc(main.onFriendSelect)
	main.friendsList.SetBorder(true)
	main.friendsList.SetTitle("Friends")

	main.addFriendBtn.SetSelectedFunc(main.showAddFriendScreen)
	main.deleteFriendBtn.SetSelectedFunc(main.showDeleteFriendScreen)
	main.editFriendBtn.SetSelectedFunc(main.showEditFriendScreen)
	main.myDetailsButton.SetSelectedFunc(main.showMyDetailsScreen)

	main.addFriendBtn.SetBorder(true)
	main.deleteFriendBtn.SetBorder(true)
	main.editFriendBtn.SetBorder(true)
	main.myDetailsButton.SetBorder(true)

	main.loadFriends()

	err = main.validateJWT()
	if err != nil {
		panic(err)
	}

	connection, _ := ws.NewWebsocketClient(localUser.JWT)

	main.ws = connection

	if connection != nil {
		main.isConnectionAlive = true
	}

	go main.listenForMessages()

	return main
}

func (s *Main) deleteFriend(username string) {
	friend := s.findFriendInTreeNode(username)

	s.friendsList.GetRoot().RemoveChild(friend)

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

func (s *Main) findFriendInTreeNode(username string) *tview.TreeNode {
	for _, friend := range s.friendsList.GetRoot().GetChildren() {
		friendSplit := strings.Split(friend.GetText(), " ")
		friendUsername := friendSplit[0]

		if friendUsername == username {
			return friend
		}
	}

	return nil
}

func (s *Main) editFriend(username, sellyID string) {
	friend := s.findFriendInTreeNode(s.selectedFriend.Username)
	friend.SetText(fmt.Sprintf("%s (%s)", username, s.truncateId(sellyID)))

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
	// removing the unread message hex color from the string in case it exists as it will mess up fetching friend data form the db
	selectedUsername := strings.ReplaceAll(node.GetText(), unreadMessageColor, "")

	node.SetText(selectedUsername)

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

func (s *Main) listenForMessages() {
	var message data.Message

	if !s.isConnectionAlive {
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

	for {
		err := s.ws.ReadJSON(&message)
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
			friendData, _ := s.db.GetFriendDataBySellyID(message.Sender)

			err = s.db.StoreMessage(message.Sender, message)
			if err != nil {
				log.Fatalf("couldn't store message: %s", err)
			}

			if message.Sender == s.selectedFriend.SellyID {
				message.Sender = friendData.Username
				s.addMessage(message)

				s.app.Draw()
			} else {
				friend := s.findFriendInTreeNode(friendData.Username)
				friendText := friend.GetText()

				friend.SetText(unreadMessageColor + friendText)
				s.moveFriendToTop(friend)

				s.app.Draw()
			}
		}
	}
}

func (s *Main) moveFriendToTop(friend *tview.TreeNode) {
	s.friendsList.GetRoot().RemoveChild(friend)

	oldList := s.friendsList.GetRoot().GetChildren()

	s.friendsList.GetRoot().ClearChildren()

	s.friendsList.GetRoot().AddChild(friend)

	for _, node := range oldList {
		s.friendsList.GetRoot().AddChild(node)
	}
}

func (s *Main) sendMessage(key tcell.Key) {
	if key == tcell.KeyEnter && s.messageInput.GetText() != "" && s.isConnectionAlive {
		s.validateJWT()

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
		AddItem(s.friendsList, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(s.internalTextView, 0, 8, false).
			AddItem(s.messageInput, 3, 1, false).
			AddItem(tview.NewFlex().SetDirection(tview.FlexColumn).
				AddItem(s.addFriendBtn, 0, 1, false).
				AddItem(s.deleteFriendBtn, 0, 1, false).
				AddItem(s.editFriendBtn, 0, 1, false).
				AddItem(s.myDetailsButton, 0, 1, false), 0, 1, false), 0, 2, false)
}
