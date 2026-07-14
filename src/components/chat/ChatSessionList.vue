<script setup lang="ts">
import { computed } from 'vue'
import { useChatStore } from '@/stores/chat'
import { ChatDotRound } from '@element-plus/icons-vue'

const emit = defineEmits<{
  (e: 'select', conversationId: string): void
}>()

const store = useChatStore()

const sessions = computed(() => store.sessions)
const currentConversationId = computed(() => store.currentSession?.conversationId)

function selectSession(conversationId: string) {
  emit('select', conversationId)
}
</script>

<template>
  <div class="session-list-inner">
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
          <span class="session-question">{{ session.latestUserMessage || '新会话' }}</span>
        </div>
      </li>
    </ul>
  </div>
</template>

<style scoped>
.session-list-inner {
  display: flex;
  flex-direction: column;
  height: 100%;
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
  padding: 10px 12px;
  cursor: pointer;
  transition: background 0.2s;
  border-radius: 8px;
  margin: 2px 4px;
}

.session-item:hover {
  background: #f1f5f9;
}

.session-item.active {
  background: #eff6ff;
}

.session-content {
  min-width: 0;
}

.session-question {
  font-size: 14px;
  color: #475569;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.session-item.active .session-question {
  color: #3b82f6;
  font-weight: 500;
}

@media (max-width: 768px) {
  .session-item {
    padding: 8px 10px;
  }
}
</style>
