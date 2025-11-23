package common

import (
	"crypto/rand"
	"encoding/base64"
	"sync"
	"time"
)

// NonceStore 存储已使用的nonce，防止重放攻击
type NonceStore struct {
	used  sync.Map // map[string]time.Time
	mutex sync.Mutex
}

var globalNonceStore = &NonceStore{}

// GenerateNonce 生成一个加密安全的随机nonce
func GenerateNonce() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		// Fallback: 使用时间戳
		return base64.URLEncoding.EncodeToString([]byte(time.Now().String()))
	}
	return base64.URLEncoding.EncodeToString(b)
}

// ValidateNonce 验证nonce是否有效（未被使用且在时间窗口内）
// timeWindow: 时间窗口（秒），建议300秒（5分钟）
func ValidateNonce(nonce string, timestamp int64, timeWindow int64) bool {
	if nonce == "" {
		return false
	}

	globalNonceStore.mutex.Lock()
	defer globalNonceStore.mutex.Unlock()

	// 检查时间窗口
	now := time.Now().Unix()
	if now-timestamp > timeWindow || timestamp-now > timeWindow {
		return false
	}

	// 检查nonce是否已使用
	if _, exists := globalNonceStore.used.Load(nonce); exists {
		return false
	}

	// 标记nonce为已使用，设置过期时间
	globalNonceStore.used.Store(nonce, time.Now().Add(time.Duration(timeWindow)*time.Second))

	return true
}

// CleanupExpiredNonces 清理过期的nonce（定期调用）
func CleanupExpiredNonces() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		now := time.Now()
		globalNonceStore.used.Range(func(key, value interface{}) bool {
			expireTime := value.(time.Time)
			if now.After(expireTime) {
				globalNonceStore.used.Delete(key)
			}
			return true
		})
	}
}

func init() {
	// 启动清理协程
	go CleanupExpiredNonces()
}
