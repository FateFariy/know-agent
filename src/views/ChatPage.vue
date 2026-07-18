<script setup lang="ts">
/**
 * ChatPage：会话主页面。
 * - 自动加载会话列表；
 * - URL 带 sessionId 时选中该会话；
 * - 首次进入无会话时创建新会话；
 * - 切换会话自动同步 URL。
 * 布局：ChatSidebar + ChatHeader + MessageList/ChatInput。
 */
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import ChatHeader from '@/components/chat/ChatHeader.vue'
import ChatInput from '@/components/chat/ChatInput.vue'
import ChatSidebar from '@/components/chat/ChatSidebar.vue'
import MessageList from '@/components/chat/MessageList.vue'
import { useChatStore } from '@/stores/chat'

const store = useChatStore()
const route = useRoute()
const router = useRouter()

const sidebarOpen = ref(false)
const sessionsReady = ref(false)

const sessionId = computed(() => (route.params.sessionId as string | undefined) || null)
const sessionExists = computed(() => {
  if (!sessionId.value) return false
  return store.sessions.some((s) => s.id === sessionId.value)
})
const showWelcome = computed(() => store.showWelcome)

let active = true
store.fetchSessions()
  .catch(() => null)
  .finally(() => {
    if (active) sessionsReady.value = true
  })

watch(
  [sessionId, sessionsReady, () => store.sessions, () => store.isCreatingNew, () => store.currentSessionId],
  async () => {
    if (!sessionsReady.value) return

    if (sessionId.value) {
      if (!sessionExists.value && !store.isCreatingNew) {
        await store.createSession().catch(() => null)
        router.replace('/chat').catch(() => null)
        return
      }
      await store.selectSession(sessionId.value).catch(() => null)
      return
    }

    if (store.isCreatingNew) return
    if (store.currentSessionId) {
      router.replace(`/chat/${store.currentSessionId}`).catch(() => null)
      return
    }
    await store.createSession().catch(() => null)
  },
  { immediate: true }
)

watch(
  () => store.currentSessionId,
  (current) => {
    if (current && current !== sessionId.value) {
      router.replace(`/chat/${current}`).catch(() => null)
    }
  }
)

function handleToggleSidebar() {
  sidebarOpen.value = !sidebarOpen.value
}

function handleCloseSidebar() {
  sidebarOpen.value = false
}

onBeforeUnmount(() => {
  active = false
})
</script>

<template>
  <div class="chat-page">
    <ChatSidebar :is-open="sidebarOpen" @close="handleCloseSidebar" />
    <div class="chat-page__main">
      <ChatHeader :show-menu-button="true" @toggle-sidebar="handleToggleSidebar" />
      <div class="chat-page__content">
        <div class="chat-page__messages">
          <MessageList
            :messages="store.messages"
            :is-loading="store.isLoading"
            :is-streaming="store.isStreaming"
            :session-key="store.currentSessionId"
          />
        </div>
        <div v-if="!showWelcome" class="chat-page__input">
          <div class="chat-page__input-inner">
            <ChatInput />
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.chat-page {
  display: flex;
  min-height: 100vh;
  background: #fafafa;
}

.chat-page__main {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  background: #ffffff;
}

.chat-page__content {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
}

.chat-page__messages {
  flex: 1;
  min-height: 0;
}

.chat-page__input {
  position: relative;
  z-index: 20;
  background: #ffffff;
  border-top: 1px solid transparent;
}

.chat-page__input-inner {
  max-width: 800px;
  margin: 0 auto;
  padding: 4px 24px 16px;
}

@media (max-width: 1023px) {
  .chat-page__main {
    width: 100%;
  }
}

@media (max-width: 640px) {
  .chat-page__input-inner {
    padding: 4px 16px 12px;
  }
}
</style>
