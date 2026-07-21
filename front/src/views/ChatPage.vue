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
import { ElDrawer } from 'element-plus'
import ChatHeader from '@/components/chat/ChatHeader.vue'
import ChatInput from '@/components/chat/ChatInput.vue'
import ChatSidebar from '@/components/chat/ChatSidebar.vue'
import MessageList from '@/components/chat/MessageList.vue'
import { useChatStore } from '@/stores/chat'
import type { SearchReference } from '@/types'

const store = useChatStore()
const route = useRoute()
const router = useRouter()

const sidebarOpen = ref(false)
const sessionsReady = ref(false)
const drawerVisible = ref(false)
const selectedReference = ref<SearchReference | null>(null)

function handleReferenceSelect(ref: SearchReference) {
  selectedReference.value = ref
  drawerVisible.value = true
}

function closeDrawer() {
  drawerVisible.value = false
  selectedReference.value = null
}

const sessionId = computed(() => (route.params.sessionId as string | undefined) || null)
// 会话存在性判断：store.sessions 尚未加载完成时不进行判断，避免刷新瞬间 sessions 仍为空时
// 误判为"会话不存在"而触发 createSession 覆盖掉 sessionStorage 恢复的 currentSessionId
const sessionExists = computed(() => {
  if (!sessionId.value) return false
  if (!store.sessionsLoaded) return true
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
      if (store.isCreatingNew) return
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
            @reference-select="handleReferenceSelect"
          />
        </div>
        <div v-if="!showWelcome" class="chat-page__input">
          <div class="chat-page__input-inner">
            <ChatInput />
          </div>
        </div>
      </div>

      <ElDrawer v-model="drawerVisible" title="引用详情" direction="rtl" size="480px" :before-close="closeDrawer"
        class="reference-drawer" :append-to-body="false">
        <div v-if="selectedReference" class="reference-detail">
          <div class="reference-detail__header">
            <h3 class="reference-detail__title">{{ selectedReference.title || selectedReference.documentName || '参考文档' }}
            </h3>
            <span v-if="selectedReference.score != null" class="reference-detail__score">相似度 {{ (selectedReference.score *
              100).toFixed(0) }}%</span>
          </div>
          <div v-if="selectedReference.sectionPath" class="reference-detail__section">
            <span class="reference-detail__label">章节路径</span>
            <span class="reference-detail__value">{{ selectedReference.sectionPath }}</span>
          </div>
          <div v-if="selectedReference.documentName" class="reference-detail__section">
            <span class="reference-detail__label">文档名称</span>
            <span class="reference-detail__value">{{ selectedReference.documentName }}</span>
          </div>
          <div v-if="selectedReference.sourceType" class="reference-detail__section">
            <span class="reference-detail__label">来源类型</span>
            <span class="reference-detail__value">{{ selectedReference.sourceType }}</span>
          </div>
          <div v-if="selectedReference.snippet" class="reference-detail__snippet">
            <span class="reference-detail__label">引用片段</span>
            <p class="reference-detail__text">{{ selectedReference.snippet }}</p>
          </div>
          <div v-if="selectedReference.url" class="reference-detail__link">
            <a :href="selectedReference.url" target="_blank" rel="noopener">查看原文</a>
          </div>
        </div>
      </ElDrawer>
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

<style>
.reference-detail {
  padding: 8px 0;
}

.reference-detail__header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  margin-bottom: 16px;
}

.reference-detail__title {
  font-size: 16px;
  font-weight: 600;
  color: #1a1a1a;
  line-height: 1.4;
  margin: 0;
  flex: 1;
  margin-right: 12px;
}

.reference-detail__score {
  padding: 2px 8px;
  border-radius: 4px;
  background: #f0fdf4;
  color: #16a34a;
  font-size: 12px;
  font-weight: 500;
  white-space: nowrap;
}

.reference-detail__section {
  display: flex;
  align-items: center;
  padding: 10px 0;
  border-bottom: 1px solid #f1f5f9;
}

.reference-detail__label {
  width: 80px;
  font-size: 13px;
  color: #64748b;
  flex-shrink: 0;
}

.reference-detail__value {
  font-size: 14px;
  color: #334155;
  flex: 1;
}

.reference-detail__snippet {
  padding: 12px 0;
  border-bottom: 1px solid #f1f5f9;
}

.reference-detail__snippet .reference-detail__label {
  display: block;
  margin-bottom: 8px;
  width: auto;
}

.reference-detail__text {
  margin: 0;
  padding: 12px;
  border-radius: 8px;
  background: #f8fafc;
  color: #334155;
  font-size: 14px;
  line-height: 1.7;
  white-space: pre-wrap;
  word-break: break-word;
}

.reference-detail__link {
  margin-top: 12px;
}

.reference-detail__link a {
  display: inline-flex;
  align-items: center;
  color: #2563eb;
  font-size: 14px;
  text-decoration: none;
  transition: color 0.15s ease;
}

.reference-detail__link a:hover {
  text-decoration: underline;
}
</style>
