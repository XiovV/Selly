package data

type LocalUser struct {
	ID      int
	SellyID string `db:"selly_id"`
	Seed    string `db:"seed"`
	JWT     string `json:"jwt"`
}

type Friend struct {
	ID       string
	SellyID  string `db:"selly_id"`
	Username string `db:"username"`
}

func (r *Repository) GetLocalUserInfo() (LocalUser, error) {
	userInfo := LocalUser{}

	err := r.db.Get(&userInfo, "SELECT * FROM user_info LIMIT 1")

	if err != nil {
		return LocalUser{}, err
	}

	return userInfo, nil
}

func (r *Repository) StoreLocalUserInfo(id, seed string) error {
	_, err := r.db.Exec("INSERT INTO user_info (selly_id, seed) VALUES ($1, $2)", id, seed)
	if err != nil {
		return err
	}

	return nil
}

func (r *Repository) GetFriends() ([]Friend, error) {
	friends := []Friend{}

	if err := r.db.Select(&friends, "SELECT * FROM friends"); err != nil {
		return friends, err
	}

	return friends, nil
}

func (r *Repository) GetFriendDataByUsername(username string) (Friend, error) {
	var friend Friend

	if err := r.db.Get(&friend, "SELECT * FROM friends WHERE username = ?", username); err != nil {
		return Friend{}, err
	}

	return friend, nil
}

func (r *Repository) GetFriendDataBySellyID(sellyId string) (Friend, error) {
	var friend Friend

	if err := r.db.Get(&friend, "SELECT * FROM friends WHERE selly_id = ?", sellyId); err != nil {
		return Friend{}, err
	}

	return friend, nil
}
