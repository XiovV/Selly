package data

type Message struct {
	Sender   string `json:"sender"`
	Receiver string `json:"receiver"`
	Message  string `json:"message"`
}

func (r *Repository) StoreMessage(sellyId string, message Message) error {
	_, err := r.db.Exec("INSERT INTO messages (selly_id, sender, message) VALUES (?, ?, ?)", sellyId, message.Sender, message.Message)
	if err != nil {
		return err
	}

	return nil
}

func (r *Repository) GetMessages(sellyId string) ([]Message, error) {
	messages := []Message{}

	if err := r.db.Select(&messages, "SELECT sender, message FROM messages WHERE selly_id = ?", sellyId); err != nil {
		return messages, err
	}

	return messages, nil
}
