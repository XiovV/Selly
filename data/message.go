package data

type Message struct {
	Sender     string `json:"sender"`
	Receiver   string `json:"receiver"`
	Message    string `json:"message"`
	DateCrated int64  `json:"date_crated"`
	Read       int    `json:"read"`
}

func (r *Repository) StoreMessage(sellyId string, message Message) error {
	_, err := r.db.Exec("INSERT INTO messages (selly_id, sender, message, date_created, read) VALUES (?, ?, ?, ?, ?)", sellyId, message.Sender, message.Message, message.DateCrated, message.Read)
	if err != nil {
		return err
	}

	return nil
}

func (r *Repository) GetCountOfUnreadMessages(sellyId string) int {
	var unread int

	r.db.QueryRowx("SELECT COUNT(*) FROM messages WHERE selly_id = ? AND read = 0", sellyId).Scan(&unread)

	return unread
}

func (r *Repository) GetMessages(sellyId string) ([]Message, error) {
	messages := []Message{}

	if err := r.db.Select(&messages, "SELECT sender, message FROM messages WHERE selly_id = ?", sellyId); err != nil {
		return messages, err
	}

	return messages, nil
}

func (r *Repository) SetRead(sellyId string) {
	r.db.Exec("UPDATE messages SET read = 1 WHERE selly_id = $1 AND read = 0", sellyId)
}
