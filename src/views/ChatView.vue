<script setup lang="ts">
import { ref, computed, onMounted, watch, nextTick } from 'vue'
import { useRouter } from 'vue-router'
import { useChatStore } from '@/stores/chat'
import { ElMessage } from 'element-plus'
import { ArrowRight, Cpu, Search, Plus, Lightning, Aim, DataBoard } from '@element-plus/icons-vue'
import ChatSessionList from '@/components/chat/ChatSessionList.vue'
import ChatMessage from '@/components/chat/ChatMessage.vue'

const router = useRouter()
const store = useChatStore()

const messageInput = ref('')
const isSending = ref(false)
const showSessionList = ref(true)
const messagesContainer = ref<HTMLElement | null>(null)

const exchanges = computed(() => store.exchanges)
const isLoading = computed(() => store.isLoading)
const documentOptions = computed(() => store.documentOptions)
const selectedDocumentId = computed(() => store.selectedDocumentId)

const quickQuestions = [
  { title: '系统交互', desc: '关于系统', icon: Lightning },
  { title: '实时数据', desc: '数据统计与监控', icon: Aim },
  { title: '业务系统', desc: '信息查询、流程审批、安全', icon: DataBoard },
]

function scrollToBottom() {
  nextTick(() => {
    if (messagesContainer.value) {
      messagesContainer.value.scrollTop = messagesContainer.value.scrollHeight
    }
  })
}

watch(exchanges, () => {
  scrollToBottom()
}, { deep: true })

async function fetchData() {
  await store.fetchSessions()
  await store.fetchDocumentOptions()
}

async function handleSelectSession(conversationId: string) {
  await store.fetchSessionDetail(conversationId)
}

async function handleDeleteSession(conversationId: string) {
  await store.deleteSession(conversationId)
  if (store.currentSession?.conversationId === conversationId) {
    store.currentSession = null
    store.exchanges = []
  }
  ElMessage.success('会话已删除')
}

async function handleSend() {
  const question = messageInput.value.trim()
  if (!question) {
    ElMessage.warning('请输入问题')
    return
  }

  isSending.value = true
  messageInput.value = ''

  try {
    let conversationId = store.currentSession?.conversationId

    if (!conversationId) {
      const res = await store.sendMessage('', question, selectedDocumentId.value || undefined)
      conversationId = res.data?.conversationId || ''
    } else {
      await store.sendMessage(conversationId, question, selectedDocumentId.value || undefined)
    }

    if (conversationId) {
      await store.fetchSessionDetail(conversationId)
    }
    await store.fetchSessions()
  } catch {
    ElMessage.error('发送失败，请重试')
  } finally {
    isSending.value = false
  }
}

function handleKeydown(e: KeyboardEvent) {
  if (e.key === 'Enter' && !e.shiftKey) {
    e.preventDefault()
    handleSend()
  }
}

function selectDocument(documentId: number | null) {
  store.selectDocument(documentId)
}

function goToDocuments() {
  router.push('/admin')
}

function handleQuickQuestion(title: string) {
  messageInput.value = title
}

onMounted(() => {
  fetchData()
})
</script>

