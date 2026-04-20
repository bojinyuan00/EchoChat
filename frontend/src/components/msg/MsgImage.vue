<!--
  图片消息组件
  支持单图/多图（最多 9 张）网格布局 + 点击大图预览
  布局规则：1张=大图, 2/4张=2列, 3/5-9张=3列
-->
<template>
  <view v-if="msg.status === 2" class="recalled-text">消息已撤回</view>
  <view v-else class="msg-image-wrap" :class="gridClass">
    <view
      v-for="(img, idx) in images"
      :key="idx"
      class="img-item"
      @tap="onPreview(idx)"
    >
      <image
        class="img-thumb"
        :src="img.thumbnail_url || img.url"
        mode="aspectFill"
        lazy-load
      />
    </view>
  </view>
</template>

<script>
import { computed } from 'vue'
import { parseExtra } from '@/utils/file'

export default {
  name: 'MsgImage',
  props: {
    msg: { type: Object, required: true },
    isSelf: { type: Boolean, default: false }
  },
  setup(props) {
    const images = computed(() => {
      const extra = parseExtra(props.msg.extra)
      return extra?.images || []
    })

    const gridClass = computed(() => {
      const count = images.value.length
      if (count <= 1) return 'grid-single'
      if (count === 2 || count === 4) return 'grid-2col'
      return 'grid-3col'
    })

    const onPreview = (idx) => {
      const urls = images.value.map(img => img.url)
      uni.previewImage({
        urls,
        current: urls[idx] || urls[0]
      })
    }

    return { images, gridClass, onPreview }
  }
}
</script>

<style scoped>
.msg-image-wrap {
  display: flex;
  flex-wrap: wrap;
  gap: 8rpx;
  max-width: 500rpx;
}
.grid-single {
  width: 400rpx;
}
.grid-single .img-item {
  width: 100%;
  min-width: 200rpx;
  max-width: 400rpx;
}
.grid-single .img-thumb {
  width: 400rpx;
  height: 300rpx;
  border-radius: 12rpx;
  display: block;
}
.grid-2col {
  width: 320rpx;
}
.grid-2col .img-item {
  width: calc(50% - 4rpx);
}
.grid-2col .img-thumb {
  width: 156rpx;
  height: 156rpx;
  border-radius: 8rpx;
  display: block;
}
.grid-3col {
  width: 360rpx;
}
.grid-3col .img-item {
  width: calc(33.33% - 6rpx);
}
.grid-3col .img-thumb {
  width: 110rpx;
  height: 110rpx;
  border-radius: 8rpx;
  display: block;
}
.img-item {
  overflow: hidden;
}
.recalled-text {
  font-size: 24rpx;
  color: #94A3B8;
  font-style: italic;
}
</style>
