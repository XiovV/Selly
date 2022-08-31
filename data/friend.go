package data

func (r *Repository) AddFriend(sellyId, username string) error {
	_, err := r.db.Exec("INSERT INTO friends (selly_id, username) VALUES (?, ?)", sellyId, username)
	if err != nil {
		return err
	}

	return nil
}

func (r *Repository) EditFriend(sellyId, newSellyId, username string) error {
	_, err := r.db.NamedExec("UPDATE friends SET selly_id=:newSellyId, username=:username WHERE selly_id=:sellyId",
		map[string]interface{}{
			"sellyId":    sellyId,
			"newSellyId": newSellyId,
			"username":   username,
		})

	if err != nil {
		return err
	}

	return err
}

func (r *Repository) DeleteFriendByUsername(username string) error {
	_, err := r.db.Exec("DELETE FROM friends WHERE username = ?", username)
	if err != nil {
		return err
	}

	return nil
}
