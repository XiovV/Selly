package friendslist

import (
	"fmt"
	"github.com/rivo/tview"
	"strings"
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
	node.SetText(fmt.Sprintf("%s (%s)", newUsername, f.truncateId(sellyId)))
}

func (f *List) MoveFriendToTop(username string) {
	friend := f.findFriendInTreeNode(username)
	friendText := friend.GetText()

	friend.SetText("[#fccb00]" + friendText)
	f.moveNodeToTop(friend)
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

func (f *List) GetFirst() *tview.TreeNode {
	return f.getRoot().GetChildren()[0]
}

func (f *List) SetCurrentFriend(node *tview.TreeNode) {
	f.treeView.SetCurrentNode(node)
}

func (f *List) findFriendInTreeNode(username string) *tview.TreeNode {
	for _, friend := range f.treeView.GetRoot().GetChildren() {
		friendSplit := strings.Split(friend.GetText(), " ")
		friendUsername := friendSplit[0]

		if friendUsername == username {
			return friend
		}
	}

	return nil
}

func (f *List) AddFriend(username, sellyId string) {
	node := tview.NewTreeNode(fmt.Sprintf("%s (%s)", username, f.truncateId(sellyId)))
	f.addChild(node)
}

func (f *List) addChild(node *tview.TreeNode) {
	f.getRoot().AddChild(node)
}

func (f *List) getRoot() *tview.TreeNode {
	return f.treeView.GetRoot()
}

func (f *List) truncateId(id string) string {
	return fmt.Sprintf("%s...%s", id[:7], id[len(id)-7:])
}