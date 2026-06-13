package models

import (
	"errors"
	"time"

	"recallo/db"
)

type Private struct {
	ID        int64     `json:"id"`
	User1     string    `json:"user1"`
	User2     string    `json:"user2"`
	CreatedAt time.Time `json:created_at`
}

func GetPrivateByID(privateId int64) (*Private, error) {
	db, err := db.GetDB()
	if err != nil {
		return nil, err
	}

	var private Private
	err = db.QueryRow("SELECT id, user1_id, user2_id, created_at FROM privates WHERE id = ?", privateId).Scan(
		&private.ID,
		&private.User1,
		&private.User2,
		&private.CreatedAt,
	)
	if err != nil {
		return nil, err
	}

	return &private, nil
}

func GetPrivateByUsers(user1Id, user2Id int64) (*Private, error) {
	if user1Id > user2Id {
		user1Id, user2Id = user2Id, user1Id
	}

	db, err := db.GetDB()
	if err != nil {
		return nil, err
	}

	var private Private
	err = db.QueryRow("SELECT id, user1_id, user2_id, created_at FROM privates WHERE user1_id = ? AND user2_id = ?", user1Id, user2Id).Scan(
		&private.ID,
		&private.User1,
		&private.User2,
		&private.CreatedAt,
	)
	if err != nil {
		return nil, err
	}

	return &private, nil
}

func GetPrivatesForUser(userId int64) ([]*Private, error) {
	db, err := db.GetDB()
	if err != nil {
		return nil, err
	}

	rows, err := db.Query(
		`
		SELECT id, user1_id, user2_id, created_at
		FROM privates
		WHERE user1_id = ? OR user2_id = ?
		ORDER BY created_at DESC
		`,
		userId,
		userId,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var privates []*Private
	for rows.Next() {
		var private Private
		err := rows.Scan(
			&private.ID,
			&private.User1,
			&private.User2,
			&private.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		privates = append(privates, &private)
	}

	return privates, nil
}

func CreatePrivate(user1Id, user2Id int64) (*Private, error) {
	if user1Id == user2Id {
		return nil, errors.New("cannot create private chat with the same user")
	}

	if user1Id > user2Id {
		user1Id, user2Id = user2Id, user1Id
	}

	db, err := db.GetDB()
	if err != nil {
		return nil, err
	}

	existingPrivate, err := GetPrivateByUsers(user1Id, user2Id)
	if err != nil {
		return nil, err
	}
	if existingPrivate != nil {
		return existingPrivate, nil
	}

	result, err := db.Exec("INSERT INTO privates (user1_id, user2_id) VALUES (?, ?)", user1Id, user2Id)
	if err != nil {
		return nil, err
	}

	privateId, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}

	return GetPrivateByID(privateId)
}
