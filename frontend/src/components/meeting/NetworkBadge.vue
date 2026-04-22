<!--
  网络质量徽标（Task 11）

  职责：
  - 渲染 4 格信号柱（类似手机信号图标），根据 level (0-4) 点亮
  - 可选展示文字标签（默认隐藏，hover/长按再显示）
  - 本组件不产生网络数据，由父页面在 Task 15 接入 mediasoup getStats 后喂入

  level 语义：
  - 4 excellent / 3 good / 2 fair / 1 poor / 0 disconnected
-->
<template>
  <view class="badge" :class="levelClass" :title="labelText">
    <view v-for="n in 4" :key="n" class="bar" :class="{ on: n <= level }"
      :style="{ height: `${n * 20}%` }"
    ></view>
    <text v-if="showLabel" class="label">{{ labelText }}</text>
  </view>
</template>

<script setup>
import { computed } from 'vue'

const props = defineProps({
  level: { type: Number, default: 4 },
  showLabel: { type: Boolean, default: false }
})

const LEVEL_LABEL = ['已断开', '很差', '一般', '良好', '优秀']

const labelText = computed(() => LEVEL_LABEL[Math.max(0, Math.min(4, props.level))])

const levelClass = computed(() => {
  if (props.level >= 4) return 'lv-excellent'
  if (props.level >= 3) return 'lv-good'
  if (props.level >= 2) return 'lv-fair'
  if (props.level >= 1) return 'lv-poor'
  return 'lv-off'
})
</script>

<style scoped>
.badge {
  display: inline-flex;
  align-items: flex-end;
  gap: 3rpx;
  height: 28rpx;
  padding: 4rpx 8rpx;
  border-radius: 8rpx;
  background: rgba(0, 0, 0, 0.35);
  backdrop-filter: blur(4px);
  position: relative;
}

.bar {
  width: 6rpx;
  background: rgba(255, 255, 255, 0.25);
  border-radius: 2rpx;
  transition: background-color 0.15s ease;
}

.lv-excellent .bar.on { background: #10B981; }
.lv-good .bar.on { background: #22C55E; }
.lv-fair .bar.on { background: #F59E0B; }
.lv-poor .bar.on { background: #EF4444; }
.lv-off .bar { background: rgba(255, 255, 255, 0.2); }

.label {
  margin-left: 8rpx;
  color: #FFFFFF;
  font-size: 20rpx;
  line-height: 1;
  align-self: center;
}
</style>
