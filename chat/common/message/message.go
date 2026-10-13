package message

const (
	LoginMesType            = "LoginMes"
	LoginResMesType         = "LoginResMes"
	RegisterMesType         = "RegisterMes"
	RegisterResMesType      = "RegisterResMes"
	NotifyUserStatusMesType = "NotifyUserStatusMes"
	SmsMesType              = "SmsMes"
	SmsTransferMesType      = "SmsTransferMes"
	SmsPrivateMesType       = "SmsPrivateMes"
)

const (
	UserOnline = iota
	UserOffline
	UserBusyStatus
)

type Message struct {
	Type string `json:"type"` //消息类型
	Data string `json:"data"` //消息的类型
}

type LoginMes struct {
	UserId   int    `json:"userId"`   //用户id
	UserPwd  string `json:"userPwd"`  //用户密码
	UserName string `json:"userName"` //用户名
}

type LoginResMes struct {
	Code    int    `json:"code"` // 返回状态码 500 表示该用户未注册 200表示登录成功
	UsersId []int  // 增加字段，保存用户id的切片
	Error   string `json:"error"` // 返回错误信息
}

type RegisterMes struct {
	User User `json:"user"` //类型就是User结构体.
}
type RegisterResMes struct {
	Code  int    `json:"code"`  // 返回状态码 400 表示该用户已经占有 200表示注册成功
	Error string `json:"error"` // 返回错误信息
}

type NotifyUserStatusMes struct {
	UserId int `json:"userId"`
	Status int `json:"status"`
}

type SmsMes struct {
	User    `json:"user"`
	Content string `json:"content"`
}

type SmsTransferMes struct {
	User    `json:"user"`
	Content string `json:"content"`
}

type SmsPrivateMes struct {
	User     `json:"user"` // 发送者信息（UserId 会被服务端覆盖，和群聊 #11 同理）
	TargetId int           `json:"targetId"` // 接收者用户ID
	Content  string        `json:"content"`
}
