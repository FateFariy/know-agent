<script lang="ts" setup>
/**
 * FeedbackButtons：复制 / 点赞 / 点踩按钮。
 * 与 React 版 FeedbackButtons 行为一致：再点一次取消当前态。
 * 图标使用内联 SVG，避免 Element Plus 不提供 ThumbsUp/ThumbsDown 带来的依赖问题。
 */
import {computed} from 'vue'
import {DocumentCopy} from '@element-plus/icons-vue'
import {ElMessage} from 'element-plus'
import {type ChatMessage, useChatStore} from '@/stores/chat'

const props = defineProps<{
  messageId: string
  feedback: ChatMessage['feedback']
  content: string
  alwaysVisible?: boolean
}>()

const store = useChatStore()

const containerClass = computed(() => ({
  'feedback-buttons': true,
  'feedback-buttons--visible': props.alwaysVisible
}))

function handleFeedback(value: 'like' | 'dislike') {
  const next = props.feedback === value ? null : value
  store.submitFeedback(props.messageId, next).catch(() => null)
}

async function handleCopy() {
  try {
    await navigator.clipboard.writeText(props.content)
    ElMessage.success('复制成功')
  } catch {
    ElMessage.error('复制失败')
  }
}
</script>

<template>
  <div :class="containerClass">
    <button
      aria-label="复制内容"
      class="feedback-buttons__btn"
      type="button"
      @click="handleCopy"
    >
      <el-icon>
        <DocumentCopy/>
      </el-icon>
    </button>
    <button
      :class="{ 'is-active-like': feedback === 'like' }"
      aria-label="点赞"
      class="feedback-buttons__btn"
      type="button"
      @click="handleFeedback('like')"
    >
      <svg fill="none" height="16" stroke="currentColor" stroke-linecap="round" stroke-linejoin="round"
           stroke-width="2" viewBox="0 0 24 24" width="16">
        <path d="M7 10v12"/>
        <path
          d="M15 5.88 14 10h5.83a2 2 0 0 1 1.92 2.56l-2.33 8A2 2 0 0 1 17.5 22H7a2 2 0 0 1-2-2v-8a2 2 0 0 1 2-2h2.76a2 2 0 0 0 1.79-1.11L15 2h0a3.13 3.13 0 0 1 3 3.88Z"/>
      </svg>
    </button>
    <button
      :class="{ 'is-active-dislike': feedback === 'dislike' }"
      aria-label="点踩"
      class="feedback-buttons__btn"
      type="button"
      @click="handleFeedback('dislike')"
    >
      <svg fill="none" height="16" stroke="currentColor" stroke-linecap="round" stroke-linejoin="round"
           stroke-width="2" viewBox="0 0 24 24" width="16">
        <path d="M17 14V2"/>
        <path
          d="M9 18.12 10 14H4.17a2 2 0 0 1-1.92-2.56l2.33-8A2 2 0 0 1 6.5 2H17a2 2 0 0 1 2 2v8a2 2 0 0 1-2 2h-2.76a2 2 0 0 0-1.79 1.11L9 22h0a3.13 3.13 0 0 1-3-3.88Z"/>
      </svg>
    </button>
  </div>
</template>

<style scoped>
.feedback-buttons {
  display: flex;
  align-items: center;
  gap: 4px;
  opacity: 0;
  transition: opacity 0.2s ease;
}

.feedback-buttons--visible {
  opacity: 1;
}

.feedback-buttons:hover {
  opacity: 1;
}

.feedback-buttons__btn {
  width: 32px;
  height: 32px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border: 0;
  border-radius: 6px;
  background: transparent;
  color: #999999;
  cursor: pointer;
  transition: background 0.15s ease, color 0.15s ease;
  font-size: 16px;
}

.feedback-buttons__btn:hover {
  background: #f5f5f5;
  color: #666666;
}

.feedback-buttons__btn.is-active-like {
  color: #10b981;
}

.feedback-buttons__btn.is-active-like:hover {
  color: #10b981;
}

.feedback-buttons__btn.is-active-dislike {
  color: #ef4444;
}

.feedback-buttons__btn.is-active-dislike:hover {
  color: #ef4444;
}
</style>
