<!--
  网络质量徽标（Task 11 / Task 15 波浪动效升级）

  职责：
  - Task 15 原创特色 6：使用 3 条横向 SVG 波浪曲线代替传统格子信号条
  - 根据 level (0-4) 动态点亮可见条数 + 颜色
  - 3 条波浪以相位差 0 / -0.3s / -0.6s 循环流动，形成"呼吸感"
  - 可选文字标签（title tooltip 保留，默认不显示 inline label）

  level 语义：
  - 4 excellent / 3 good / 2 fair / 1 poor / 0 disconnected
-->
<template>
  <view class="badge" :class="levelClass" :title="labelText">
    <view v-if="level <= 0" class="dc-icon" aria-label="已断开">
      <svg viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round">
        <line x1="4" y1="12" x2="20" y2="12"></line>
        <circle cx="12" cy="17" r="1" fill="currentColor"></circle>
      </svg>
    </view>
    <template v-else>
      <svg
        v-for="n in visibleLines"
        :key="n"
        class="wave"
        :class="`wave-${n}`"
        viewBox="0 0 24 12"
        width="24"
        height="10"
        fill="none"
        stroke="currentColor"
        stroke-width="2"
        stroke-linecap="round"
      >
        <path d="M 0 6 C 3 2, 6 10, 9 6 S 15 2, 18 6 S 22 10, 24 6" />
      </svg>
    </template>
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

/** level → 可见条数（4=3 条 / 3=2 条 / 2/1=1 条，0 由 dc-icon 承接） */
const visibleLines = computed(() => {
  if (props.level >= 4) return 3
  if (props.level >= 3) return 2
  if (props.level >= 1) return 1
  return 0
})

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
  align-items: center;
  gap: 2rpx;
  height: 36rpx;
  padding: 4rpx 10rpx;
  border-radius: 10rpx;
  background: rgba(0, 0, 0, 0.35);
  backdrop-filter: blur(4px);
  position: relative;
}

.wave {
  animation: wave-flow 1.2s linear infinite;
  flex-shrink: 0;
  /* 三条波浪相位差：通过 animation-delay 负值错开 */
}
.wave-1 { animation-delay: 0s; }
.wave-2 { animation-delay: -0.3s; }
.wave-3 { animation-delay: -0.6s; }

@keyframes wave-flow {
  from { transform: translateX(0); }
  to   { transform: translateX(-6px); }
}

.lv-excellent { color: #10B981; }
.lv-good      { color: #22C55E; }
.lv-fair      { color: #F59E0B; }
.lv-poor      { color: #EF4444; }
.lv-off       { color: #EF4444; }

.dc-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 24rpx;
}

.label {
  margin-left: 8rpx;
  color: #FFFFFF;
  font-size: 20rpx;
  line-height: 1;
  align-self: center;
}

/* 可访问性：减少动效偏好下停止流动 */
@media (prefers-reduced-motion: reduce) {
  .wave { animation: none; }
}
</style>
