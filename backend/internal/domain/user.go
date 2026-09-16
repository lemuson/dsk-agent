package domain

import "encoding/json"

type Role string
const (
	RoleUser       Role = "user"
	RoleManager    Role = "manager"
	RoleSupervisor Role = "supervisor"
)

type User struct {
	ID           int            `json:"id"`
	Name         string         `json:"name"`
	Email        string         `json:"email"`
	PasswordHash string         `json:"-"`
	Role         Role           `json:"role"`
	BudgetMax    *int64         `json:"budget_max,omitempty"`
	Preferences  map[string]any `json:"preferences,omitempty"`
}

type ClientPreferences struct {
	ClientID    int            `json:"client_id"`
	BudgetMax   *int64         `json:"budget_max,omitempty"`
	Preferences map[string]any `json:"preferences"`
}

type UpdateUserRequest struct {
	Name     string `json:"name,omitempty"`
	Email    string `json:"email,omitempty"`
	Password string `json:"password,omitempty"`
}

func DecodePreferences(raw []byte) (map[string]any, error) {
	preferences := make(map[string]any)
	if len(raw) == 0 {
		return preferences, nil
	}
	if err := json.Unmarshal(raw, &preferences); err != nil {
		return nil, err
	}
	return preferences, nil
}
