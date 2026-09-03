package domain

import "time"

type Role string

const (
	RoleAdmin Role = "admin"
	RoleUser  Role = "user"
)

type User struct {
	ID           int64      `json:"id"`
	Email        string     `json:"email"`
	PasswordHash string     `json:"-"`
	Role         Role       `json:"role"`
	IsActive     bool       `json:"is_active"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	DeletedAt    *time.Time `json:"deleted_at,omitempty"`
}

type Actor struct {
	UserID int64
	Role   Role
}

func (a Actor) IsAdmin() bool { return a.Role == RoleAdmin }

type RefreshSession struct {
	ID                int64
	UserID            int64
	TokenID           string
	TokenHash         string
	ExpiresAt         time.Time
	RevokedAt         *time.Time
	ReplacedByTokenID *string
	CreatedAt         time.Time
}
