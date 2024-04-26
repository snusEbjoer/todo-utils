package types

import "time"

type UserCreds struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type Tokens struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}
type User struct {
	Id             int       `json:"id"`
	Username       string    `json:"username"`
	Description    *string   `json:"description"`
	HashedPassword string    `json:"hashed_password"`
	Created_at     time.Time `json:"created_at"`
}

type MessageBody struct {
	StatusCode int    `json:"status_code"`
	Message    string `json:"message"`
}
