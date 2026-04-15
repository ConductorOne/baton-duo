package duo

type User struct {
	Email     string `json:"email"`
	FirstName string `json:"firstname"`
	LastName  string `json:"lastname"`
	RealName  string `json:"realname"`
	Status    string `json:"status"`
	UserID    string `json:"user_id"`
	Username  string `json:"username"`
	Created   int64  `json:"created"`
	LastLogin int64  `json:"last_login"`
	Notes     string `json:"notes"`
}

type Group struct {
	Desc    string `json:"desc"`
	GroupID string `json:"group_id"`
	Name    string `json:"name"`
	Status  string `json:"status"`
}

type Admin struct {
	AdminID string `json:"admin_id"`
	Email   string `json:"email"`
	Name    string `json:"name"`
	Role    string `json:"role"`
	RoleID  string `json:"role_id"`
	Status  string `json:"status"`
}

type Account struct {
	Name string `json:"name"`
}

type Role struct {
	RoleID      string `json:"role_id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	InUse       bool   `json:"in_use"`
	IsCustom    bool   `json:"is_custom"`
}
