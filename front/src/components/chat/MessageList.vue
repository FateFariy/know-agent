<script lang="ts" setup>
/**
 * MessageList：消息列表 + 自动滚动到底部。
 * 不使用 Virtuoso 也能保持简洁：监听消息变化与流式状态，平滑滚动到底部。
 * 同时拦截三连击，避免选择跨越多条消息。
 */
import {nextTick, onBeforeUnmount, ref, watch} from 'vue'
import MessageItem from './MessageItem.vue'
import WelcomeScreen from './WelcomeScreen.vue'
import type {ChatMessage} from '@/stores/chat'
import type {SearchReference} from '@/types'

const props = defineProps<{
  messages: ChatMessage[]
  isLoading: boolean
  isStreaming: boolean
  sessionKey?: string | null
}>()

const scrollerRef = ref<HTMLElement | null>(null)
const lastSessionRef = ref<string | null>(null)
const pendingScrollRef = ref(true)

const emit = defineEmits<{
  (e: 'reference-select', ref: SearchReference): void
}>()

function scrollToBottom() {
  const scroller = scrollerRef.value
  if (!scroller) return
  scroller.scrollTop = scroller.scrollHeight
}

watch(
  () => props.sessionKey,
  (next) => {
    const nextKey = next ?? 'empty'
    if (lastSessionRef.value !== nextKey) {
      lastSessionRef.value = nextKey
      pendingScrollRef.value = true
    }
  }
)

watch(
  [() => props.messages.length, () => props.isStreaming, () => props.isLoading, () => props.sessionKey],
  async () => {
    await nextTick()
    if (props.isStreaming || props.isLoading) {
      scrollToBottom()
      return
    }
    if (pendingScrollRef.value) {
      scrollToBottom()
      window.setTimeout(scrollToBottom, 240)
      pendingScrollRef.value = false
    } else {
      scrollToBottom()
    }
  },
  {flush: 'post'}
)

function handleMouseDown(e: MouseEvent) {
  if (e.detail < 3) return
  e.preventDefault()
  const target = e.target as HTMLElement | null
  const root = scrollerRef.value
  if (!target || !root) return
  const block = target.closest('p, li, h1, h2, h3, h4, h5, h6, pre, blockquote, td, th') as HTMLElement | null
  const container = block && root.contains(block) ? block : root
  const sel = window.getSelection()
  if (sel) {
    const range = document.createRange()
    range.selectNodeContents(container)
    sel.removeAllRanges()
    sel.addRange(range)
  }
}

onBeforeUnmount(() => {
  scrollerRef.value = null
})
</script>

<template>
  <div v-if="messages.length === 0">
    <div v-if="isLoading" class="messages-list__loading"/>
    <WelcomeScreen v-else/>
  </div>
  <div
    v-else
    :key="sessionKey ?? 'empty'"
    ref="scrollerRef"
    class="messages-list"
    @mousedown="handleMouseDown"
  >
    <div class="messages-list__inner">
      <div
        v-for="(message, index) in messages"
        :key="message.id"
        :class="{ 'messages-list__row--last': index === messages.length - 1 }"
        class="messages-list__row"
      >
        <MessageItem :is-last="index === messages.length - 1" :message="message" @reference-select="(ref) => emit('reference-select', ref)"/>
      </div>
      <div aria-hidden="true" class="messages-list__footer"/>
    </div>
  </div>
</template>

<style scoped>
.messages-list {
  height: 100%;
  overflow-y: auto;
  background: #ffffff;
}

.messages-list__loading {
  height: 100%;
}

.messages-list__inner {
  max-width: 800px;
  margin: 0 auto;
  padding: 40px 24px 8px;
  display: flex;
  flex-direction: column;
  gap: 40px;
}

.messages-list__row {
  width: 100%;
}

.messages-list__row--last {
  animation: fade-up 0.4s ease both;
}

.messages-list__footer {
  height: 32px;
}

@keyframes fade-up {
  from {
    opacity: 0;
    transform: translateY(8px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

@media (max-width: 768px) {
  .messages-list__inner {
    padding: 24px 16px 8px;
    gap: 28px;
  }
}
</style>
