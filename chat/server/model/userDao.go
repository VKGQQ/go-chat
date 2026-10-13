package model

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"

	"github.com/VKGQQ/goproject_1/chat/common/message"
)

// daoTimeout 单次 Redis 调用的最长等待时间。
// context.Background() 永不超时：Redis 假死（非宕机，而是卡住不响应）时，
// 登录/注册请求会无限期挂起，对应客户端永远等不到回复。
// 所有跨网络调用都必须有截止时间，到点未完成自动取消并返回 context.DeadlineExceeded。
const daoTimeout = 3 * time.Second

// MyUserDao 全局唯一的 UserDao 实例，在 main 中初始化 Redis 后注入。
var MyUserDao *UserDao

// UserDao 用户账号的数据访问对象，基于 Redis Hash "users" 存取。
type UserDao struct {
	client *redis.Client
}

// NewUserDao 创建 UserDao 实例。
func NewUserDao(client *redis.Client) (userDao *UserDao) {
	userDao = &UserDao{
		client: client,
	}
	return
}

// getUserById 根据用户 id 查询账号；用户不存在时返回 ErrorUserNotExists。
func (this *UserDao) getUserById(ctx context.Context, id int) (user *message.User, err error) {
	res, err := this.client.HGet(ctx, "users", strconv.Itoa(id)).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			err = ErrorUserNotExists
		}
		return
	}
	user = &message.User{}
	err = json.Unmarshal([]byte(res), user)
	if err != nil {
		fmt.Println("json.Unmarshal err=", err)
		return
	}
	return
}

// Login 校验用户 id 与密码。
// 校验通过返回用户实例；用户不存在返回 ErrorUserNotExists，密码错误返回 ErrorUserPwd。
func (this *UserDao) Login(userId int, userPwd string) (user *message.User, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), daoTimeout)
	defer cancel() // 释放定时器资源，无论成功失败都必须调用
	user, err = this.getUserById(ctx, userId)
	if err != nil {
		return
	}
	// 库里存的是 bcrypt 哈希，用 CompareHashAndPassword 做恒定时间比较。
	// 它返回的错误不区分“密码不匹配”和“哈希格式异常”，统一按密码错误处理，
	// 避免向外部暴露存储细节
	if bcrypt.CompareHashAndPassword([]byte(user.UserPwd), []byte(userPwd)) != nil {
		err = ErrorUserPwd
		return
	}
	return
}

// Register 注册新用户。
// 白名单判断（fail closed）：只有“确认用户不存在”才允许注册。
// Redis 超时、宕机等未知错误说明无法确定用户是否存在，
// 此时放行会覆盖已有账号，必须原样返回错误。
func (this *UserDao) Register(user *message.User) (err error) {
	ctx, cancel := context.WithTimeout(context.Background(), daoTimeout)
	defer cancel() // 释放定时器资源，无论成功失败都必须调用
	_, err = this.getUserById(ctx, user.UserId)
	if err == nil {
		err = ErrorUserExists
		return
	}
	if !errors.Is(err, ErrorUserNotExists) {
		return
	}
	// 确认用户不存在，清掉业务错误后放行注册
	err = nil
	// 明文密码绝不落盘：入库前替换为 bcrypt 哈希。
	// bcrypt 慢哈希（2^cost 轮运算）且随机盐内嵌在结果里，
	// 即使 Redis 数据泄漏，攻击者也无法低成本还原明文、彩虹表直接失效
	hashedPwd, err := bcrypt.GenerateFromPassword([]byte(user.UserPwd), bcrypt.DefaultCost)
	if err != nil {
		return
	}
	user.UserPwd = string(hashedPwd)
	data, err := json.Marshal(user)
	if err != nil {
		return
	}
	// field 与 getUserById 统一使用 id 的字符串形式
	err = this.client.HSet(ctx, "users", strconv.Itoa(user.UserId), string(data)).Err()
	if err != nil {
		fmt.Println("保存注册用户错误 err=", err)
		return
	}
	return
}
