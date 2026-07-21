<script lang="ts" setup>
/**
 * MessageItem：单条消息渲染。
 * - 用户消息：右对齐气泡；
 * - 助手消息：左侧展示，深度思考折叠面板 + 流式正文(thinking / text) + 引用 + 推荐问题 + 反馈按钮。
 * - 流式阶段：thinking 阶段在 assistant-body 顶部实时显示思考内容(text + 光标)；
 *             text 阶段切到 MarkdownRenderer；
 *             未到任何事件时显示 ai-wait dots 占位。
 */
import { computed, ref } from 'vue'
import { ArrowDown, Document, Loading, Promotion } from '@element-plus/icons-vue'
import FeedbackButtons from './FeedbackButtons.vue'
import MarkdownRenderer from './MarkdownRenderer.vue'
import { useChatStore } from '@/stores/chat'
import type { ChatMessage } from '@/stores/chat'
import type { SearchReference } from '@/types'

const props = defineProps<{
  message: ChatMessage
  isLast?: boolean
}>()

const store = useChatStore()
const thinkingExpanded = ref(false)

const emit = defineEmits<{
  (e: 'reference-select', ref: SearchReference): void
}>()
const isUser = computed(() => props.message.role === 'user')
const isThinking = computed(() => Boolean(props.message.isThinking))
const isStreaming = computed(() => props.message.status === 'streaming')
const hasThinking = computed(() => Boolean(props.message.thinking && props.message.thinking.trim().length > 0))
const hasContent = computed(() => props.message.content.trim().length > 0)
// 仅当 streaming 且既没收到 thinking 也没收到 text 时，才显示 dots 占位
const isWaiting = computed(() => isStreaming.value && !isThinking.value && !hasContent.value)
const isError = computed(() => props.message.status === 'error')
const isCancelled = computed(() => props.message.status === 'cancelled')
const hasReferences = computed(() => Boolean(props.message.references?.length))
const hasRecommendations = computed(() => Boolean(props.message.recommendations?.length))
const thinkingDuration = computed(() => props.message.thinkingDuration ? `${props.message.thinkingDuration}秒` : '')
const showFeedback = computed(() =>
  props.message.role === 'assistant' &&
  props.message.status !== 'streaming' &&
  Boolean(props.message.id) &&
  !props.message.id.startsWith('assistant-')
)

/** 点击推荐问题：直接发送 */
function handleRecommend(question: string) {
  if (store.isStreaming) return
  store.sendMessage(question)
}

function handleReferenceClick(index: number) {
  const refItem = props.message.references?.[index - 1]
  if (refItem) {
    emit('reference-select', refItem)
  }
}
</script>

<template>
  <div v-if="isUser" class="message-item message-item--user">
    <div class="user-bubble">
      <p class="user-bubble__text">{{ message.content }}</p>
    </div>
  </div>

  <div v-else class="message-item message-item--assistant">
    <div class="assistant-stack">
      <!-- 折叠面板：仅在 streaming 结束后展示，作为历史查看入口 -->
      <div v-if="!isThinking && hasThinking" class="thinking-panel">
        <button :aria-expanded="thinkingExpanded" class="thinking-panel__header" type="button"
          @click="thinkingExpanded = !thinkingExpanded">
          <span class="thinking-panel__icon">
            <svg fill="none" height="16" stroke="currentColor" stroke-linecap="round" stroke-linejoin="round"
              stroke-width="2" viewBox="0 0 24 24" width="16">
              <path d="M9 6a3 3 0 0 1 3-3v0a3 3 0 0 1 3 3v0a3 3 0 0 1-3 3" />
              <path d="M9 18a3 3 0 0 0 3 3v0a3 3 0 0 0 3-3v0a3 3 0 0 0-3-3" />
              <path d="M3 12h3" />
              <path d="M18 12h3" />
              <path d="M5.6 5.6l2.1 2.1" />
              <path d="M16.3 16.3l2.1 2.1" />
              <path d="M5.6 18.4l2.1-2.1" />
              <path d="M16.3 7.7l2.1-2.1" />
            </svg>
          </span>
          <span class="thinking-panel__title">深度思考</span>
          <span v-if="thinkingDuration" class="thinking-panel__duration">{{
            thinkingDuration
          }}</span>
          <el-icon :class="{ 'thinking-panel__caret--open': thinkingExpanded }" class="thinking-panel__caret">
            <ArrowDown />
          </el-icon>
        </button>
        <div v-if="thinkingExpanded" class="thinking-panel__body">
          <div class="thinking-panel__content">{{ message.thinking }}</div>
        </div>
      </div>

      <div class="assistant-body">
        <!-- 思考中：流式 thinking 预览（实时显示 message.thinking + 闪烁光标） -->
        <div v-if="isThinking" class="stream-preview stream-preview--thinking">
          <div class="stream-preview__header">
            <el-icon class="stream-preview__spinner">
              <Loading />
            </el-icon>
            <span class="stream-preview__label">正在思考…</span>
            <span v-if="thinkingDuration" class="stream-preview__duration">{{ thinkingDuration }}</span>
          </div>
          <p class="stream-preview__content">
            {{ message.thinking || '' }}<span class="stream-preview__cursor" />
          </p>
        </div>

        <!-- 等待首个事件：dots 占位 -->
        <div v-else-if="isWaiting" aria-label="思考中" class="ai-wait">
          <span class="ai-wait__dot" />
          <span class="ai-wait__dot" />
          <span class="ai-wait__dot" />
        </div>

        <!-- 正式回答：Markdown 流式 -->
        <MarkdownRenderer v-else-if="hasContent" :content="message.content" @reference-click="handleReferenceClick" />
        <p v-if="isError" class="message-notice message-notice--error">生成已中断。</p>
        <p v-else-if="isCancelled" class="message-notice message-notice--cancelled">（已停止生成）</p>

        <!-- 推荐问题（按钮列表，点击直接发送） -->
        <div v-if="hasRecommendations" class="recommend">
          <div class="recommend__header">
            <el-icon class="recommend__icon">
              <Promotion />
            </el-icon>
            <span>推荐问题</span>
          </div>
          <div class="recommend__list">
            <button v-for="(q, idx) in message.recommendations" :key="idx" class="recommend__btn" type="button"
              @click="handleRecommend(q)">{{ q }}</button>
          </div>
        </div>

        <FeedbackButtons v-if="showFeedback" :always-visible="Boolean(isLast)" :content="message.content"
          :feedback="message.feedback ?? null" :message-id="message.id" />
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

  0%,
  80%,
  100% {
    transform: scale(0.6);
    opacity: 0.5;
  }

  40% {
    transform: scale(1);
    opacity: 1;
  }
}

