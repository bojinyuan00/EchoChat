/**
 * 文件上传模块 API
 *
 * 对应后端路由：POST /api/v1/upload
 * 使用 uni.uploadFile 上传文件到 MinIO
 */

import { BASE_URL } from '@/utils/request'
import { getToken } from '@/utils/storage'

/**
 * 上传文件
 * @param {string} filePath - 本地文件路径（如从 uni.chooseImage 获取）
 * @returns {Promise<Object>} 返回 { url: '...' }
 */
const uploadFile = (filePath) => {
  return new Promise((resolve, reject) => {
    const token = getToken()
    uni.uploadFile({
      url: `${BASE_URL}/api/v1/upload`,
      filePath,
      name: 'file',
      header: {
        Authorization: token ? `Bearer ${token}` : ''
      },
      success: (res) => {
        if (res.statusCode === 200) {
          try {
            const data = JSON.parse(res.data)
            if (data.code === 0) {
              resolve(data)
            } else {
              uni.showToast({ title: data.message || '上传失败', icon: 'none' })
              reject(data)
            }
          } catch {
            uni.showToast({ title: '响应解析失败', icon: 'none' })
            reject({ code: -1, message: '响应解析失败' })
          }
        } else {
          uni.showToast({ title: '上传失败', icon: 'none' })
          reject({ code: res.statusCode, message: '上传失败' })
        }
      },
      fail: (err) => {
        uni.showToast({ title: '网络异常', icon: 'none' })
        reject({ code: -1, message: err.errMsg || '网络异常' })
      }
    })
  })
}

export default {
  uploadFile
}
