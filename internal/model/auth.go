package model

const (
	OlympicsAdminGroup string = "olympics-admin"
)

type Claims struct {
	Username string   `json:"preferred_username"`
	Email    string   `json:"email"`
	Groups   []string `json:"groups"`
	Name     string   `json:"given_name"`
	LastName string   `json:"family_name"`
	Expire   int64    `json:"exp"`
	IssuedAt int64    `json:"iat"`
}

// IsAdmin reports whether the caller belongs to the olympics-admin group.
func (c *Claims) IsAdmin() bool {
	for _, g := range c.Groups {
		if g == OlympicsAdminGroup {
			return true
		}
	}
	return false
}
