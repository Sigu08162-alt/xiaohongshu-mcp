package user

import "fmt"

// UserID is a value object representing a user identifier.
type UserID struct {
	value string
}

// NewUserID creates a UserID, returning an error if id is empty.
func NewUserID(id string) (UserID, error) {
	if id == "" {
		return UserID{}, fmt.Errorf("user ID 不能为空")
	}
	return UserID{value: id}, nil
}

// String returns the underlying user ID string.
func (u UserID) String() string {
	return u.value
}
