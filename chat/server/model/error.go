// Package model 提供服务端的数据访问层（DAO），
// 封装用户账号与离线消息在 Redis 上的读写。
package model

import (
	"errors"
)

// 账号相关的业务错误，供登录/注册流程用 errors.Is 判定。
var (
	ErrorUserNotExists = errors.New("用户不存在")
	ErrorUserExists    = errors.New("用户已经存在")
	ErrorUserPwd       = errors.New("密码不正确")
)
