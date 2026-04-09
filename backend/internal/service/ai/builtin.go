package ai

import (
	"crypto/aes"
	"crypto/cipher"
	cryptorand "crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"io"
	"math/rand/v2"
)

// 内置免费 Key（智谱）的 AES-GCM 密文
// 暂未配置内置 Key，开发者可使用 EncryptBuiltInKey 生成密文后填入此处
const builtInEncryptedKey = "XdSkHxsdio1XcIq3Fggd5yKNW7XWhSB2X4s0XYcKZSTuQal3JSmEODaeFAhch49hMmuC8Tf9gAOmmN4VihzANkzYWTYMR859evS5UeY="

// 内置免费文本模型清单（仅 chat 类型可用于 Slug 生成）
var builtInChatModels = []string{
	"glm-4.7-flash",
	"glm-4-flash-250414",
}

// 内置 Key 使用的协议端点
const builtInBaseURL = "https://open.bigmodel.cn/api/paas/v4"

// PickBuiltInModel 随机选择一个内置免费模型，分摊调用压力
func PickBuiltInModel() string {
	return builtInChatModels[rand.IntN(len(builtInChatModels))]
}

// BuiltInModels 返回所有内置免费模型清单（前端展示用）
func BuiltInModels() []string {
	return append([]string{}, builtInChatModels...)
}

// NewBuiltInProvider 返回内置 Key 使用的 provider 实例（OpenAI 兼容协议）
func NewBuiltInProvider() Provider {
	return &openAICompatProvider{baseURL: builtInBaseURL}
}

// DecryptBuiltInKey 解密内置 Key，失败或未配置则返回空
func DecryptBuiltInKey() string {
	if builtInEncryptedKey == "" {
		return ""
	}
	data, err := base64.StdEncoding.DecodeString(builtInEncryptedKey)
	if err != nil {
		return ""
	}
	key := deriveBuiltInEncryptKey()
	block, err := aes.NewCipher(key)
	if err != nil {
		return ""
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return ""
	}
	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return ""
	}
	plaintext, err := gcm.Open(nil, data[:nonceSize], data[nonceSize:], nil)
	if err != nil {
		return ""
	}
	return string(plaintext)
}

// EncryptBuiltInKey 将明文 Key 加密为 Base64 密文
// 开发者使用：调用此函数得到密文后填入 builtInEncryptedKey 常量
func EncryptBuiltInKey(plainKey string) (string, error) {
	key := deriveBuiltInEncryptKey()
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(cryptorand.Reader, nonce); err != nil {
		return "", err
	}
	ciphertext := gcm.Seal(nonce, nonce, []byte(plainKey), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// deriveBuiltInEncryptKey 从 App 名称派生 16 字节 AES-128 加密密钥
func deriveBuiltInEncryptKey() []byte {
	h := sha256.Sum256([]byte("Gridea Pro"))
	return h[:16]
}