<template>
  <div class="chat-view">
    <aside class="chat-sidebar" :class="{ hidden: !showSessionList }">
      <div class="sidebar-header">
        <button class="new-chat-btn" @click="store.currentSession = null; store.exchanges = []">
          <Plus :size="18" />
          <span>新对话</span>
        </button>
        <button class="sidebar-close" @click="showSessionList = false">
          <span class="close-icon">×</span>
        </button>
      </div>

      <div class="search-box">
        <Search :size="16" />
        <input type="text" placeholder="搜索对话" class="search-input" />
      </div>

      <div class="sidebar-nav">
        <button class="nav-item active">
          <FolderOpen :size="16" />
          <span>最近对话</span>
        </button>
      </div>

      <div class="session-list">
        <ChatSessionList @select="handleSelectSession" @delete="handleDeleteSession" />
      </div>

      <div class="sidebar-footer">
        <button class="admin-btn" @click="goToDocuments">
          <Cpu :size="16" />
          <span>知识库管理</span>
        </button>
      </div>
    </aside>

    <div class="sidebar-overlay" v-show="!showSessionList" @click="showSessionList = true"></div>

    <main class="chat-main">
      <div v-if="!store.currentSession" class="welcome-screen">
        <div class="welcome-header">
          <button class="menu-btn" @click="showSessionList = !showSessionList">
            <span class="menu-icon">☰</span>
          </button>
          <h1 class="page-title">新对话</h1>
          <button class="admin-header-btn" @click="goToDocuments">
            <Cpu :size="18" />
          </button>
        </div>

        <div class="hero-section">
          <div class="hero-content">
            <h2 class="hero-title">
              把问题变成<span class="accent">清晰答案</span>
            </h2>
            <p class="hero-subtitle">
              结构化提问、知识检索与深度思考，一次对话出可执行方案
            </p>
          </div>

          <div class="search-container">
            <div class="search-wrapper">
              <div class="document-selector">
                <select :value="selectedDocumentId || ''"
                  @change="selectDocument(($event.target as HTMLSelectElement).value ? Number(($event.target as HTMLSelectElement).value) : null)"
                  class="doc-select">
                  <option value="">全部文档</option>
                  <option v-for="doc in documentOptions" :key="doc.documentId" :value="doc.documentId">
                    {{ doc.documentName }}
                  </option>
                </select>
              </div>
              <textarea v-model="messageInput" placeholder="输入你的问题..." class="hero-input" rows="2"
                @keydown="handleKeydown"></textarea>
              <button class="send-btn" @click="handleSend" :disabled="!messageInput.trim()">
                <ArrowRight :size="18" />
              </button>
            </div>
            <div class="footer-hint">
              <span>按 Enter 发送，Shift + Enter 换行</span>
            </div>
          </div>

          <div class="quick-cards">
            <div v-for="item in quickQuestions" :key="item.title" class="quick-card"
              @click="handleQuickQuestion(item.title)">
              <div class="card-icon">
                <component :is="item.icon" :size="20" />
              </div>
              <div class="card-content">
                <h4>{{ item.title }}</h4>
                <p>{{ item.desc }}</p>
              </div>
              <ArrowRight :size="16" class="card-arrow" />
            </div>
          </div>
        </div>
      </div>

      <div v-else class="chat-container">
        <header class="chat-header">
          <button class="toggle-sidebar-btn" @click="showSessionList = !showSessionList">
            <span class="menu-icon">☰</span>
          </button>
          <div class="chat-title">
            <Cpu :size="18" />
            <span>{{ store.currentSession.selectedDocumentName || '智能问答' }}</span>
          </div>
          <div class="document-selector-header">
            <select :value="selectedDocumentId || ''"
              @change="selectDocument(($event.target as HTMLSelectElement).value ? Number(($event.target as HTMLSelectElement).value) : null)"
              class="doc-select">
              <option value="">全部文档</option>
              <option v-for="doc in documentOptions" :key="doc.documentId" :value="doc.documentId">
                {{ doc.documentName }}
              </option>
            </select>
          </div>
        </header>

        <div ref="messagesContainer" class="messages-container">
          <div v-if="exchanges.length === 0" class="empty-messages">
            <Cpu :size="48" />
            <p>开始提问吧</p>
            <p class="empty-hint">基于您选择的文档获取智能回答</p>
          </div>

          <ChatMessage v-for="exchange in exchanges" :key="exchange.exchangeId" :exchange="exchange" />

          <div v-if="isLoading || isSending" class="loading-indicator">
            <div class="loading-dots">
              <span></span>
              <span></span>
              <span></span>
            </div>
            <span class="loading-text">正在思考中...</span>
          </div>
        </div>

        <footer class="chat-footer">
          <div class="input-container">
            <textarea v-model="messageInput" placeholder="输入您的问题..." class="message-input" rows="2"
              @keydown="handleKeydown" :disabled="isSending"></textarea>
            <button class="send-btn" @click="handleSend" :disabled="!messageInput.trim() || isSending">
              <ArrowRight :size="18" />
            </button>
          </div>
        </footer>
      </div>
    </main>
  </div>
