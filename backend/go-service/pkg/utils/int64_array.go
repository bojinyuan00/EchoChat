// Package utils 提供通用工具类型和函数
package utils

import (
	"database/sql/driver"
	"fmt"
	"strconv"
	"strings"
)

// Int64Array 自定义 PostgreSQL BIGINT[] 类型
// 实现 sql.Scanner 和 driver.Valuer 接口，用于 GORM 与 PostgreSQL 数组字段交互
type Int64Array []int64

// Scan 从数据库读取 PostgreSQL BIGINT[] 格式：{1,2,3}
func (a *Int64Array) Scan(src interface{}) error {
	if src == nil {
		*a = nil
		return nil
	}

	var s string
	switch v := src.(type) {
	case []byte:
		s = string(v)
	case string:
		s = v
	default:
		return fmt.Errorf("Int64Array.Scan: 不支持的类型 %T", src)
	}

	s = strings.TrimSpace(s)
	if s == "{}" || s == "" {
		*a = Int64Array{}
		return nil
	}

	s = strings.Trim(s, "{}")
	parts := strings.Split(s, ",")
	result := make(Int64Array, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		n, err := strconv.ParseInt(p, 10, 64)
		if err != nil {
			return fmt.Errorf("Int64Array.Scan: 解析 '%s' 失败: %w", p, err)
		}
		result = append(result, n)
	}
	*a = result
	return nil
}

// Value 写入数据库时转换为 PostgreSQL BIGINT[] 格式：{1,2,3}
func (a Int64Array) Value() (driver.Value, error) {
	if a == nil {
		return nil, nil
	}
	if len(a) == 0 {
		return "{}", nil
	}

	parts := make([]string, len(a))
	for i, v := range a {
		parts[i] = strconv.FormatInt(v, 10)
	}
	return "{" + strings.Join(parts, ",") + "}", nil
}
