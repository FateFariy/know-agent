<script lang="ts" setup>
/**
 * ChatInput：对话输入区。
 * - 多行 textarea + 自动撑高；
 * - 深度思考开关；
 * - 发送/停止按钮；
 * - Enter 发送，Shift+Enter 换行。
 * 复刻 React ChatInput 的视觉与交互。
 */
import {computed, onBeforeUnmount, ref, watch} from 'vue'
import {MagicStick, Sunny} from '@element-plus/icons-vue'
import {useChatStore} from '@/stores/chat'

const store = useChatStore()

const value = ref('')
const isFocused = ref(false)
const isComposing = ref(false)
const textareaRef = ref<HTMLTextAreaElement | null>(null)

const placeholder = computed(() =>
  store.deepThinkingEnabled ? '输入需要深度分析的问题...' : '输入你的问题...'
)
const hasContent = computed(() => value.value.trim().length > 0)

function focusInput() {
  const el = textareaRef.value
  if (!el) return
  el.focus({preventScroll: true})
}

function adjustHeight() {
  const el = textareaRef.value
  if (!el) return
  el.style.height = 'auto'
  const next = Math.min(el.scrollHeight, 160)
  el.style.height = `${next}px`
}

watch(value, () => {
  adjustHeight()
})

watch(
  () => store.inputFocusKey,
  () => {
    focusInput()
  }
)

function onCompositionStart() {
  isComposing.value = true
}

function onCompositionEnd() {
  isComposing.value = false
}

async function handleSubmit() {
  if (store.isStreaming) {
    store.cancelGeneration()
    focusInput()
    return
  }
  if (!value.value.trim()) return
  const next = value.value
  value.value = ''
  focusInput()
  await store.sendMessage(next)
  focusInput()
}

function onKeyDown(e: KeyboardEvent) {
  if (e.key === 'Enter' && !e.shiftKey) {
    if (e.isComposing || isComposing.value || e.keyCode === 229) return
    e.preventDefault()
    handleSubmit()
  }
}

function toggleDeepThinking() {
  store.setDeepThinkingEnabled(!store.deepThinkingEnabled)
}

onBeforeUnmount(() => {
  textareaRef.value = null
})
</script>

<template>
  <div class="chat-input">
    <div
      :class="{ 'chat-input__card--focused': isFocused }"
      class="chat-input__card"
    >
      <div class="chat-input__field">
        <textarea
          ref="textareaRef"
          v-model="value"
          :placeholder="placeholder"
          aria-label="聊天输入框"
          class="chat-input__textarea"
          rows="1"
          @blur="isFocused = false"
          @compositionend="onCompositionEnd"
          @compositionstart="onCompositionStart"
          @focus="isFocused = true"
          @keydown="onKeyDown"
        />
        <div aria-hidden="true" class="chat-input__fade"/>
      </div>
      <div class="chat-input__actions">
        <button
          :aria-pressed="store.deepThinkingEnabled"
          :class="{ 'chat-input__chip--active': store.deepThinkingEnabled }"
          :disabled="store.isStreaming"
          class="chat-input__chip"
          type="button"
          @click="toggleDeepThinking"
        >
          <el-icon class="chat-input__chip-icon">
            <MagicStick/>
          </el-icon>
          深度思考
          <span
            v-if="store.deepThinkingEnabled"
            class="chat-input__chip-dot"
          />
        </button>
        <button
          :aria-label="store.isStreaming ? '停止生成' : '发送消息'"
          :class="{
            'chat-input__submit--streaming': store.isStreaming,
            'chat-input__submit--active': hasContent
          }"
          :disabled="!hasContent && !store.isStreaming"
          class="chat-input__submit"
          type="button"
          @click="handleSubmit"
        >
          <svg
            v-if="store.isStreaming"
            fill="currentColor"
            height="16"
            viewBox="0 0 16 16"
            width="16"
          >
            <rect height="10" rx="1.5" width="10" x="3" y="3"/>
          </svg>
          <svg
            v-else
            fill="none"
            height="16"
            stroke="currentColor"
            stroke-linecap="round"
            stroke-linejoin="round"
            stroke-width="2"
            viewBox="0 0 24 24"
            width="16"
          >
            <path d="M5 12h14"/>
            <path d="m13 6 6 6-6 6"/>
          </svg>
        </button>
      </div>
    </div>

    <p v-if="store.deepThinkingEnabled" class="chat-input__hint chat-input__hint--accent">
      <el-icon>
        <Sunny/>
      </el-icon>
      深度思考模式已开启，AI将进行更深入的分析推理
    </p>
    <p class="chat-input__hint">
      <kbd>Enter</kbd> 发送
      <span class="chat-input__hint-sep">·</span>
      <kbd>Shift + Enter</kbd> 换行
      <span v-if="store.isStreaming" class="chat-input__hint-pulse">生成中...</span>
    </p>
  </div>