</template>

<style scoped>
.chat-view {
  display: flex;
  height: 100%;
  background: #f5f7fa;
}

.chat-sidebar {
  width: 280px;
  background: #fff;
  border-right: 1px solid #e8ecf0;
  display: flex;
  flex-direction: column;
  position: fixed;
  left: 0;
  top: 0;
  bottom: 0;
  z-index: 100;
  transition: transform 0.3s ease;
  box-shadow: 2px 0 12px rgba(0, 0, 0, 0.06);
}

.chat-sidebar.hidden {
  transform: translateX(-100%);
}

.sidebar-overlay {
  display: none;
}

.sidebar-header {
  display: flex;
  gap: 8px;
  padding: 12px;
  border-bottom: 1px solid #f0f2f5;
}

.new-chat-btn {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  padding: 10px 12px;
  background: #3b82f6;
  color: #fff;
  border: none;
  border-radius: 8px;
  font-size: 14px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s;
}

.new-chat-btn:hover {
  background: #2563eb;
}

.sidebar-close {
  display: none;
  background: transparent;
  border: none;
  color: #909399;
  cursor: pointer;
  padding: 8px;
  border-radius: 4px;
}

.search-box {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 12px;
  background: #f5f7fa;
  margin: 10px 12px;
  border-radius: 8px;
}

.search-input {
  flex: 1;
  border: none;
  background: transparent;
  font-size: 13px;
  color: #303133;
  outline: none;
}

.search-input::placeholder {
  color: #c0c4cc;
}

.sidebar-nav {
  padding: 0 12px;
}

.nav-item {
  display: flex;
  align-items: center;
  gap: 8px;
  width: 100%;
  padding: 8px 12px;
  background: transparent;
  border: none;
  border-radius: 6px;
  font-size: 13px;
  color: #606266;
  cursor: pointer;
  transition: all 0.2s;
}

.nav-item:hover {
  background: #f5f7fa;
}

.nav-item.active {
  background: #eff6ff;
  color: #3b82f6;
}

.session-list {
  flex: 1;
  overflow-y: auto;
  padding: 8px 0;
}

.sidebar-footer {
  padding: 12px;
  border-top: 1px solid #f0f2f5;
}

.admin-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  width: 100%;
  padding: 10px 12px;
  background: #f5f7fa;
  border: 1px solid #e4e7ed;
  border-radius: 8px;
  font-size: 13px;
  color: #606266;
  cursor: pointer;
  transition: all 0.2s;
}

.admin-btn:hover {
  background: #eff6ff;
  border-color: #c7d2fe;
  color: #3b82f6;
}

.chat-main {
  flex: 1;
  display: flex;
  flex-direction: column;
  margin-left: 280px;
  transition: margin-left 0.3s ease;
}

.chat-sidebar.hidden+.chat-main {
  margin-left: 0;
}

.welcome-screen {
  flex: 1;
  display: flex;
  flex-direction: column;
  background: linear-gradient(180deg, #ffffff 0%, #f8fafc 100%);
}

.welcome-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 16px 24px;
  background: rgba(255, 255, 255, 0.8);
  backdrop-filter: blur(12px);
}

.menu-btn {
  background: transparent;
  border: none;
  color: #606266;
  cursor: pointer;
  padding: 8px;
  border-radius: 6px;
  font-size: 16px;
}

.menu-btn:hover {
  background: #f5f7fa;
}

.page-title {
  font-size: 16px;
  font-weight: 600;
  color: #303133;
  margin: 0;
}

