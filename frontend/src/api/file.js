/**
 * 文件上传 API 封装
 * 提供通用上传、图片上传（含缩略图）、语音上传接口
 *
 * BASE_URL 复用 @/utils/request 中的统一配置，保证与业务接口一致
 * （开发环境：http://{hostname}:8085；生产环境：相对路径由 Nginx 反代）
 */

import { useUserStore } from '@/store/user'
import { BASE_URL } from '@/utils/request'

/**
 * 通用文件上传（最大 50MB）
 * @param {string} filePath 本地文件临时路径
 * @param {string} [fileName] 文件名
 * @returns {Promise<{ url, file_name, size }>}
 */
export function uploadFile(filePath, fileName) {
  return _doUpload('/api/v1/upload', filePath, fileName)
}

/**
 * 图片上传（自动生成缩略图，最大 20MB）
 * @param {string} filePath 本地图片临时路径
 * @param {string} [fileName] 文件名
 * @returns {Promise<{ url, thumbnail_url, file_name, size, width, height, mime_type }>}
 */
export function uploadImage(filePath, fileName) {
  return _doUpload('/api/v1/upload/image', filePath, fileName)
}

/**
 * 语音上传（最大 5MB）
 * @param {string} filePath 本地语音文件临时路径
 * @param {number} duration 语音时长（秒）
 * @param {string} [fileName] 文件名
 * @param {Blob} [blob] 可选：H5 端 MediaRecorder 产出的原始 Blob。传入时走 fetch+FormData
 *                     路径，能精确控制 filename/mimeType（`uni.uploadFile` 对 blob: URL 推断的
 *                     filename 通常缺扩展名，会被后端白名单拦下）
 * @returns {Promise<{ url, file_name, size, duration, mime_type }>}
 */
export function uploadVoice(filePath, duration, fileName, blob) {
  if (blob && typeof window !== 'undefined' && typeof FormData !== 'undefined') {
    return _doUploadBlob('/api/v1/upload/voice', blob, fileName || 'voice.webm', {
      duration: String(duration)
    })
  }
  return _doUpload('/api/v1/upload/voice', filePath, fileName, { duration: String(duration) })
}

/**
 * H5 专用：直接用 Blob 上传（避免 uni.uploadFile 对 blob: URL 的 filename 不可控）
 */
function _doUploadBlob(apiPath, blob, fileName, extraFormData = {}) {
  const userStore = useUserStore()
  const token = userStore.token
  const fd = new FormData()
  fd.append('file', blob, fileName)
  Object.entries(extraFormData).forEach(([k, v]) => fd.append(k, v))
  return fetch(`${BASE_URL}${apiPath}`, {
    method: 'POST',
    headers: token ? { Authorization: `Bearer ${token}` } : {},
    body: fd
  }).then(async (resp) => {
    let result
    try {
      result = await resp.json()
    } catch (_) {
      throw new Error(`上传失败（${resp.status}）`)
    }
    if (result.code === 0) return result.data
    throw new Error(result.message || '上传失败')
  })
}

/**
 * 内部上传方法
 */
function _doUpload(apiPath, filePath, fileName, extraFormData = {}) {
  const userStore = useUserStore()
  const token = userStore.token

  return new Promise((resolve, reject) => {
    uni.uploadFile({
      url: `${BASE_URL}${apiPath}`,
      filePath,
      name: 'file',
      formData: {
        ...extraFormData,
        ...(fileName ? { file_name: fileName } : {})
      },
      header: {
        Authorization: token ? `Bearer ${token}` : ''
      },
      success: (res) => {
        try {
          const result = typeof res.data === 'string' ? JSON.parse(res.data) : res.data
          if (result.code === 0) {
            resolve(result.data)
          } else {
            reject(new Error(result.message || '上传失败'))
          }
        } catch (e) {
          reject(new Error('解析上传结果失败'))
        }
      },
      fail: (err) => {
        reject(new Error(err.errMsg || '网络错误'))
      }
    })
  })
}

export default {
  uploadFile,
  uploadImage,
  uploadVoice
}