</template>

<style scoped>
.chat-input {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.chat-input__card {
  display: flex;
  flex-direction: column;
  border-radius: 16px;
  border: 1px solid #e5e5e5;
  background: #ffffff;
  padding: 12px 16px 8px;
  transition: border-color 0.2s ease, box-shadow 0.2s ease;
}

.chat-input__card--focused {
  border-color: #d4d4d4;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.06);
}

.chat-input__card:hover {
  border-color: #d4d4d4;
}

.chat-input__field {
  position: relative;
}

.chat-input__textarea {
  display: block;
  width: 100%;
  min-height: 44px;
  max-height: 160px;
  padding: 8px 8px 12px;
  border: 0;
  background: transparent;
  font-size: 15px;
  line-height: 1.6;
  color: #333333;
  resize: none;
  outline: none;
}

.chat-input__textarea::placeholder {
  color: #999999;
}

.chat-input__fade {
  position: absolute;
  left: 0;
  right: 0;
  bottom: 0;
  height: 10px;
  background: linear-gradient(180deg, rgba(255, 255, 255, 0), rgba(255, 255, 255, 0.4) 40%, rgba(255, 255, 255, 0.9));
  pointer-events: none;
}

.chat-input__actions {
  position: relative;
  margin-top: 8px;
  display: flex;
  align-items: center;
}

.chat-input__chip {
  position: absolute;
  left: 0;
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 6px 12px;
  border: 1px solid transparent;
  background: #f5f5f5;
  color: #999999;
  border-radius: 8px;
  font-size: 12px;
  font-weight: 500;
  cursor: pointer;
  transition: background 0.15s ease, color 0.15s ease, border-color 0.15s ease;
}

.chat-input__chip:hover {
  background: #eeeeee;
}

.chat-input__chip--active {
  background: #dbeafe;
  border-color: #bfdbfe;
  color: #2563eb;
}

.chat-input__chip--active:hover {
  background: #dbeafe;
}

.chat-input__chip:disabled {
  cursor: not-allowed;
  opacity: 0.6;
}

.chat-input__chip-icon {
  font-size: 14px;
}

.chat-input__chip--active .chat-input__chip-icon {
  color: #3b82f6;
}

.chat-input__chip-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: #3b82f6;
  animation: pulse 1.2s ease-in-out infinite;
}

.chat-input__submit {
  margin-left: auto;
  width: 40px;
  height: 40px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border: 0;
  border-radius: 999px;
  background: #f5f5f5;
  color: #cccccc;
  cursor: pointer;
  transition: background 0.15s ease, color 0.15s ease, transform 0.15s ease;
}

.chat-input__submit--active {
  background: #3b82f6;
  color: #ffffff;
}

.chat-input__submit--active:hover {
  background: #2563eb;
}

.chat-input__submit--streaming {
  background: #fee2e2;
  color: #ef4444;
}

.chat-input__submit--streaming:hover {
  background: #fecaca;
}

.chat-input__submit:disabled {
  cursor: not-allowed;
}

.chat-input__hint {
  margin: 0;
  text-align: center;
  font-size: 12px;
  color: #999999;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  flex-wrap: wrap;
}

.chat-input__hint kbd {
  padding: 1px 6px;
  border-radius: 4px;
  background: #f5f5f5;
  color: #666666;
  font-size: 11px;
  box-shadow: 0 1px 0 rgba(15, 23, 42, 0.04);
}

.chat-input__hint--accent {
  color: #2563eb;
}

.chat-input__hint-sep {
  padding: 0 4px;
}

.chat-input__hint-pulse {
  margin-left: 8px;
  color: #3b82f6;
  animation: pulse 1.6s ease-in-out infinite;
}

@keyframes pulse {
  0%, 100% {
    opacity: 0.45;
  }
  50% {
    opacity: 1;
  }
}
</style>
