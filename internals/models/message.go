package models

import (
	"time"

	"recallo/db"
)

type Message struct {
	ID        int64 `json:"id"`
	FromID    int64 `json:"from_id"`
	PrivateID int64 `json:"private_id"`

	MessageType string `json:"message_type"`
	Content     string `json:"content"`

	Delivered bool `json:"delivered"`
	Read      bool `json:"read"`

	CreatedAt time.Time `json:"created_at"`
}

func boolToInt(val bool) int {
	if val {
		return 1
	}
	return 0
}

func CreateMessage(m *Message) error {
	db, err := db.GetDB()
	if err != nil {
		return err
	}

	_, err = db.Exec(
		"INSERT INTO messages (from_id, private_id, message_type, content, delivered, read) VALUES (?, ?, ?, ?, ?, ?)",
		m.FromID,
		m.PrivateID,
		m.MessageType,
		m.Content,
		boolToInt(m.Delivered),
		boolToInt(m.Read),
	)
	if err != nil {
		return err
	}

	return nil
}

func GetMessagesByPrivateID(privateId int64, page, limit int) ([]*Message, error) {
	db, err := db.GetDB()
	if err != nil {
		return nil, err
	}

	offset := (page - 1) * limit

	rows, err := db.Query(
		`
		SELECT id, from_id, private_id, message_type, content, delivered, read, created_at
		FROM messages
		WHERE private_id = ?
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?
		`,
		privateId,
		limit,
		offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []*Message
	for rows.Next() {
		var m Message
		var deliveredInt, readInt int
		err := rows.Scan(
			&m.ID,
			&m.FromID,
			&m.PrivateID,
			&m.MessageType,
			&m.Content,
			&deliveredInt,
			&readInt,
			&m.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		m.Delivered = deliveredInt == 1
		m.Read = readInt == 1
		messages = append(messages, &m)
	}

	return messages, nil
}

func GetMessageByID(messageId int64) (*Message, error) {
	db, err := db.GetDB()
	if err != nil {
		return nil, err
	}

	var m Message
	var deliveredInt, readInt int
	err = db.QueryRow(
		"SELECT id, from_id, private_id, message_type, content, delivered, read, created_at FROM messages WHERE id = ?",
		messageId,
	).Scan(
		&m.ID,
		&m.FromID,
		&m.PrivateID,
		&m.MessageType,
		&m.Content,
		&deliveredInt,
		&readInt,
		&m.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	m.Delivered = deliveredInt == 1
	m.Read = readInt == 1

	return &m, nil
}

func GetUndeliveredMessagesByPrivateID(privateId int64) ([]*Message, error) {
	db, err := db.GetDB()
	if err != nil {
		return nil, err
	}

	rows, err := db.Query(
		`
		SELECT id, from_id, private_id, message_type, content, delivered, read, created_at
		FROM messages
		WHERE private_id = ? AND delivered = 0
		ORDER BY created_at DESC
		`,
		privateId,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	messages := []*Message{}
	for rows.Next() {
		var m Message
		var deliveredInt, readInt int
		err := rows.Scan(
			&m.ID,
			&m.FromID,
			&m.PrivateID,
			&m.MessageType,
			&m.Content,
			&deliveredInt,
			&readInt,
			&m.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		m.Delivered = deliveredInt == 1
		m.Read = readInt == 1
		messages = append(messages, &m)
	}

	return messages, nil
}

func MarkMessageAsDelivered(messageId int64) error {
	db, err := db.GetDB()
	if err != nil {
		return err
	}

	_, err = db.Exec("UPDATE messages SET delivered = 1 WHERE id = ?", messageId)
	if err != nil {
		return err
	}

	return nil
}

func MarkMessageAsRead(messageId int64) error {
	db, err := db.GetDB()
	if err != nil {
		return err
	}

	_, err = db.Exec("UPDATE messages SET read = 1 WHERE id = ?", messageId)
	if err != nil {
		return err
	}

	return nil
}
