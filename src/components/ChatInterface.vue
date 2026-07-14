<script setup lang="ts">
import { ref, watch, nextTick } from 'vue'
import { ElInput, ElButton, ElLoading, ElMessage } from 'element-plus'
import { ArrowRightBold, Document, Refresh } from '@element-plus/icons-vue'
import { useChatStore } from '@/stores/chat'
import MessageBubble from './MessageBubble.vue'
import type { DocumentOption } from '@/types'

const chatStore = useChatStore()
const messageInput = ref('')
const messagesContainer = ref<HTMLElement | null>(null)
const conversationId = ref('')
const selectedDocument = ref<DocumentOption | null>(null)

async function initSession() {
  conversationId.value = await chatStore.createSession(selectedDocument.value?.documentId ?? undefined)
  await chatStore.fetchSessionDetail(conversationId.value)
}

async function sendMessage() {
  if (!messageInput.value.trim() || !conversationId.value) return
  const question = messageInput.value.trim()
  messageInput.value = ''
  try {
    await chatStore.sendMessage(conversationId.value, question, selectedDocument.value?.documentId ?? undefined)
    scrollToBottom()
  } catch {
    ElMessage.error('发送失败，请重试')
  }
}

function handleKeydown(e: Event | KeyboardEvent) {
  if ((e as KeyboardEvent).key === 'Enter' && !(e as KeyboardEvent).shiftKey) {
    e.preventDefault()
    sendMessage()
  }
}

function scrollToBottom() {
  nextTick(() => {
    if (messagesContainer.value) {
      messagesContainer.value.scrollTop = messagesContainer.value.scrollHeight
    }
  })
}

watch(() => chatStore.exchanges, () => {
  scrollToBottom()
}, { deep: true })

watch(selectedDocument, async () => {
  conversationId.value = ''
  await chatStore.fetchDocumentOptions()
})

async function handleDocumentChange(doc: DocumentOption | null) {
  selectedDocument.value = doc
  await initSession()
}
</script>

<template>
  <div class="chat-interface">
    <div class="chat-header">
      <div class="header-left">
        <Document :size="20" class="header-icon" />
        <span class="header-title">智能问答</span>
      </div>
      <div class="header-right">
        <select
          v-if="chatStore.documentOptions.length > 0"
          :value="selectedDocument?.documentId ?? ''"
          @change="(e: Event) => {
            const target = e.target as HTMLSelectElement
            const doc = chatStore.documentOptions.find(d => d.documentId === Number(target.value))
            handleDocumentChange(doc || null)
          }"
          class="document-select"
        >
          <option value="">选择知识库文档</option>
          <option v-for="doc in chatStore.documentOptions" :key="doc.documentId" :value="doc.documentId">
            {{ doc.documentName }}
          </option>
        </select>
        <ElButton
          type="text"
          icon="Refresh"
          @click="initSession"
          :disabled="chatStore.isLoading"
        >
          <Refresh :size="18" />
        </ElButton>
      </div>
    </div>

    <div ref="messagesContainer" class="messages-container">
      <div v-if="!conversationId" class="empty-state">
        <div class="empty-icon">
          <Document :size="48" />
        </div>
        <p class="empty-title">开始智能问答</p>
        <p class="empty-desc">选择一个知识库文档，或直接输入问题开始对话</p>
        <ElButton type="primary" @click="initSession">开始对话</ElButton>
      </div>

      <template v-else-if="chatStore.exchanges.length === 0">
        <div class="empty-chat">
          <p>暂无对话记录，开始提问吧！</p>
        </div>
      </template>

      <div v-else class="messages-list">
        <MessageBubble
          v-for="exchange in chatStore.exchanges"
          :key="exchange.exchangeId"
          :exchange="exchange"
        />
      </div>

      <ElLoading v-if="chatStore.isLoading" class="loading-overlay">
        <span>正在思考中...</span>
      </ElLoading>
    </div>

    <div class="input-container">
      <ElInput
        v-model="messageInput"
        type="textarea"
        :rows="3"
        placeholder="输入您的问题..."
        resize="none"
        @keydown="handleKeydown"
        :disabled="!conversationId || chatStore.isLoading"
      />
      <ElButton
        type="primary"
        icon="Send"
        @click="sendMessage"
        :disabled="!messageInput.trim() || !conversationId || chatStore.isLoading"
        class="send-btn"
      >
        <ArrowRightBold :size="18" />
      </ElButton>
    </div>
  </div>
</template>

<style scoped>
.chat-interface {
  display: flex;
  flex-direction: column;
  height: 100%;
  background: #fff;
  border-radius: 12px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.08);
  overflow: hidden;
}

.chat-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px 24px;
  background: linear-gradient(90deg, #3b82f6 0%, #2563eb 100%);
  color: #fff;
}

.header-left {
  display: flex;
  align-items: center;
  gap: 10px;
}

.header-icon {
  opacity: 0.9;
}

.header-title {
  font-size: 16px;
  font-weight: 600;
}

.header-right {
  display: flex;
  align-items: center;
  gap: 12px;
}

.document-select {
  padding: 6px 12px;
  border: none;
  border-radius: 6px;
  background: rgba(255, 255, 255, 0.2);
  color: #fff;
  font-size: 14px;
  cursor: pointer;
}

.document-select option {
  background: #2563eb;
  color: #fff;
}

.messages-container {
  flex: 1;
  overflow-y: auto;
  padding: 24px;
  background: #f8fafc;
  position: relative;
}

.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  height: 100%;
  color: #64748b;
}

.empty-icon {
  margin-bottom: 16px;
  opacity: 0.5;
}

.empty-title {
  font-size: 18px;
  font-weight: 600;
  margin-bottom: 8px;
  color: #334155;
}

.empty-desc {
  font-size: 14px;
  margin-bottom: 24px;
}

.empty-chat {
  display: flex;
  justify-content: center;
  align-items: center;
  height: 100%;
  color: #94a3b8;
}

.messages-list {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.loading-overlay {
  position: absolute;
  bottom: 0;
  left: 0;
  right: 0;
  padding: 16px;
  background: rgba(248, 250, 252, 0.9);
}

.input-container {
  display: flex;
  gap: 12px;
  padding: 16px 24px;
  border-top: 1px solid #e2e8f0;
  background: #fff;
}

.input-container .el-input {
  flex: 1;
}

.send-btn {
  align-self: flex-end;
  padding: 8px 20px;
}
</style>
