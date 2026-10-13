package model

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"

	"github.com/VKGQQ/go-chat/chat/common/message"
)

// newTestUserDao 每个用例使用独立的 miniredis 实例（内存版 Redis），
// 无需真实 Redis 服务，测试快且互不干扰。
func newTestUserDao(t *testing.T) (*UserDao, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	return NewUserDao(client), mr
}

func testUser(id int) *message.User {
	return &message.User{UserId: id, UserPwd: "secret123", UserName: "tester"}
}

func TestUserDao_RegisterAndLogin(t *testing.T) {
	dao, _ := newTestUserDao(t)

	if err := dao.Register(testUser(100)); err != nil {
		t.Fatalf("注册失败: %v", err)
	}
	user, err := dao.Login(100, "secret123")
	if err != nil {
		t.Fatalf("正确密码登录应成功: %v", err)
	}
	if user.UserName != "tester" {
		t.Errorf("登录返回的用户昵称不一致: %q", user.UserName)
	}
}

func TestUserDao_RegisterDuplicate(t *testing.T) {
	dao, _ := newTestUserDao(t)

	if err := dao.Register(testUser(100)); err != nil {
		t.Fatalf("首次注册失败: %v", err)
	}
	if err := dao.Register(testUser(100)); !errors.Is(err, ErrorUserExists) {
		t.Fatalf("重复注册应返回 ErrorUserExists, got %v", err)
	}
}

func TestUserDao_LoginErrors(t *testing.T) {
	dao, _ := newTestUserDao(t)
	_ = dao.Register(testUser(100))

	tests := []struct {
		name    string
		userId  int
		pwd     string
		wantErr error
	}{
		{"密码错误", 100, "wrongpwd", ErrorUserPwd},
		{"用户不存在", 999, "secret123", ErrorUserNotExists},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := dao.Login(tt.userId, tt.pwd); !errors.Is(err, tt.wantErr) {
				t.Errorf("期望 %v, got %v", tt.wantErr, err)
			}
		})
	}
}

// TestUserDao_PasswordStoredHashed 验证安全底线：落库的必须是 bcrypt 哈希，不是明文。
func TestUserDao_PasswordStoredHashed(t *testing.T) {
	dao, mr := newTestUserDao(t)

	if err := dao.Register(testUser(100)); err != nil {
		t.Fatalf("注册失败: %v", err)
	}
	raw := mr.HGet("users", "100")
	if raw == "" {
		t.Fatal("miniredis 中应存在该用户")
	}
	if strings.Contains(raw, "secret123") {
		t.Fatal("明文密码出现在存储中")
	}
	var stored message.User
	if err := json.Unmarshal([]byte(raw), &stored); err != nil {
		t.Fatalf("存储格式应为合法 JSON: %v", err)
	}
	if bcrypt.CompareHashAndPassword([]byte(stored.UserPwd), []byte("secret123")) != nil {
		t.Fatal("存储的哈希无法用原密码通过 bcrypt 校验")
	}
}
