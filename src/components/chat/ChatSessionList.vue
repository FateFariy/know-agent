<script setup lang="ts">
import { computed } from 'vue'
import { useChatStore } from '@/stores/chat'
import { ChatDotRound, Delete } from '@element-plus/icons-vue'

const emit = defineEmits<{
  (e: 'select', conversationId: string): void
  (e: 'delete', conversationId: string): void
}>()

const store = useChatStore()

const sessions = computed(() => store.sessions)
const currentConversationId = computed(() => store.currentSession?.conversationId)

function selectSession(conversationId: string) {
  emit('select', conversationId)
}

function deleteSession(conversationId: string) {
  emit('delete', conversationId)
}

function formatTime(timeStr: string): string {
  if (!timeStr) return ''
  const date = new Date(timeStr)
  const now = new Date()
  const diff = now.getTime() - date.getTime()

  if (diff < 60000) return '刚刚'
  if (diff < 3600000) return `${Math.floor(diff / 60000)}分钟前`
  if (diff < 86400000) return `${Math.floor(diff / 3600000)}小时前`
  if (diff < 604800000) return `${Math.floor(diff / 86400000)}天前`

  return date.toLocaleDateString('zh-CN')
}
</script>

<template>
  <div class="session-list">
    <div class="session-list-header">
      <MessageSquare class="header-icon" />
      <span class="header-title">会话列表</span>
    </div>

    <div v-if="sessions.length === 0" class="empty-state">
      <ChatDotRound class="empty-icon" />
      <p>暂无会话</p>
      <p class="empty-hint">开始您的第一次问答吧</p>
    </div>

    <ul v-else class="session-items">
      <li v-for="session in sessions" :key="session.conversationId" class="session-item"
        :class="{ active: currentConversationId === session.conversationId }"
        @click="selectSession(session.conversationId)">
        <div class="session-content">
          <div class="session-preview">
            <span class="session-question">{{ session.latestUserMessage || '新会话' }}</span>
            <span class="session-answer">{{ session.latestAssistantMessage || '等待回复...' }}</span>
          </div>
          <div class="session-meta">
            <span class="session-time">{{ formatTime(session.updatedTime) }}</span>
            <span class="session-count">{{ session.messageCount }}条消息</span>
          </div>
        </div>
        <button class="delete-btn" @click.stop="deleteSession(session.conversationId)" title="删除会话">
          <Delete :size="16" />
        </button>
      </li>
    </ul>
  </div>
</template>

<style scoped>
.session-list {
  display: flex;
  flex-direction: column;
  height: 100%;
}

.session-list-header {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 16px;
  border-bottom: 1px solid #e2e8f0;
  background: #fff;
}

.header-icon {
  width: 20px;
  height: 20px;
  color: #3b82f6;
}

.header-title {
  font-size: 16px;
  font-weight: 600;
  color: #1e293b;
}

.empty-state {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 40px;
  color: #94a3b8;
}

.empty-icon {
  width: 48px;
  height: 48px;
  margin-bottom: 16px;
  opacity: 0.5;
}

.empty-hint {
  font-size: 14px;
  margin-top: 8px;
}

.session-items {
  flex: 1;
  list-style: none;
  padding: 0;
  margin: 0;
  overflow-y: auto;
}

.session-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 16px;
  border-bottom: 1px solid #f1f5f9;
  cursor: pointer;
  transition: background 0.2s;
}

.session-item:hover {
  background: #f8fafc;
}

.session-item.active {
  background: #eff6ff;
  border-left: 3px solid #3b82f6;
}

.session-content {
  flex: 1;
  min-width: 0;
}

.session-preview {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.session-question {
  font-size: 14px;
  font-weight: 500;
  color: #1e293b;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.session-answer {
  font-size: 13px;
  color: #64748b;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.session-meta {
  display: flex;
  gap: 12px;
  margin-top: 8px;
}

.session-time,
.session-count {
  font-size: 12px;
  color: #94a3b8;
}

.delete-btn {
  display: none;
  background: transparent;
  border: none;
  color: #94a3b8;
  cursor: pointer;
  padding: 6px;
  border-radius: 4px;
  transition: all 0.2s;
}

.delete-btn:hover {
  background: #fee2e2;
  color: #dc2626;
}

.session-item:hover .delete-btn {
  display: block;
}

@media (max-width: 768px) {
  .session-item {
    padding: 10px 12px;
  }

  .session-meta {
    flex-direction: column;
    gap: 4px;
  }
}
</style>
