<script lang="ts" setup>
/**
 * MessageItem：单条消息渲染。
 * - 用户消息：右对齐气泡；
 * - 助手消息：左侧展示，深度思考折叠面板 + Markdown 正文 + 反馈按钮。
 */
import {computed, ref} from 'vue'
import {ArrowDown} from '@element-plus/icons-vue'
import FeedbackButtons from './FeedbackButtons.vue'
import MarkdownRenderer from './MarkdownRenderer.vue'
import ThinkingIndicator from './ThinkingIndicator.vue'
import type {ChatMessage} from '@/stores/chat'

const props = defineProps<{
  message: ChatMessage
  isLast?: boolean
}>()

const thinkingExpanded = ref(false)
const isUser = computed(() => props.message.role === 'user')
const isThinking = computed(() => Boolean(props.message.isThinking))
const hasThinking = computed(() => Boolean(props.message.thinking && props.message.thinking.trim().length > 0))
const hasContent = computed(() => props.message.content.trim().length > 0)
const isWaiting = computed(() => props.message.status === 'streaming' && !isThinking.value && !hasContent.value)
const isError = computed(() => props.message.status === 'error')
const isCancelled = computed(() => props.message.status === 'cancelled')
const thinkingDuration = computed(() => props.message.thinkingDuration ? `${props.message.thinkingDuration}秒` : '')
const showFeedback = computed(() =>
  props.message.role === 'assistant' &&
  props.message.status !== 'streaming' &&
  Boolean(props.message.id) &&
  !props.message.id.startsWith('assistant-')
)
</script>

<template>
  <div v-if="isUser" class="message-item message-item--user">
    <div class="user-bubble">
      <p class="user-bubble__text">{{ message.content }}</p>
    </div>
  </div>

  <div v-else class="message-item message-item--assistant">
    <div class="assistant-stack">
      <ThinkingIndicator
        v-if="isThinking"
        :content="message.thinking"
        :duration="message.thinkingDuration"
      />

      <div v-else-if="hasThinking" class="thinking-panel">
        <button
          :aria-expanded="thinkingExpanded"
          class="thinking-panel__header"
          type="button"
          @click="thinkingExpanded = !thinkingExpanded"
        >
          <span class="thinking-panel__icon">
            <svg fill="none" height="16" stroke="currentColor" stroke-linecap="round" stroke-linejoin="round"
                 stroke-width="2" viewBox="0 0 24 24" width="16"><path
              d="M9 6a3 3 0 0 1 3-3v0a3 3 0 0 1 3 3v0a3 3 0 0 1-3 3"/><path
              d="M9 18a3 3 0 0 0 3 3v0a3 3 0 0 0 3-3v0a3 3 0 0 0-3-3"/><path d="M3 12h3"/><path
              d="M18 12h3"/><path d="M5.6 5.6l2.1 2.1"/><path d="M16.3 16.3l2.1 2.1"/><path
              d="M5.6 18.4l2.1-2.1"/><path d="M16.3 7.7l2.1-2.1"/></svg>
          </span>
          <span class="thinking-panel__title">深度思考</span>
          <span v-if="thinkingDuration" class="thinking-panel__duration">{{
              thinkingDuration
            }}</span>
          <el-icon
            :class="{ 'thinking-panel__caret--open': thinkingExpanded }"
            class="thinking-panel__caret"
          >
            <ArrowDown/>
          </el-icon>
        </button>
        <div v-if="thinkingExpanded" class="thinking-panel__body">
          <div class="thinking-panel__content">{{ message.thinking }}</div>
        </div>
      </div>

      <div class="assistant-body">
        <div v-if="isWaiting" aria-label="思考中" class="ai-wait">
          <span class="ai-wait__dot"/>
          <span class="ai-wait__dot"/>
          <span class="ai-wait__dot"/>
        </div>
        <MarkdownRenderer v-if="hasContent" :content="message.content"/>
        <p v-if="isError" class="message-notice message-notice--error">生成已中断。</p>
        <p v-else-if="isCancelled" class="message-notice message-notice--cancelled">（已停止生成）</p>
        <FeedbackButtons
          v-if="showFeedback"
          :always-visible="Boolean(isLast)"
          :content="message.content"
          :feedback="message.feedback ?? null"
          :message-id="message.id"
        />
      </div>
    </div>
  </div>
</template>

<style scoped>
.message-item {
  display: flex;
  width: 100%;
}

.message-item--user {
  justify-content: flex-end;
}

.user-bubble {
  max-width: 75%;
  padding: 12px 16px;
  border-radius: 16px;
  background: #f5f5f5;
  color: #333333;
  font-size: 15px;
  line-height: 1.6;
  white-space: pre-wrap;
  word-break: break-word;
}

.user-bubble__text {
  margin: 0;
}

.message-item--assistant {
  display: block;
}

.assistant-stack {
  display: flex;
  flex-direction: column;
  gap: 16px;
  min-width: 0;
}

.thinking-panel {
  border: 1px solid #bfdbfe;
  background: #dbeafe;
  border-radius: 10px;
  overflow: hidden;
}

.thinking-panel__header {
  display: flex;
  align-items: center;
  gap: 8px;
  width: 100%;
  padding: 12px 16px;
  background: transparent;
  border: 0;
  cursor: pointer;
  text-align: left;
  transition: background 0.15s ease;
}

.thinking-panel__header:hover {
  background: rgba(191, 219, 254, 0.3);
}

.thinking-panel__icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  border-radius: 8px;
  background: #bfdbfe;
  color: #2563eb;
}

.thinking-panel__title {
  font-size: 14px;
  font-weight: 500;
  color: #2563eb;
}

.thinking-panel__duration {
  font-size: 12px;
  color: #2563eb;
  background: #bfdbfe;
  padding: 2px 8px;
  border-radius: 999px;
  margin-left: 4px;
}

.thinking-panel__caret {
  margin-left: auto;
  color: #3b82f6;
  font-size: 14px;
  transition: transform 0.2s ease;
}

.thinking-panel__caret--open {
  transform: rotate(180deg);
}

.thinking-panel__body {
  border-top: 1px solid #bfdbfe;
  padding: 12px 16px 16px;
}

.thinking-panel__content {
  font-size: 14px;
  line-height: 1.6;
  white-space: pre-wrap;
  word-break: break-word;
  color: #1e40af;
}

.assistant-body {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.ai-wait {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 4px 0;
}

.ai-wait__dot {
  display: inline-block;
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: #3b82f6;
  animation: wait-bounce 1.2s infinite ease-in-out;
}

.ai-wait__dot:nth-child(2) {
  animation-delay: 0.15s;
}

.ai-wait__dot:nth-child(3) {
  animation-delay: 0.3s;
}

@keyframes wait-bounce {
  0%, 80%, 100% {
    transform: scale(0.6);
    opacity: 0.5;
  }
  40% {
    transform: scale(1);
    opacity: 1;
  }
}

.message-notice {
  font-size: 12px;
  margin: 0;
}

.message-notice--error {
  color: #f43f5e;
}

.message-notice--cancelled {
  color: #999999;
}
</style>
