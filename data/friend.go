package data

func (r *Repository) AddFriend(sellyId, username string) error {
	_, err := r.db.Exec("INSERT INTO friends (selly_id, username) VALUES (?, ?)", sellyId, username)
	if err != nil {
		return err
	}

	return nil
}

func (r *Repository) DeleteFriendByUsername(username string) error {
	_, err := r.db.Exec("DELETE FROM friends WHERE username = ?", username)
	if err != nil {
		return err
	}

	return nil
}
