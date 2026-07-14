<!--
  文件消息组件
  展示文件图标 + 文件名 + 大小 + 下载/预览操作
  根据文件扩展名显示不同颜色的图标
-->
<template>
  <view v-if="msg.status === 2" class="recalled-text">消息已撤回</view>
  <view v-else class="msg-file-wrap" :class="{ 'file-self': isSelf }" @tap="onTap">
    <view class="file-icon-wrap" :style="{ backgroundColor: fileInfo.color + '20' }">
      <text class="file-icon-label" :style="{ color: fileInfo.color }">{{ fileInfo.label }}</text>
    </view>
    <view class="file-detail">
      <text class="file-name" :class="{ 'file-name-self': isSelf }">{{ fileData.file_name || '未知文件' }}</text>
      <text class="file-size" :class="{ 'file-size-self': isSelf }">{{ formattedSize }}</text>
    </view>
  </view>
</template>

<script>
import { computed } from 'vue'
import { parseExtra, formatFileSize, getFileTypeInfo, isPreviewable, normalizeMediaUrl } from '@/utils/file'

export default {
  name: 'MsgFile',
  props: {
    msg: { type: Object, required: true },
    isSelf: { type: Boolean, default: false }
  },
  setup(props) {
    const fileData = computed(() => {
      const extra = parseExtra(props.msg.extra)
      return extra?.file || {}
    })

    const fileInfo = computed(() => getFileTypeInfo(fileData.value.file_name))
    const formattedSize = computed(() => formatFileSize(fileData.value.size))

    const onTap = () => {
      const url = normalizeMediaUrl(fileData.value.url)
      if (!url) return

      const fileName = fileData.value.file_name || ''

      if (isPreviewable(fileName)) {
        // #ifdef H5
        window.open(url, '_blank')
        return
        // #endif
        // #ifndef H5
        uni.downloadFile({
          url,
          success: (res) => {
            if (res.statusCode === 200) {
              uni.openDocument({
                filePath: res.tempFilePath,
                showMenu: true,
                fail: () => uni.showToast({ title: '无法预览此文件', icon: 'none' })
              })
            }
          },
          fail: () => uni.showToast({ title: '下载失败', icon: 'none' })
        })
        // #endif
      } else {
        // #ifdef H5
        const a = document.createElement('a')
        a.href = url
        a.download = fileName
        a.target = '_blank'
        document.body.appendChild(a)
        a.click()
        document.body.removeChild(a)
        // #endif
        // #ifndef H5
        uni.downloadFile({
          url,
          success: (res) => {
            uni.saveFile({
              tempFilePath: res.tempFilePath,
              success: () => uni.showToast({ title: '文件已保存', icon: 'success' }),
              fail: () => uni.showToast({ title: '保存失败', icon: 'none' })
            })
          }
        })
        // #endif
      }
    }

    return { fileData, fileInfo, formattedSize, onTap }
  }
}
</script>

<style scoped>
.msg-file-wrap {
  display: flex;
  align-items: center;
  gap: 16rpx;
  min-width: 300rpx;
  max-width: 450rpx;
  cursor: pointer;
}
.file-icon-wrap {
  flex-shrink: 0;
  width: 80rpx;
  height: 80rpx;
  border-radius: 16rpx;
  display: flex;
  align-items: center;
  justify-content: center;
}
.file-icon-label {
  font-size: 22rpx;
  font-weight: 700;
}
.file-detail {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 4rpx;
  overflow: hidden;
}
.file-name {
  font-size: 26rpx;
  color: #1E293B;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.file-name-self { color: #FFFFFF; }
.file-size {
  font-size: 22rpx;
  color: #94A3B8;
}
.file-size-self { color: rgba(255, 255, 255, 0.7); }
.recalled-text {
  font-size: 24rpx;
  color: #94A3B8;
  font-style: italic;
}
</style>
