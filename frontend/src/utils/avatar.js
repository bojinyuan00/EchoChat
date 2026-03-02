/**
 * 头像工具函数
 *
 * 提供头像颜色生成和首字母提取功能，供联系人相关页面统一使用。
 * 颜色基于用户名首字符 charCode 做确定性映射，保证同一用户每次展示颜色一致。
 */

export const AVATAR_COLORS = ['#7C3AED', '#2563EB', '#0891B2', '#059669', '#D97706', '#DC2626', '#7C3AED', '#4F46E5']

/** 根据用户名生成确定性头像背景色 */
export const getAvatarColor = (name) => {
  if (!name) return AVATAR_COLORS[0]
  const code = name.charCodeAt(0)
  return AVATAR_COLORS[code % AVATAR_COLORS.length]
}

/** 提取用户名首字母（大写） */
export const getInitial = (name) => {
  if (!name) return '?'
  return name.charAt(0).toUpperCase()
}
