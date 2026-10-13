package model

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

// OfflineMessage 存入 Redis 的离线消息格式。
// 不复用 SmsPrivateMes：它嵌入完整 User（含密码），入库会泄漏敏感信息；
// 离线消息只需要"谁发的、发给谁、内容、时间"这些最小字段。
type OfflineMessage struct {
	SenderId  int       `json:"senderId"`
	TargetId  int       `json:"targetId"`
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
}

var MyOfflineDao *OfflineDao

type OfflineDao struct {
	client *redis.Client
}

func NewOfflineDao(client *redis.Client) *OfflineDao {
	return &OfflineDao{client: client}
}

// offlineKey 一个用户一个 List：offline:msg:{userId}
func offlineKey(userId int) string {
	return "offline:msg:" + strconv.Itoa(userId)
}

// StoreOfflineMessage 追加一条离线消息。
// RPUSH 入队尾，与 LPOP 配对构成 FIFO 队列，用户上线后按发送顺序收到消息。
func (this *OfflineDao) StoreOfflineMessage(msg *OfflineMessage) (err error) {
	ctx, cancel := context.WithTimeout(context.Background(), daoTimeout)
	defer cancel()
	data, err := json.Marshal(msg)
	if err != nil {
		return
	}
	if err = this.client.RPush(ctx, offlineKey(msg.TargetId), data).Err(); err != nil {
		fmt.Println("存离线消息失败 err=", err)
	}
	return
}

// PopOfflineMessage 原子地取出并删除最早的一条离线消息（LPOP）。
// 返回 (nil, nil) 表示队列为空，属正常情况。
// 取+删一步完成，不存在"取了没删"的泄漏窗口；
// 投递失败时调用方应 RequeueOfflineMessage 推回队首。
func (this *OfflineDao) PopOfflineMessage(userId int) (msg *OfflineMessage, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), daoTimeout)
	defer cancel()
	data, err := this.client.LPop(ctx, offlineKey(userId)).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) { // 队列为空
			return nil, nil
		}
		return nil, err
	}
	msg = &OfflineMessage{}
	if err = json.Unmarshal([]byte(data), msg); err != nil {
		fmt.Println("json.Unmarshal err=", err)
		return nil, err
	}
	return
}

// RequeueOfflineMessage 把投递失败的消息推回队首（LPUSH），
// 保持先进先出顺序，等用户下次上线重新投递。
func (this *OfflineDao) RequeueOfflineMessage(msg *OfflineMessage) (err error) {
	ctx, cancel := context.WithTimeout(context.Background(), daoTimeout)
	defer cancel()
	data, err := json.Marshal(msg)
	if err != nil {
		return
	}
	if err = this.client.LPush(ctx, offlineKey(msg.TargetId), data).Err(); err != nil {
		fmt.Println("回推离线消息失败 err=", err)
	}
	return
}
