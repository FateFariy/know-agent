<script setup lang="ts">
import { ref } from 'vue'
import { ArrowDown, ArrowUp, User, Box, WarnTriangleFilled } from '@element-plus/icons-vue'
import type { Exchange } from '@/types'

defineProps<{
  exchange: Exchange
}>()

const showReferences = ref(false)
const showThinking = ref(false)

function formatTime(time: string) {
  if (!time) return ''
  return new Date(time).toLocaleString('zh-CN', {
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  })
}

function toggleReferences() {
  showReferences.value = !showReferences.value
}

function toggleThinking() {
  showThinking.value = !showThinking.value
}
</script>

<template>
  <div class="message-bubble">
    <div class="message-user">
      <div class="user-avatar user">
        <User :size="20" />
      </div>
      <div class="message-content">
        <div class="message-header">
          <span class="sender">用户</span>
          <span class="time">{{ formatTime(exchange.createTime) }}</span>
        </div>
        <div class="message-text user-text">{{ exchange.question }}</div>
      </div>
    </div>

    <div class="message-assistant">
      <div class="user-avatar assistant">
        <Box :size="20" />
      </div>
      <div class="message-content">
        <div class="message-header">
          <span class="sender">AI助手</span>
          <span class="time">{{ formatTime(exchange.updateTime) }}</span>
        </div>

        <div v-if="exchange.errorMessage" class="error-message">
          <WarnTriangleFilled :size="16" class="error-icon" />
          <span>{{ exchange.errorMessage }}</span>
        </div>

        <div v-else class="message-text assistant-text">
          {{ exchange.answer || '暂无回答' }}
        </div>

        <div v-if="exchange.thinkingSteps?.length > 0" class="expandable-section">
          <button class="expand-btn" @click="toggleThinking">
            <component :is="showThinking ? ArrowUp : ArrowDown" :size="16" />
            <span>思维步骤</span>
          </button>
          <div v-if="showThinking" class="expand-content">
            <div
              v-for="(step, index) in exchange.thinkingSteps"
              :key="index"
              class="thinking-step"
            >
              <span class="step-number">{{ index + 1 }}</span>
              <span>{{ step }}</span>
            </div>
          </div>
        </div>

        <div v-if="exchange.references?.length > 0" class="expandable-section">
          <button class="expand-btn" @click="toggleReferences">
            <component :is="showReferences ? ArrowUp : ArrowDown" :size="16" />
            <span>参考来源 ({{ exchange.references.length }})</span>
          </button>
          <div v-if="showReferences" class="expand-content references-list">
            <div
              v-for="ref in exchange.references"
              :key="ref.referenceId"
              class="reference-item"
            >
              <div class="reference-header">
                <span class="reference-title">{{ ref.title }}</span>
                <span class="reference-score">相似度: {{ (ref.score * 100).toFixed(1) }}%</span>
              </div>
              <div class="reference-snippet">{{ ref.snippet }}</div>
              <div class="reference-meta">
                <span>{{ ref.documentName }}</span>
                <span v-if="ref.sectionPath">{{ ref.sectionPath }}</span>
              </div>
            </div>
          </div>
        </div>

        <div v-if="exchange.recommendations?.length > 0" class="recommendations">
          <span class="rec-label">推荐问题:</span>
          <span
            v-for="(rec, index) in exchange.recommendations"
            :key="index"
            class="rec-item"
          >
            {{ rec }}
          </span>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.message-bubble {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.message-user,
.message-assistant {
  display: flex;
  gap: 12px;
}

.user-avatar {
  width: 40px;
  height: 40px;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.user-avatar.user {
  background: #e0e7ff;
  color: #4338ca;
}

.user-avatar.assistant {
  background: #dbeafe;
  color: #1d4ed8;
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
  font-size: 12px;
  font-weight: 600;
}

.message-user .sender {
  color: #4338ca;
}

.message-assistant .sender {
  color: #1d4ed8;
}

.time {
  font-size: 12px;
  color: #94a3b8;
}

.message-text {
  padding: 12px 16px;
  border-radius: 12px;
  font-size: 14px;
  line-height: 1.6;
  white-space: pre-wrap;
  word-break: break-word;
}

.user-text {
  background: #f3f4f6;
  color: #1f2937;
  border-top-left-radius: 4px;
}

.assistant-text {
  background: #eff6ff;
  color: #1e40af;
  border-top-right-radius: 4px;
}

.error-message {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 12px 16px;
  background: #fef2f2;
  color: #dc2626;
  border-radius: 12px;
  font-size: 14px;
}

.error-icon {
  flex-shrink: 0;
}

.expandable-section {
  margin-top: 12px;
}

.expand-btn {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 6px 12px;
  border: none;
  background: #f1f5f9;
  color: #64748b;
  border-radius: 6px;
  font-size: 12px;
  cursor: pointer;
  transition: all 0.2s ease;
}

.expand-btn:hover {
  background: #e2e8f0;
  color: #334155;
}

.expand-content {
  margin-top: 8px;
  padding: 12px;
  background: #f8fafc;
  border-radius: 8px;
}

.thinking-step {
  display: flex;
  gap: 8px;
  padding: 6px 0;
  font-size: 13px;
  color: #475569;
}

.step-number {
  width: 24px;
  height: 24px;
  background: #3b82f6;
  color: #fff;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 12px;
  flex-shrink: 0;
}

.references-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.reference-item {
  padding: 12px;
  background: #fff;
  border-radius: 8px;
  border: 1px solid #e2e8f0;
}

.reference-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 8px;
}

.reference-title {
  font-size: 14px;
  font-weight: 600;
  color: #1e40af;
}

.reference-score {
  font-size: 12px;
  color: #f59e0b;
}

.reference-snippet {
  font-size: 13px;
  color: #475569;
  margin-bottom: 8px;
  line-height: 1.5;
}

.reference-meta {
  display: flex;
  gap: 8px;
  font-size: 12px;
  color: #94a3b8;
}

.recommendations {
  margin-top: 12px;
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  align-items: center;
}

.rec-label {
  font-size: 12px;
  color: #64748b;
}

.rec-item {
  padding: 4px 12px;
  background: #fef3c7;
  color: #d97706;
  border-radius: 16px;
  font-size: 12px;
}
</style>