.admin-header-btn {
  display: none;
  background: #f5f7fa;
  border: 1px solid #e4e7ed;
  border-radius: 8px;
  padding: 8px;
  color: #606266;
  cursor: pointer;
  transition: all 0.2s;
}

.admin-header-btn:hover {
  background: #eff6ff;
  border-color: #c7d2fe;
  color: #3b82f6;
}

.hero-section {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 40px 24px;
  max-width: 800px;
  margin: 0 auto;
  width: 100%;
}

.hero-content {
  text-align: center;
  margin-bottom: 40px;
}

.hero-title {
  font-size: 36px;
  font-weight: 700;
  color: #1a1a1a;
  margin: 0 0 16px 0;
  letter-spacing: -0.02em;
}

.accent {
  color: #3b82f6;
}

.hero-subtitle {
  font-size: 15px;
  color: #64748b;
  margin: 0;
  line-height: 1.6;
}

.search-container {
  width: 100%;
  margin-bottom: 48px;
}

.search-wrapper {
  display: flex;
  align-items: flex-end;
  gap: 12px;
  background: #fff;
  border-radius: 16px;
  padding: 12px;
  box-shadow: 0 4px 24px rgba(0, 0, 0, 0.08);
  border: 1px solid #e8ecf0;
  transition: all 0.25s ease;
}

.search-wrapper:focus-within {
  box-shadow: 0 8px 32px rgba(59, 130, 246, 0.15);
  border-color: #3b82f6;
}

.document-selector {
  flex-shrink: 0;
}

.doc-select {
  padding: 10px 14px;
  border: 1px solid #e4e7ed;
  border-radius: 10px;
  font-size: 13px;
  color: #606266;
  background: #fafafa;
  cursor: pointer;
  outline: none;
  transition: all 0.2s;
}

.doc-select:focus {
  border-color: #3b82f6;
  background: #fff;
}

.hero-input {
  flex: 1;
  border: none;
  outline: none;
  font-size: 15px;
  line-height: 1.5;
  resize: none;
  min-height: 40px;
  max-height: 120px;
  background: transparent;
}

.hero-input::placeholder {
  color: #c0c4cc;
}

.send-btn {
  flex-shrink: 0;
  padding: 12px 18px;
  background: #3b82f6;
  border: none;
  border-radius: 12px;
  color: #fff;
  cursor: pointer;
  transition: all 0.25s ease;
}

.send-btn:hover:not(:disabled) {
  background: #2563eb;
  transform: translateY(-1px);
}

.send-btn:disabled {
  background: #d9d9d9;
  cursor: not-allowed;
}

.footer-hint {
  margin-top: 12px;
  text-align: center;
}

.footer-hint span {
  font-size: 12px;
  color: #909399;
}

.quick-cards {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 16px;
  width: 100%;
}

.quick-card {
  display: flex;
  flex-direction: column;
  gap: 12px;
  padding: 20px;
  background: #fff;
  border-radius: 14px;
  border: 1px solid #f0f2f5;
  cursor: pointer;
  transition: all 0.25s ease;
}

.quick-card:hover {
  border-color: #3b82f6;
  box-shadow: 0 8px 24px rgba(59, 130, 246, 0.1);
  transform: translateY(-2px);
}

