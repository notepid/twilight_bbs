package message

import (
	"database/sql"
	"fmt"
)

// Repo handles database operations for messages and areas.
type Repo struct {
	db *sql.DB
}

// NewRepo creates a new message repository.
func NewRepo(db *sql.DB) *Repo {
	return &Repo{db: db}
}

// ListAreas returns all message areas the user has access to read.
func (r *Repo) ListAreas(userLevel int) ([]*Area, error) {
	rows, err := r.db.Query(`
		SELECT a.id, a.name, a.description, a.read_level, a.write_level, a.sort_order,
		       COALESCE((SELECT COUNT(*) FROM messages WHERE area_id = a.id), 0) as total
		FROM message_areas a
		WHERE a.read_level <= ?
		ORDER BY a.sort_order, a.name
	`, userLevel)
	if err != nil {
		return nil, fmt.Errorf("list areas: %w", err)
	}
	defer rows.Close()

	var areas []*Area
	for rows.Next() {
		a := &Area{}
		if err := rows.Scan(&a.ID, &a.Name, &a.Description, &a.ReadLevel,
			&a.WriteLevel, &a.SortOrder, &a.TotalMsgs); err != nil {
			return nil, err
		}
		areas = append(areas, a)
	}
	return areas, rows.Err()
}

// ListAreasWithNew returns areas with new message count for a user.
func (r *Repo) ListAreasWithNew(userID, userLevel int) ([]*Area, error) {
	areas, err := r.ListAreas(userLevel)
	if err != nil {
		return nil, err
	}

	for _, a := range areas {
		var lastRead int
		r.db.QueryRow(`
			SELECT COALESCE(last_read_id, 0) FROM message_read
			WHERE user_id = ? AND area_id = ?
		`, userID, a.ID).Scan(&lastRead)

		var newCount int
		r.db.QueryRow(`
			SELECT COUNT(*) FROM messages WHERE area_id = ? AND id > ?
		`, a.ID, lastRead).Scan(&newCount)

		a.NewMsgs = newCount
	}

	return areas, nil
}

// GetArea returns a single area by ID.
func (r *Repo) GetArea(id int) (*Area, error) {
	a := &Area{}
	err := r.db.QueryRow(`
		SELECT id, name, description, read_level, write_level, sort_order
		FROM message_areas WHERE id = ?
	`, id).Scan(&a.ID, &a.Name, &a.Description, &a.ReadLevel, &a.WriteLevel, &a.SortOrder)
	if err != nil {
		return nil, fmt.Errorf("get area %d: %w", id, err)
	}
	return a, nil
}

// ListMessages returns messages in an area, paginated.
func (r *Repo) ListMessages(areaID, offset, limit int) ([]*Message, error) {
	rows, err := r.db.Query(`
		SELECT m.id, m.area_id, m.from_user_id, 
		       COALESCE(uf.username, 'Unknown') as from_name,
		       m.to_user_id,
		       COALESCE(ut.username, '') as to_name,
		       m.subject, m.created_at
		FROM messages m
		LEFT JOIN users uf ON uf.id = m.from_user_id
		LEFT JOIN users ut ON ut.id = m.to_user_id
		WHERE m.area_id = ?
		ORDER BY m.id ASC
		LIMIT ? OFFSET ?
	`, areaID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list messages: %w", err)
	}
	defer rows.Close()

	var messages []*Message
	for rows.Next() {
		msg := &Message{}
		var toUserID sql.NullInt64
		var toName sql.NullString
		if err := rows.Scan(&msg.ID, &msg.AreaID, &msg.FromUserID, &msg.FromName,
			&toUserID, &toName, &msg.Subject, &msg.CreatedAt); err != nil {
			return nil, err
		}
		if toUserID.Valid {
			id := int(toUserID.Int64)
			msg.ToUserID = &id
		}
		if toName.Valid {
			msg.ToName = toName.String
		}
		messages = append(messages, msg)
	}
	return messages, rows.Err()
}

