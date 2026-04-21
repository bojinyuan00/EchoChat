package utils

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"math/big"
)

// meetingCodeDigitGroups 会议号的 3 段 × 3 位数字布局：XXX-XXX-XXX
// 每段由 [0, 1000) 的 crypto/rand 数字零填充为 3 位构成
// 去除连字符后总长度与 constants.MeetingRoomCodeLength (=9) 对齐
const meetingCodeDigitGroups = 3
const meetingCodeDigitPerGroup = 3

// GenerateMeetingRoomCode 生成 9 位随机会议号，格式为 XXX-XXX-XXX
// 使用 crypto/rand 保证足够熵，服务端持久化时可按需去除连字符
// 返回示例：329-471-086
func GenerateMeetingRoomCode() (string, error) {
	parts := make([]string, meetingCodeDigitGroups)
	maxPerGroup := big.NewInt(1000) // 每段 [0, 999]
	for i := 0; i < meetingCodeDigitGroups; i++ {
		n, err := rand.Int(rand.Reader, maxPerGroup)
		if err != nil {
			return "", fmt.Errorf("生成会议号随机数失败: %w", err)
		}
		parts[i] = fmt.Sprintf("%0*d", meetingCodeDigitPerGroup, n.Int64())
	}
	return parts[0] + "-" + parts[1] + "-" + parts[2], nil
}

// GenerateMeetingInviteToken 生成邀请链接用的高熵 Token（32 hex 字符，16 字节随机）
// 用于 Redis key echo:meeting:invite:{token} 的兑换凭证
// Token 不包含会议号等敏感信息，仅作为索引，查询时回查 Redis 值获取完整 payload
func GenerateMeetingInviteToken() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("生成邀请 Token 随机数失败: %w", err)
	}
	return hex.EncodeToString(buf), nil
}
