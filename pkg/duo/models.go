package duo

type User struct {
	Email     string `url:"email"`
	FirstName string `url:"firstname"`
	LastName  string `url:"lastname"`
	RealName  string `url:"realname"`
	Status    string `url:"status"`
	UserID    string `json:"user_id"`
	Username  string `url:"username"`
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
	Status  string `json:"status"`
}
