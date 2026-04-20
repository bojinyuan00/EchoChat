<!--
  文本消息组件
  接收 msg prop，展示文本内容
  撤回消息时显示"消息已撤回"
-->
<template>
  <view class="msg-text-wrap">
    <text v-if="msg.status === 2" class="recalled-text">消息已撤回</text>
    <text v-else class="msg-text" :class="{ 'msg-text-self': isSelf }">{{ msg.content }}</text>
  </view>
</template>

<script>
export default {
  name: 'MsgText',
  props: {
    msg: { type: Object, required: true },
    isSelf: { type: Boolean, default: false }
  }
}
</script>

<style scoped>
.msg-text-wrap {
  word-break: break-word;
  overflow-wrap: break-word;
}
.msg-text {
  font-size: 30rpx;
  line-height: 42rpx;
  color: #1E293B;
}
.msg-text-self {
  color: #FFFFFF;
}
.recalled-text {
  font-size: 24rpx;
  color: #94A3B8;
  font-style: italic;
}
</style>
