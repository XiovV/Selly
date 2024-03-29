package friendslist

import (
	"fmt"
	"github.com/rivo/tview"
	"strings"
)

const (
	unreadMessageColor = "[#fccb00]"
)

type List struct {
	treeView *tview.TreeView
}

func New() *List {
	f := List{}

	f.treeView = tview.NewTreeView()

	f.treeView.SetRoot(tview.NewTreeNode("")).SetTopLevel(1)
	f.treeView.SetBorder(true)
	f.treeView.SetTitle("Friends")

	return &f
}

func (f *List) SetSelectedFunc(handler func(node *tview.TreeNode)) {
	f.treeView.SetSelectedFunc(handler)
}

func (f *List) GetTreeView() *tview.TreeView {
	return f.treeView
}

func (f *List) RemoveFriend(username string) {
	node := f.findFriendInTreeNode(username)

	f.getRoot().RemoveChild(node)
}

func (f *List) EditFriendText(oldUsername, newUsername, sellyId string) {
	node := f.findFriendInTreeNode(oldUsername)
	node.SetText(fmt.Sprintf("%s (%s)", newUsername, truncateId(sellyId)))
}

func (f *List) SanitizeNode(node *tview.TreeNode) {
	removedColor := strings.ReplaceAll(node.GetText(), unreadMessageColor, "")

	parsed := parseText(removedColor)
	parsed.SetUnreadMessagesCounter(0)

	node.SetText(parsed.String())
}

func (f *List) IncrementUnreadMessages(username string) {
	friend := f.findFriendInTreeNode(username)

	friendText := friend.GetText()

	parsedText := parseText(friendText)
	parsedText.IncrementUnreadMessages()

	friend.SetText(unreadMessageColor + parsedText.String())
	f.moveNodeToTop(friend)
}

func (f *List) SetUnreadCounter(username string, counter int) {
	if counter == 0 {
		return
	}

	friend := f.findFriendInTreeNode(username)

	friendText := friend.GetText()

	parsedText := parseText(friendText)
	parsedText.SetUnreadMessagesCounter(counter)

	friend.SetText(unreadMessageColor + parsedText.String())
}

func (f *List) moveNodeToTop(node *tview.TreeNode) {
	f.treeView.GetRoot().RemoveChild(node)

	oldList := f.treeView.GetRoot().GetChildren()

	f.treeView.GetRoot().ClearChildren()

	f.treeView.GetRoot().AddChild(node)

	for _, n := range oldList {
		f.treeView.GetRoot().AddChild(n)
	}
}

func (f *List) MoveToTop(username string) {
	node := f.findFriendInTreeNode(username)

	f.moveNodeToTop(node)
}

func (f *List) GetFirst() *tview.TreeNode {
	children := f.getRoot().GetChildren()

	if len(children) > 0 {
		return children[0]
	}

	return nil
}

func (f *List) SetCurrentFriend(node *tview.TreeNode) {
	f.treeView.SetCurrentNode(node)
}

func (f *List) findFriendInTreeNode(username string) *tview.TreeNode {
	for _, friend := range f.treeView.GetRoot().GetChildren() {
		friendSplit := strings.Split(friend.GetText(), " ")
		friendUsername := strings.ReplaceAll(friendSplit[0], unreadMessageColor, "")

		if friendUsername == username {
			return friend
		}
	}

	return nil
}

func (f *List) AddFriend(username, sellyId string) {
	node := tview.NewTreeNode(fmt.Sprintf("%s (%s)", username, truncateId(sellyId)))
	f.addChild(node)
}

func (f *List) addChild(node *tview.TreeNode) {
	f.getRoot().AddChild(node)
}

func (f *List) getRoot() *tview.TreeNode {
	return f.treeView.GetRoot()
}

func truncateId(id string) string {
	return fmt.Sprintf("%s...%s", id[:7], id[len(id)-7:])
}
