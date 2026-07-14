<script setup lang="ts">
import { computed } from 'vue'
import type { Exchange } from '@/types'
import { User, Cpu, Warning, Document } from '@element-plus/icons-vue'

const props = defineProps<{
  exchange: Exchange
}>()



const statusText = computed(() => {
  switch (props.exchange.status) {
    case 0: return '完成'
    case 1: return '处理中'
    case -1: return '失败'
    default: return ''
  }
})

const statusClass = computed(() => {
  switch (props.exchange.status) {
    case 0: return 'status-success'
    case 1: return 'status-processing'
    case -1: return 'status-error'
    default: return ''
  }
})

function formatTime(timeStr: string): string {
  if (!timeStr) return ''
  const date = new Date(timeStr)
  return date.toLocaleString('zh-CN', {
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  })
}
</script>

<template>
  <div class="chat-message">
    <div class="message-user">
      <div class="avatar user-avatar">
        <User :size="20" />
      </div>
      <div class="message-content">
        <div class="message-header">
          <span class="sender">用户</span>
          <span class="time">{{ formatTime(exchange.createTime) }}</span>
        </div>
        <div class="message-body user-body">
          {{ exchange.question }}
        </div>
      </div>
    </div>

    <div class="message-assistant">
      <div class="avatar assistant-avatar">
        <Cpu :size="20" />
      </div>
      <div class="message-content">
        <div class="message-header">
          <span class="sender">智能助手</span>
          <span :class="['status', statusClass]">{{ statusText }}</span>
        </div>
        <div v-if="exchange.status === -1" class="error-message">
          <Warning :size="16" />
          <span>{{ exchange.errorMessage || '回答失败，请重试' }}</span>
        </div>
        <div v-else class="message-body assistant-body">
          <div v-if="exchange.thinkingSteps?.length" class="thinking-section">
            <h4 class="section-title">思考步骤</h4>
            <ul class="thinking-list">
              <li v-for="(step, index) in exchange.thinkingSteps" :key="index">
                {{ step }}
              </li>
            </ul>
          </div>

          <div class="answer-content" v-html="exchange.answer.replace(/\n/g, '<br/>')"></div>

          <div v-if="exchange.references?.length" class="references-section">
            <h4 class="section-title">参考来源</h4>
            <div class="references-list">
              <div v-for="ref in exchange.references" :key="ref.referenceId" class="reference-item">
                <Document :size="14" />
                <div class="reference-info">
                  <span class="reference-title">{{ ref.title }}</span>
                  <span class="reference-snippet">{{ ref.snippet }}</span>
                </div>
              </div>
            </div>
          </div>

          <div v-if="exchange.recommendations?.length" class="recommendations-section">
            <h4 class="section-title">相关推荐</h4>
            <ul class="recommendations-list">
              <li v-for="(rec, index) in exchange.recommendations" :key="index">
                {{ rec }}
              </li>
            </ul>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.chat-message {
  margin-bottom: 24px;
}

.message-user,
.message-assistant {
  display: flex;
  gap: 12px;
  margin-bottom: 16px;
}

.avatar {
  width: 40px;
  height: 40px;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.user-avatar {
  background: linear-gradient(135deg, #6366f1, #8b5cf6);
  color: #fff;
}

.assistant-avatar {
  background: linear-gradient(135deg, #3b82f6, #06b6d4);
  color: #fff;
}

.message-content {
  flex: 1;
  min-width: 0;
}

.message-header {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 8px;
}

.sender {
  font-size: 14px;
  font-weight: 600;
  color: #475569;
}

.time {
  font-size: 12px;
  color: #94a3b8;
}

.status {
  font-size: 12px;
  padding: 2px 8px;
  border-radius: 10px;
}

.status-success {
  background: #dcfce7;
  color: #16a34a;
}

.status-processing {
  background: #fef3c7;
  color: #d97706;
}

.status-error {
  background: #fee2e2;
  color: #dc2626;
}

.message-body {
  padding: 12px 16px;
  border-radius: 12px;
  line-height: 1.6;
}

.user-body {
  background: #3b82f6;
  color: #fff;
  border-bottom-right-radius: 4px;
}

.assistant-body {
  background: #fff;
  border: 1px solid #e2e8f0;
  border-bottom-left-radius: 4px;
}

.error-message {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 12px 16px;
  background: #fee2e2;
  border-radius: 8px;
  color: #dc2626;
}

.thinking-section,
.references-section,
.recommendations-section {
  margin-top: 16px;
  padding-top: 16px;
  border-top: 1px solid #f1f5f9;
}

.section-title {
  font-size: 14px;
  font-weight: 600;
  color: #475569;
  margin: 0 0 12px 0;
}

.thinking-list,
.recommendations-list {
  list-style: none;
  padding: 0;
  margin: 0;
}

.thinking-list li,
.recommendations-list li {
  font-size: 13px;
  color: #64748b;
  padding: 4px 0;
  padding-left: 20px;
  position: relative;
}

.thinking-list li::before,
.recommendations-list li::before {
  content: '•';
  position: absolute;
  left: 0;
  color: #3b82f6;
}

.references-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.reference-item {
  display: flex;
  gap: 8px;
  padding: 10px 12px;
  background: #f8fafc;
  border-radius: 8px;
}

.reference-info {
  flex: 1;
  min-width: 0;
}

.reference-title {
  font-size: 13px;
  font-weight: 500;
  color: #3b82f6;
  display: block;
  margin-bottom: 4px;
}

.reference-snippet {
  font-size: 12px;
  color: #64748b;
  display: block;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.answer-content {
  font-size: 14px;
  color: #334155;
}

.answer-content :deep(p) {
  margin-bottom: 8px;
}

.answer-content :deep(strong) {
  font-weight: 600;
}

.answer-content :deep(ul),
.answer-content :deep(ol) {
  padding-left: 24px;
  margin: 8px 0;
}

.answer-content :deep(li) {
  margin-bottom: 4px;
}

@media (max-width: 768px) {

  .message-user,
  .message-assistant {
    gap: 8px;
  }

  .avatar {
    width: 32px;
    height: 32px;
  }

  .message-body {
    padding: 10px 12px;
    font-size: 13px;
  }
}
</style>
