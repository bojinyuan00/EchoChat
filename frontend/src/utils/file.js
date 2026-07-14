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

/**
 * 归一化上传资源 URL，解决跨设备访问失败问题
 *
 * 背景：后端 FileService.buildURL 写死返回 `http(s)://{minio.endpoint}/{bucket}/{object}`，
 * 开发环境 endpoint = `localhost:9000`。PC 端因 localhost 指向本机 MinIO 能命中；
 * 手机端的 localhost 是手机自己，不跑 MinIO，导致图片/语音/文件全部加载失败。
 *
 * 解决思路：把所有落到 `*:9000` 的绝对 URL 重写为同源相对路径 `/minio/...`，
 * 由 vite 代理（dev）或 Nginx（prod）转发到真正的 MinIO。
 * 这样无论 origin 是 localhost 还是局域网 IP，所有设备都能走同一条路径。
 *
 * 匹配规则（宽松）：
 *   - 协议：http 或 https
 *   - host：任意（localhost / 127.0.0.1 / 内网 IP / 公网域名皆可）
 *   - 端口：必须是 9000（EchoChat 约定 MinIO 走 9000）
 * 对非 9000 端口的 URL 原样返回。
 *
 * @param {string} url 原始 URL（可能是绝对路径或已经是相对路径）
 * @returns {string} 归一化后的 URL
 */
export function normalizeMediaUrl(url) {
  if (!url || typeof url !== 'string') return url || ''
  // 匹配 `http(s)://<host>:9000` 前缀，整体替换为 `/minio`；其余部分（/bucket/object）原样保留
  return url.replace(/^https?:\/\/[^/]+:9000/i, '/minio')
}