/* 流式 thinking 预览：直接铺在 assistant-body 顶部，与正文同位置实时更新 */
.stream-preview {
  border: 1px solid #bfdbfe;
  background: #f0f7ff;
  border-radius: 10px;
  padding: 12px 14px;
}

.stream-preview__header {
  display: flex;
  align-items: center;
  gap: 8px;
  color: #2563eb;
  margin-bottom: 8px;
}

.stream-preview__spinner {
  font-size: 14px;
  animation: stream-spin 1s linear infinite;
}

@keyframes stream-spin {
  to {
    transform: rotate(360deg);
  }
}

.stream-preview__label {
  font-size: 13px;
  font-weight: 600;
}

.stream-preview__duration {
  font-size: 11px;
  color: #2563eb;
  background: #dbeafe;
  padding: 1px 8px;
  border-radius: 999px;
  font-variant-numeric: tabular-nums;
}

.stream-preview__content {
  margin: 0;
  font-size: 14px;
  line-height: 1.65;
  color: #1e40af;
  white-space: pre-wrap;
  word-break: break-word;
}

.stream-preview__cursor {
  display: inline-block;
  width: 6px;
  height: 14px;
  margin-left: 2px;
  background: #3b82f6;
  vertical-align: -2px;
  animation: stream-blink 1s infinite;
}

@keyframes stream-blink {

  0%,
  100% {
    opacity: 0.2;
  }

  50% {
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

.references {
  margin-top: 8px;
  padding: 10px 12px;
  border: 1px solid #e2e8f0;
  border-radius: 8px;
  background: #f8fafc;
}

.references__header {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  font-weight: 600;
  color: #475569;
  margin-bottom: 6px;
}

.references__icon {
  color: #3b82f6;
  font-size: 14px;
}

.references__list {
  list-style: none;
  margin: 0;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.references__item {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 13px;
  line-height: 1.5;
  color: #334155;
}

.references__index {
  color: #94a3b8;
  font-variant-numeric: tabular-nums;
  flex-shrink: 0;
}

.references__title {
  color: #2563eb;
  text-decoration: none;
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

a.references__title:hover {
  text-decoration: underline;
}

.references__score {
  font-size: 11px;
  color: #94a3b8;
  background: #e2e8f0;
  padding: 1px 6px;
  border-radius: 8px;
  flex-shrink: 0;
}

.recommend {
  margin-top: 8px;
}

.recommend__header {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  font-weight: 600;
  color: #475569;
  margin-bottom: 6px;
}

.recommend__icon {
  color: #3b82f6;
  font-size: 14px;
}

.recommend__list {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.recommend__btn {
  padding: 6px 12px;
  border: 1px solid #cbd5e1;
  border-radius: 999px;
  background: #ffffff;
  color: #334155;
  font-size: 13px;
  line-height: 1.4;
  cursor: pointer;
  transition: all 0.15s ease;
  text-align: left;
  max-width: 100%;
}

.recommend__btn:hover {
  border-color: #3b82f6;
  color: #2563eb;
  background: #eff6ff;
}
</style>
