package friendslist

import (
	"fmt"
	"strconv"
	"strings"
)

type ListText struct {
	username       string
	sellyId        string
	unreadMessages int
}

func NewListText(username, sellyId string, unreadMessages int) ListText {
	return ListText{
		username:       username,
		sellyId:        sellyId,
		unreadMessages: unreadMessages,
	}
}

func parseText(text string) ListText {
	textSplit := strings.Split(text, " ")

	username := textSplit[0]
	truncatedId := textSplit[1]
	unreadMessages := parseUnreadMessages(text)

	return ListText{
		username:       username,
		sellyId:        truncatedId,
		unreadMessages: unreadMessages,
	}
}

func parseUnreadMessages(text string) int {
	textSplit := strings.Split(text, " ")

	if len(textSplit) < 3 {
		return 0
	}

	r := strings.NewReplacer("(", "", ")", "")

	unreadMessages, _ := strconv.Atoi(r.Replace(textSplit[2]))
	return unreadMessages
}

func (t *ListText) IncrementUnreadMessages() {
	t.unreadMessages += 1
}

func (t *ListText) SetUnreadMessagesCounter(n int) {
	t.unreadMessages = n
}

func (t *ListText) String() string {
	//TODO: consider refactoring this
	var s string
	if strings.Contains(t.sellyId, "(") {
		s = fmt.Sprintf("%s %s", t.username, t.sellyId)
	} else {
		s = fmt.Sprintf("%s (%s)", t.username, truncateId(t.sellyId))
	}

	if t.unreadMessages > 0 {
		s += fmt.Sprintf(" (%d)", t.unreadMessages)
	}

	return s
}
