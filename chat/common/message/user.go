package message

// User 用户账号信息。
// UserPwd 仅在传输阶段承载用户输入，服务端落库前会替换为 bcrypt 哈希。
type User struct {
	UserId     int    `json:"userId"`
	UserPwd    string `json:"userPwd"`
	UserName   string `json:"userName"`
	UserStatus int    `json:"userStatus"`
}
