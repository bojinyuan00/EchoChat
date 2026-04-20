<!--
  输入栏"+"展开面板
  包含图片和文件两个入口
  网格布局，点击后触发对应的选择事件
-->
<template>
  <view class="more-panel" v-if="visible">
    <view class="panel-grid">
      <view class="panel-item" @tap="onChooseImage">
        <view class="item-icon" style="background-color: #DBEAFE;">
          <uni-icons type="image" :size="28" color="#2563EB" />
        </view>
        <text class="item-label">图片</text>
      </view>
      <view class="panel-item" @tap="onChooseFile">
        <view class="item-icon" style="background-color: #FEE2E2;">
          <uni-icons type="paperclip" :size="28" color="#DC2626" />
        </view>
        <text class="item-label">文件</text>
      </view>
    </view>
  </view>
</template>

<script>
export default {
  name: 'MorePanel',
  props: {
    visible: { type: Boolean, default: false }
  },
  emits: ['choose-image', 'choose-file'],
  setup(props, { emit }) {
    const onChooseImage = () => emit('choose-image')
    const onChooseFile = () => emit('choose-file')
    return { onChooseImage, onChooseFile }
  }
}
</script>

<style scoped>
.more-panel {
  background-color: #F8FAFC;
  border-top: 1rpx solid #E2E8F0;
  padding: 24rpx;
  padding-bottom: calc(24rpx + env(safe-area-inset-bottom, 0));
}
.panel-grid {
  display: flex;
  gap: 32rpx;
}
.panel-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 12rpx;
}
.item-icon {
  width: 100rpx;
  height: 100rpx;
  border-radius: 24rpx;
  display: flex;
  align-items: center;
  justify-content: center;
}
.item-icon:active { opacity: 0.7; }
.item-label {
  font-size: 22rpx;
  color: #64748B;
}
</style>
