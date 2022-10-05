package data

import "time"

func (r *Repository) AddFriend(sellyId, username string) error {
	_, err := r.db.Exec("INSERT INTO friends (selly_id, username) VALUES (?, ?)", sellyId, username)

	return err
}

func (r *Repository) GetFriends() ([]Friend, error) {
	friends := []Friend{}

	if err := r.db.Select(&friends, "SELECT * FROM friends"); err != nil {
		return friends, err
	}

	return friends, nil
}

func (r *Repository) GetFriendsSorted() ([]Friend, error) {
	friends := []Friend{}

	if err := r.db.Unsafe().Select(&friends, "SELECT selly_id, username, last_interaction FROM friends ORDER BY last_interaction DESC"); err != nil {
		return friends, err
	}

	return friends, nil
}
func (r *Repository) GetFriendDataByUsername(username string) (Friend, error) {
	var friend Friend

	if err := r.db.Get(&friend, "SELECT selly_id, username, last_interaction FROM friends WHERE username = ?", username); err != nil {
		return Friend{}, err
	}

	return friend, nil
}

func (r *Repository) GetFriendDataBySellyID(sellyId string) (Friend, error) {
	var friend Friend

	if err := r.db.Get(&friend, "SELECT selly_id, username, last_interaction FROM friends WHERE selly_id = ?", sellyId); err != nil {
		return Friend{}, err
	}

	return friend, nil
}

func (r *Repository) UpdateLastInteraction(sellyId string) error {
	_, err := r.db.Exec("UPDATE friends SET last_interaction = $1 WHERE selly_id = $2", time.Now().Unix(), sellyId)

	return err
}

func (r *Repository) EditFriend(sellyId, newSellyId, username string) error {
	_, err := r.db.Exec("UPDATE friends SET selly_id = $1, username = $2 WHERE selly_id = $3", newSellyId, username, sellyId)

	if err != nil {
		return err
	}

	// if the user changes the sellyId, it needs to be updated in the messages table, the selly_id and sender fields need to be updated.
	// ON DELETE CASCADE is set up, but for some reason the selly_id field still doesn't get updated, so we're updating it here manually.
	if sellyId != newSellyId {
		tx, err := r.db.Begin()
		if err != nil {
			return err
		}

		_, err = tx.Exec("UPDATE messages SET selly_id = $1", newSellyId)
		if err != nil {
			return err
		}

		_, err = tx.Exec("UPDATE messages SET sender = $1 WHERE sender = $2", newSellyId, sellyId)
		if err != nil {
			return err
		}

		err = tx.Commit()
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *Repository) DeleteFriendByUsername(username string) error {
	_, err := r.db.Exec("DELETE FROM friends WHERE username = ?", username)

	return err
}
