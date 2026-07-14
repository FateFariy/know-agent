<script setup lang="ts">
import { ref, computed, onMounted, watch, nextTick } from 'vue'
import { useRouter } from 'vue-router'
import { useChatStore } from '@/stores/chat'
import { ElMessage } from 'element-plus'
import { ArrowRight, Paperclip, Close, Cpu, Star } from '@element-plus/icons-vue'
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

onMounted(() => {
  fetchData()
})
</script>

<template>
  <div class="chat-view">
    <aside class="chat-sidebar" :class="{ hidden: !showSessionList }">
      <ChatSessionList @select="handleSelectSession" @delete="handleDeleteSession" />
    </aside>

    <main class="chat-main">
      <div v-if="!store.currentSession" class="welcome-screen">
        <div class="welcome-content">
          <div class="welcome-icon">
            <Cpu :size="64" />
          </div>
          <h2>欢迎使用智能知识库</h2>
          <p>选择一个文档，开始您的问答之旅</p>

          <div class="document-selector">
            <label class="selector-label">选择文档</label>
            <div class="document-options">
              <button class="doc-option" :class="{ selected: selectedDocumentId === null }"
                @click="selectDocument(null)">
                <Star :size="18" />
                <span>全部文档</span>
              </button>
              <button v-for="doc in documentOptions" :key="doc.documentId" class="doc-option"
                :class="{ selected: selectedDocumentId === doc.documentId }" @click="selectDocument(doc.documentId)">
                <Paperclip :size="18" />
                <span>{{ doc.documentName }}</span>
              </button>
            </div>
          </div>

          <button class="upload-btn" @click="goToDocuments">
            <Paperclip :size="18" />
            <span>上传新文档</span>
          </button>
        </div>
      </div>

      <div v-else class="chat-container">
        <header class="chat-header">
          <button class="toggle-sidebar-btn" @click="showSessionList = !showSessionList">
            <Close v-if="showSessionList" :size="20" />
            <span v-else>会话</span>
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
          <div class="footer-hint">
            <span>按 Enter 发送，Shift + Enter 换行</span>
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
}

