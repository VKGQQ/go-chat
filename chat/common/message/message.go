// Package message 定义客户端与服务端之间的通信协议：
// 消息类型常量、用户状态常量，以及各类消息的结构体。
package message

// 消息类型常量，标识 Message.Type 的取值。
const (
	LoginMesType            = "LoginMes"            // 登录请求
	LoginResMesType         = "LoginResMes"         // 登录响应
	RegisterMesType         = "RegisterMes"         // 注册请求
	RegisterResMesType      = "RegisterResMes"      // 注册响应
	NotifyUserStatusMesType = "NotifyUserStatusMes" // 用户上/下线通知
	SmsMesType              = "SmsMes"              // 群聊消息（客户端上行）
	SmsTransferMesType      = "SmsTransferMes"      // 群聊消息（服务端下行转发）
	SmsPrivateMesType       = "SmsPrivateMes"       // 私聊消息
)

// 用户在线状态常量，标识 NotifyUserStatusMes.Status 的取值。
const (
	UserOnline     = iota // 在线
	UserOffline           // 离线
	UserBusyStatus        // 忙碌
)

// Message 是协议的最外层信封：Type 标识消息类别，
// Data 承载对应消息结构体序列化后的 JSON 串。
type Message struct {
	Type string `json:"type"`
	Data string `json:"data"`
}

// LoginMes 登录请求。
type LoginMes struct {
	UserId   int    `json:"userId"`
	UserPwd  string `json:"userPwd"`
	UserName string `json:"userName"`
}

// LoginResMes 登录响应。
// Code：200 登录成功；403 密码错误；500 用户不存在；505 服务器内部错误。
// UsersId 为当前在线用户 id 列表，供客户端初始化在线列表。
type LoginResMes struct {
	Code    int    `json:"code"`
	UsersId []int  // 在线用户 id 切片
	Error   string `json:"error"`
}

// RegisterMes 注册请求。
type RegisterMes struct {
	User User `json:"user"`
}

// RegisterResMes 注册响应。
// Code：200 注册成功；505 用户已存在；506 服务器内部错误；507 注册信息不合法。
type RegisterResMes struct {
	Code  int    `json:"code"`
	Error string `json:"error"`
}

// NotifyUserStatusMes 用户上/下线通知，由服务端向其他在线用户广播。
type NotifyUserStatusMes struct {
	UserId int `json:"userId"`
	Status int `json:"status"`
}

// SmsMes 群聊消息（客户端上行）。
type SmsMes struct {
	User    `json:"user"`
	Content string `json:"content"`
}

// SmsTransferMes 群聊消息（服务端下行转发）。
type SmsTransferMes struct {
	User    `json:"user"`
	Content string `json:"content"`
}

// SmsPrivateMes 私聊消息。TargetId 为接收者用户 id；
// 发送者 UserId 由服务端按登录会话强制覆盖，客户端上报值不可信。
type SmsPrivateMes struct {
	User     `json:"user"`
	TargetId int    `json:"targetId"`
	Content  string `json:"content"`
}
