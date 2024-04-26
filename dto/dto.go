package dto

import (
	"database/sql"
	"time"
)

type UserDto struct {
	Id          int       `json:"id"`
	Username    string    `json:"username"`
	Description string    `json:"description"`
	Created_at  time.Time `json:"created_at"`
}

type TodoDto struct {
	Id          int          `json:"id"`
	Title       string       `json:"title"`
	Description string       `json:"description"`
	CreatedAt   time.Time    `json:"created_at"`
	UpdatedAt   sql.NullTime `json:"updated_at"`
	DoneAt      sql.NullTime `json:"done_at"`
	UserId      int          `json:"user_id"`
}