.card-icon {
  width: 44px;
  height: 44px;
  background: linear-gradient(135deg, #eff6ff, #dbeafe);
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #3b82f6;
}

.card-content h4 {
  font-size: 15px;
  font-weight: 600;
  color: #303133;
  margin: 0 0 4px 0;
}

.card-content p {
  font-size: 13px;
  color: #909399;
  margin: 0;
  line-height: 1.4;
}

.card-arrow {
  align-self: flex-end;
  color: #c0c4cc;
  transition: all 0.2s;
}

.quick-card:hover .card-arrow {
  color: #3b82f6;
  transform: translateX(4px);
}

.chat-container {
  flex: 1;
  display: flex;
  flex-direction: column;
  background: #f8fafc;
}

.chat-header {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 14px 20px;
  background: #fff;
  border-bottom: 1px solid #e8ecf0;
}

.toggle-sidebar-btn {
  background: transparent;
  border: none;
  color: #606266;
  cursor: pointer;
  padding: 8px;
  border-radius: 6px;
  font-size: 16px;
}

.toggle-sidebar-btn:hover {
  background: #f5f7fa;
}

.chat-title {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 16px;
  font-weight: 600;
  color: #303133;
}

.document-selector-header {
  margin-left: auto;
}

.messages-container {
  flex: 1;
  padding: 24px;
  overflow-y: auto;
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.empty-messages {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  height: 100%;
  color: #909399;
}

.empty-messages .empty-hint {
  font-size: 13px;
  margin-top: 8px;
}

.loading-indicator {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 16px 20px;
  background: #fff;
  border-radius: 14px;
  border: 1px solid #e8ecf0;
  align-self: flex-start;
}

.loading-dots {
  display: flex;
  gap: 6px;
}

.loading-dots span {
  width: 8px;
  height: 8px;
  background: #3b82f6;
  border-radius: 50%;
  animation: bounce 1.4s infinite ease-in-out both;
}

.loading-dots span:nth-child(1) {
  animation-delay: -0.32s;
}

.loading-dots span:nth-child(2) {
  animation-delay: -0.16s;
}

@keyframes bounce {

  0%,
  80%,
  100% {
    transform: scale(0);
    opacity: 0.5;
  }

  40% {
    transform: scale(1);
    opacity: 1;
  }
}

.loading-text {
  font-size: 14px;
  color: #64748b;
}

.chat-footer {
  padding: 16px 24px;
  background: #fff;
  border-top: 1px solid #e8ecf0;
}

.input-container {
  display: flex;
  gap: 12px;
  align-items: flex-end;
}

.message-input {
  flex: 1;
  padding: 12px 16px;
  border: 1px solid #e4e7ed;
  border-radius: 14px;
  font-size: 14px;
  line-height: 1.5;
  resize: none;
  outline: none;
  transition: all 0.25s ease;
  min-height: 44px;
  max-height: 160px;
}

.message-input:focus {
  border-color: #3b82f6;
}

.message-input:disabled {
  background: #fafafa;
  cursor: not-allowed;
}

@media (max-width: 768px) {
  .chat-sidebar {
    width: 260px;
    transform: translateX(-100%);
    box-shadow: 4px 0 20px rgba(0, 0, 0, 0.15);
  }

  .chat-sidebar:not(.hidden) {
    transform: translateX(0);
  }

  .sidebar-close {
    display: block;
  }

  .sidebar-overlay {
    display: block;
    position: fixed;
    top: 0;
    left: 0;
    right: 0;
    bottom: 0;
    background: rgba(0, 0, 0, 0.5);
    z-index: 99;
  }

  .chat-main {
    margin-left: 0;
  }

  .welcome-header {
    padding: 12px 16px;
  }

  .admin-header-btn {
    display: block;
  }

  .hero-section {
    padding: 24px 16px;
  }

  .hero-title {
    font-size: 24px;
  }

  .search-wrapper {
    flex-direction: column;
    align-items: stretch;
  }

  .document-selector {
    width: 100%;
  }

  .doc-select {
    width: 100%;
  }

  .quick-cards {
    grid-template-columns: 1fr;
  }

  .quick-card {
    flex-direction: row;
    align-items: center;
  }

  .card-icon {
    flex-shrink: 0;
  }

  .card-content {
    flex: 1;
  }

  .messages-container {
    padding: 12px;
  }

  .chat-header {
    padding: 12px 14px;
  }

  .chat-footer {
    padding: 12px;
  }
}

@media (max-width: 480px) {
  .hero-title {
    font-size: 20px;
  }

  .hero-subtitle {
    font-size: 13px;
  }

  .quick-card {
    padding: 16px;
  }

  .card-icon {
    width: 36px;
    height: 36px;
  }
}
</style>
