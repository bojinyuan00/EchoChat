<!--
  会议视频网格（Task 11）

  职责：
  - 根据成员数量自适应网格布局（1/2/3-4/5-6/7-9/10+）
  - 手机端竖排为主，桌面端以 16:9 单元格填充
  - 不关心业务数据，仅负责排版；视频块由 slot #tile 自定义渲染，保持数据与组件解耦
-->
<template>
  <view class="grid-root">
    <view class="grid" :class="layoutClass" :style="gridStyle">
      <view
        v-for="(tile, idx) in tiles"
        :key="tile.key || tile.userId || idx"
        class="grid-cell"
      >
        <slot name="tile" :tile="tile" :index="idx" />
      </view>
    </view>
    <view v-if="!tiles.length" class="empty">
      <text>暂无成员，等待其他人加入…</text>
    </view>
  </view>
</template>

<script setup>
import { computed } from 'vue'

const props = defineProps({
  tiles: { type: Array, default: () => [] }
})

/** 根据成员数量计算列数 */
const columns = computed(() => {
  const n = props.tiles.length
  if (n <= 1) return 1
  if (n <= 4) return 2
  if (n <= 9) return 3
  if (n <= 16) return 4
  return Math.ceil(Math.sqrt(n))
})

const rows = computed(() => {
  const n = props.tiles.length
  if (!n) return 1
  return Math.ceil(n / columns.value)
})

const gridStyle = computed(() => ({
  'grid-template-columns': `repeat(${columns.value}, minmax(0, 1fr))`,
  'grid-template-rows': `repeat(${rows.value}, minmax(0, 1fr))`
}))

const layoutClass = computed(() => `layout-${Math.min(props.tiles.length, 10)}`)
</script>

<style scoped>
.grid-root {
  position: relative;
  width: 100%;
  height: 100%;
  padding: 16rpx;
  box-sizing: border-box;
  overflow: hidden;
}

.grid {
  width: 100%;
  height: 100%;
  display: grid;
  gap: 12rpx;
}

.grid-cell {
  position: relative;
  min-width: 0;
  min-height: 0;
  overflow: hidden;
}

.empty {
  position: absolute;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  color: rgba(255, 255, 255, 0.55);
  font-size: 26rpx;
  letter-spacing: 1rpx;
}

/* 移动端单人时占满整屏 */
@media (max-width: 640px) {
  .grid.layout-1 { grid-template-rows: 1fr !important; }
  .grid.layout-2 {
    grid-template-columns: 1fr !important;
    grid-template-rows: 1fr 1fr !important;
  }
}
</style>
