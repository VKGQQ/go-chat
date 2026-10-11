package model

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"

	"github.com/redis/go-redis/v9"

	"github.com/VKGQQ/goproject_1/chat/common/message"
)

var (
	MyUserDao *UserDao
)

//定义一个UserDao 结构体体
//完成对User 结构体的各种操作

type UserDao struct {
	client *redis.Client
}

// 使用工厂模式，创建一个UserDao实例

func NewUserDao(client *redis.Client) (userDao *UserDao) {
	userDao = &UserDao{
		client: client,
	}
	return
}

// 1. 根据用户id 返回 一个User实例+err
func (this *UserDao) getUserById(ctx context.Context, id int) (user *message.User, err error) {
	//通过给定id 去 redis查询这个用户
	res, err := this.client.HGet(ctx, "users", strconv.Itoa(id)).Result()
	if err != nil {
		//错误!
		if errors.Is(err, redis.Nil) { //表示在 users 哈希中，没有找到对应id
			err = ErrorUserNotExists
		}
		return
	}
	user = &message.User{}
	//这里我们需要把res 反序列化成User实例
	err = json.Unmarshal([]byte(res), user)
	if err != nil {
		fmt.Println("json.Unmarshal err=", err)
		return
	}
	return
}

// 完成登录的校验 Login
// 1. Login 完成对用户的验证
// 2. 如果用户的id和pwd都正确，则返回一个user实例
// 3. 如果用户的id或pwd有错误，则返回对应的错误信息

func (this *UserDao) Login(userId int, userPwd string) (user *message.User, err error) {
	ctx := context.Background()
	user, err = this.getUserById(ctx, userId)
	if err != nil {
		return
	}
	//这时证明这个用户是获取到
	if user.UserPwd != userPwd {
		err = ErrorUserPwd
		return
	}
	return
}

func (this *UserDao) Register(user *message.User) (err error) {
	ctx := context.Background()
	_, err = this.getUserById(ctx, user.UserId)
	if err == nil {
		err = ErrorUserExists
		return
	}
	//这时，说明id在redis还没有，则可以完成注册
	data, err := json.Marshal(user) //序列化
	if err != nil {
		return
	}
	//入库（field 与 getUserById 统一为字符串形式）
	err = this.client.HSet(ctx, "users", strconv.Itoa(user.UserId), string(data)).Err()
	if err != nil {
		fmt.Println("保存注册用户错误 err=", err)
		return
	}
	return
}
