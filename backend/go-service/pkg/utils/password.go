package utils

import "golang.org/x/crypto/bcrypt"

// HashPassword 使用 bcrypt 对明文密码进行加密
// cost 使用默认值 10，在安全性和性能之间取得平衡
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

// CheckPassword 校验明文密码是否与 bcrypt 哈希匹配
// 返回 true 表示密码正确，false 表示密码错误或哈希无效
func CheckPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}