// GetMessage returns a single message by ID with full body.
func (r *Repo) GetMessage(id int) (*Message, error) {
	msg := &Message{}
	var toUserID sql.NullInt64
	var toName sql.NullString
	var replyToID sql.NullInt64

	err := r.db.QueryRow(`
		SELECT m.id, m.area_id, m.from_user_id,
		       COALESCE(uf.username, 'Unknown') as from_name,
		       m.to_user_id,
		       COALESCE(ut.username, '') as to_name,
		       m.subject, m.body, m.reply_to_id, m.created_at
		FROM messages m
		LEFT JOIN users uf ON uf.id = m.from_user_id
		LEFT JOIN users ut ON ut.id = m.to_user_id
		WHERE m.id = ?
	`, id).Scan(&msg.ID, &msg.AreaID, &msg.FromUserID, &msg.FromName,
		&toUserID, &toName, &msg.Subject, &msg.Body, &replyToID, &msg.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("get message %d: %w", id, err)
	}

	if toUserID.Valid {
		id := int(toUserID.Int64)
		msg.ToUserID = &id
	}
	if toName.Valid {
		msg.ToName = toName.String
	}
	if replyToID.Valid {
		id := int(replyToID.Int64)
		msg.ReplyToID = &id
	}

	return msg, nil
}

// Post creates a new message.
func (r *Repo) Post(areaID, fromUserID int, toUserID *int, subject, body string, replyToID *int) (int, error) {
	result, err := r.db.Exec(`
		INSERT INTO messages (area_id, from_user_id, to_user_id, subject, body, reply_to_id)
		VALUES (?, ?, ?, ?, ?, ?)
	`, areaID, fromUserID, toUserID, subject, body, replyToID)
	if err != nil {
		return 0, fmt.Errorf("post message: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}
	return int(id), nil
}

// MarkRead updates the last-read pointer for a user in an area.
func (r *Repo) MarkRead(userID, areaID, messageID int) error {
	_, err := r.db.Exec(`
		INSERT INTO message_read (user_id, area_id, last_read_id) VALUES (?, ?, ?)
		ON CONFLICT(user_id, area_id) DO UPDATE SET last_read_id = MAX(last_read_id, excluded.last_read_id)
	`, userID, areaID, messageID)
	return err
}

// GetNewMessages returns unread messages in an area for a user.
func (r *Repo) GetNewMessages(userID, areaID int) ([]*Message, error) {
	var lastRead int
	r.db.QueryRow(`
		SELECT COALESCE(last_read_id, 0) FROM message_read
		WHERE user_id = ? AND area_id = ?
	`, userID, areaID).Scan(&lastRead)

	return r.getMessagesAfter(areaID, lastRead)
}

// getMessagesAfter returns messages in an area after a given ID.
func (r *Repo) getMessagesAfter(areaID, afterID int) ([]*Message, error) {
	rows, err := r.db.Query(`
		SELECT m.id, m.area_id, m.from_user_id,
		       COALESCE(uf.username, 'Unknown') as from_name,
		       m.to_user_id,
		       COALESCE(ut.username, '') as to_name,
		       m.subject, m.body, m.created_at
		FROM messages m
		LEFT JOIN users uf ON uf.id = m.from_user_id
		LEFT JOIN users ut ON ut.id = m.to_user_id
		WHERE m.area_id = ? AND m.id > ?
		ORDER BY m.id ASC
	`, areaID, afterID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []*Message
	for rows.Next() {
		msg := &Message{}
		var toUserID sql.NullInt64
		var toName sql.NullString
		if err := rows.Scan(&msg.ID, &msg.AreaID, &msg.FromUserID, &msg.FromName,
			&toUserID, &toName, &msg.Subject, &msg.Body, &msg.CreatedAt); err != nil {
			return nil, err
		}
		if toUserID.Valid {
			id := int(toUserID.Int64)
			msg.ToUserID = &id
		}
		if toName.Valid {
			msg.ToName = toName.String
		}
		messages = append(messages, msg)
	}
	return messages, rows.Err()
}

// CountMessages returns the total number of messages in an area.
func (r *Repo) CountMessages(areaID int) int {
	var count int
	r.db.QueryRow("SELECT COUNT(*) FROM messages WHERE area_id = ?", areaID).Scan(&count)
	return count
}
