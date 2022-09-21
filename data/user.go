package data

import (
	"crypto/sha256"
	"fmt"
	"strings"
)

type LocalUser struct {
	ID      int
	SellyID string `db:"selly_id"`
	Seed    string `db:"seed"`
	JWT     string `json:"jwt"`
}

type Friend struct {
	ID              string
	SellyID         string `db:"selly_id"`
	Username        string `db:"username"`
	LastInteraction int    `db:"last_interaction"`
}

func (u *LocalUser) GetHashedSeed() string {
	seed := strings.ReplaceAll(u.Seed, ", ", "")

	hashedSeed := sha256.Sum256([]byte(seed))
	return fmt.Sprintf("%x", hashedSeed[:])
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

func (r *Repository) UpdateJWT(jwt string) error {
	_, err := r.db.Exec("UPDATE user_info SET jwt = $1", jwt)
	if err != nil {
		return err
	}

	return nil
}