.chat-sidebar {
  width: 320px;
  border-right: 1px solid #e2e8f0;
  background: linear-gradient(180deg, #fafbfc 0%, #f1f5f9 100%);
  transition: width 0.3s ease;
  display: flex;
  flex-direction: column;
}

.chat-sidebar.hidden {
  width: 0;
  overflow: hidden;
}

.chat-main {
  flex: 1;
  display: flex;
  flex-direction: column;
  background: linear-gradient(180deg, #f0f4f8 0%, #f8fafc 100%);
}

.welcome-screen {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 40px;
}

.welcome-content {
  text-align: center;
  max-width: 500px;
  width: 100%;
}

.welcome-icon {
  width: 140px;
  height: 140px;
  margin: 0 auto 32px;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #fff;
  box-shadow: 0 20px 60px rgba(102, 126, 234, 0.3);
  animation: iconFloat 3s ease-in-out infinite;
}

@keyframes iconFloat {
  0%, 100% {
    transform: translateY(0);
  }
  50% {
    transform: translateY(-10px);
  }
}

.welcome-content h2 {
  font-size: 28px;
  font-weight: 700;
  color: #0f172a;
  margin-bottom: 16px;
  background: linear-gradient(135deg, #0f172a, #334155);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  background-clip: text;
}

.welcome-content p {
  font-size: 15px;
  color: #64748b;
  margin-bottom: 40px;
  line-height: 1.6;
}

.document-selector {
  background: #fff;
  border-radius: 16px;
  padding: 24px;
  margin-bottom: 24px;
  box-shadow: 0 4px 24px rgba(0, 0, 0, 0.06);
  border: 1px solid #f1f5f9;
}

.selector-label {
  display: block;
  font-size: 15px;
  font-weight: 600;
  color: #334155;
  margin-bottom: 16px;
  text-align: left;
}

.document-options {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
}

.doc-option {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 16px;
  background: #f8fafc;
  border: 2px solid transparent;
  border-radius: 10px;
  font-size: 13px;
  color: #475569;
  cursor: pointer;
  transition: all 0.25s ease;
}

.doc-option:hover {
  background: #eff6ff;
  border-color: #93c5fd;
  transform: translateY(-1px);
}

.doc-option.selected {
  background: linear-gradient(135deg, #3b82f6, #2563eb);
  border-color: #3b82f6;
  color: #fff;
  box-shadow: 0 4px 12px rgba(59, 130, 246, 0.3);
}

.upload-btn {
  display: inline-flex;
  align-items: center;
  gap: 10px;
  padding: 14px 32px;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  border: none;
  border-radius: 12px;
  font-size: 15px;
  font-weight: 600;
  color: #fff;
  cursor: pointer;
  transition: all 0.25s ease;
  box-shadow: 0 4px 16px rgba(102, 126, 234, 0.4);
}

.upload-btn:hover {
  transform: translateY(-2px);
  box-shadow: 0 6px 20px rgba(102, 126, 234, 0.5);
}

.chat-container {
  flex: 1;
  display: flex;
  flex-direction: column;
}

.chat-header {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 14px 20px;
  background: #fff;
  border-bottom: 1px solid #e2e8f0;
  box-shadow: 0 1px 4px rgba(0, 0, 0, 0.04);
}

.toggle-sidebar-btn {
  display: none;
  background: #f1f5f9;
  border: none;
  color: #475569;
  cursor: pointer;
  padding: 8px 12px;
  border-radius: 8px;
  font-size: 14px;
  transition: all 0.2s;
}

.toggle-sidebar-btn:hover {
  background: #e2e8f0;
}

.chat-title {
  display: flex;
  align-items: center;
  gap: 10px;
  font-size: 17px;
  font-weight: 600;
  color: #0f172a;
}

.document-selector-header {
  margin-left: auto;
}

.doc-select {
  padding: 8px 16px;
  border: 2px solid #e2e8f0;
  border-radius: 8px;
  font-size: 13px;
  color: #475569;
  background: #fff;
  cursor: pointer;
  transition: all 0.2s;
  outline: none;
}

.doc-select:focus {
  border-color: #3b82f6;
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
  color: #94a3b8;
  padding: 40px;
}

.empty-messages .empty-hint {
  font-size: 13px;
  margin-top: 8px;
}

.loading-indicator {
  display: flex;
  align-items: center;
  gap: 14px;
  padding: 20px;
  background: #fff;
  border-radius: 16px;
  border: 1px solid #e2e8f0;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.04);
  align-self: flex-start;
  max-width: 80%;
}

.loading-dots {
  display: flex;
  gap: 8px;
}

.loading-dots span {
  width: 10px;
  height: 10px;
  background: linear-gradient(135deg, #667eea, #764ba2);
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
  border-top: 1px solid #e2e8f0;
  box-shadow: 0 -1px 4px rgba(0, 0, 0, 0.04);
}

.input-container {
  display: flex;
  gap: 14px;
  align-items: flex-end;
}

.message-input {
  flex: 1;
  padding: 14px 18px;
  border: 2px solid #e2e8f0;
  border-radius: 16px;
  font-size: 15px;
  line-height: 1.5;
  resize: none;
  outline: none;
  transition: all 0.25s ease;
  min-height: 48px;
  max-height: 180px;
}

.message-input:focus {
  border-color: #667eea;
  box-shadow: 0 0 0 3px rgba(102, 126, 234, 0.1);
}

.message-input::placeholder {
  color: #94a3b8;
}

.message-input:disabled {
  background: #f8fafc;
  cursor: not-allowed;
  opacity: 0.7;
}

.send-btn {
  align-self: flex-end;
  padding: 14px 24px;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  border: none;
  border-radius: 14px;
  color: #fff;
  cursor: pointer;
  transition: all 0.25s ease;
  box-shadow: 0 4px 14px rgba(102, 126, 234, 0.4);
}

.send-btn:hover:not(:disabled) {
  transform: translateY(-2px);
  box-shadow: 0 6px 18px rgba(102, 126, 234, 0.5);
}

.send-btn:active:not(:disabled) {
  transform: translateY(0);
}

.send-btn:disabled {
  background: #cbd5e1;
  cursor: not-allowed;
  box-shadow: none;
}

.footer-hint {
  margin-top: 10px;
  text-align: center;
}

.footer-hint span {
  font-size: 12px;
  color: #94a3b8;
}

@media (max-width: 768px) {
  .chat-sidebar {
    position: fixed;
    left: 0;
    top: 0;
    bottom: 0;
    z-index: 100;
    width: 280px;
    transform: translateX(-100%);
    transition: transform 0.3s ease;
    box-shadow: 4px 0 20px rgba(0, 0, 0, 0.1);
  }

  .chat-sidebar.hidden {
    transform: translateX(-100%);
  }

  .chat-sidebar:not(.hidden) {
    transform: translateX(0);
  }

  .toggle-sidebar-btn {
    display: block;
  }

  .document-selector-header {
    display: none;
  }

  .chat-title {
    font-size: 15px;
  }

  .messages-container {
    padding: 16px;
    gap: 16px;
  }

  .chat-footer {
    padding: 12px 16px;
  }

  .welcome-screen {
    padding: 20px;
  }

  .welcome-icon {
    width: 100px;
    height: 100px;
    margin-bottom: 20px;
  }

  .welcome-content h2 {
    font-size: 22px;
  }
}

@media (max-width: 480px) {
  .welcome-content {
    padding: 16px;
  }

  .welcome-icon {
    width: 80px;
    height: 80px;
    margin-bottom: 16px;
  }

  .welcome-content h2 {
    font-size: 20px;
  }

  .document-options {
    flex-direction: column;
  }

  .doc-option {
    justify-content: center;
  }

  .upload-btn {
    width: 100%;
    justify-content: center;
  }

  .messages-container {
    padding: 12px;
  }

  .chat-header {
    padding: 12px 14px;
  }

  .chat-footer {
    padding: 10px 12px;
  }

  .input-container {
    gap: 10px;
  }

  .send-btn {
    padding: 12px 18px;
  }
}
</style>
