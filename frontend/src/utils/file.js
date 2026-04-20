/**
 * 文件工具函数
 * 提供图片压缩、文件大小格式化、文件类型判断等工具
 */

/**
 * 文件大小格式化（自动选择 B/KB/MB/GB 单位）
 * @param {number} bytes 文件大小（字节）
 * @returns {string} 格式化后的文件大小
 */
export function formatFileSize(bytes) {
  if (!bytes || bytes === 0) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB']
  const k = 1024
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + units[i]
}

/**
 * 根据文件扩展名获取文件类型图标信息
 * @param {string} fileName 文件名
 * @returns {{ icon: string, color: string, label: string }}
 */
export function getFileTypeInfo(fileName) {
  if (!fileName) return { icon: 'document', color: '#718096', label: '文件' }

  const ext = fileName.split('.').pop()?.toLowerCase() || ''
  const typeMap = {
    pdf: { icon: 'document', color: '#E53E3E', label: 'PDF' },
    doc: { icon: 'document', color: '#2B6CB0', label: 'Word' },
    docx: { icon: 'document', color: '#2B6CB0', label: 'Word' },
    xls: { icon: 'document', color: '#276749', label: 'Excel' },
    xlsx: { icon: 'document', color: '#276749', label: 'Excel' },
    ppt: { icon: 'document', color: '#C05621', label: 'PPT' },
    pptx: { icon: 'document', color: '#C05621', label: 'PPT' },
    zip: { icon: 'document', color: '#6B46C1', label: 'ZIP' },
    rar: { icon: 'document', color: '#6B46C1', label: 'RAR' },
    '7z': { icon: 'document', color: '#6B46C1', label: '7Z' },
    jpg: { icon: 'image', color: '#38A169', label: '图片' },
    jpeg: { icon: 'image', color: '#38A169', label: '图片' },
    png: { icon: 'image', color: '#38A169', label: '图片' },
    gif: { icon: 'image', color: '#38A169', label: '图片' },
    mp3: { icon: 'sound', color: '#D69E2E', label: '音频' },
    wav: { icon: 'sound', color: '#D69E2E', label: '音频' },
    aac: { icon: 'sound', color: '#D69E2E', label: '音频' },
    m4a: { icon: 'sound', color: '#D69E2E', label: '音频' }
  }

  return typeMap[ext] || { icon: 'document', color: '#718096', label: '文件' }
}

/**
 * 判断文件名是否为可预览类型（PDF/图片）
 * @param {string} fileName 文件名
 * @returns {boolean}
 */
export function isPreviewable(fileName) {
  if (!fileName) return false
  const ext = fileName.split('.').pop()?.toLowerCase() || ''
  return ['pdf', 'jpg', 'jpeg', 'png', 'gif', 'webp'].includes(ext)
}

/**
 * 解析消息 extra JSON
 * @param {string|object} extra 扩展数据（JSON 字符串或对象）
 * @returns {object|null}
 */
export function parseExtra(extra) {
  if (!extra) return null
  if (typeof extra === 'object') return extra
  try {
    return JSON.parse(extra)
  } catch {
    return null
  }
}
