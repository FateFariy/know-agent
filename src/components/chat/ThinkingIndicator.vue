<script lang="ts" setup>
/**
 * ThinkingIndicator：深度思考中动画。
 * 复刻 React ThinkingIndicator 的浅蓝色面板与脑图标。
 */
import {Loading} from '@element-plus/icons-vue'

defineProps<{
  content?: string
  duration?: number
}>()
</script>

<template>
  <div class="thinking-indicator">
    <div class="thinking-indicator__header">
      <el-icon class="thinking-indicator__spinner">
        <Loading/>
      </el-icon>
      <span class="thinking-indicator__title">正在深度思考...</span>
      <span v-if="duration" class="thinking-indicator__duration">{{ duration }}秒</span>
    </div>
    <div class="thinking-indicator__body">
      <svg
        class="thinking-indicator__brain"
        fill="none"
        height="16"
        stroke="currentColor"
        stroke-linecap="round"
        stroke-linejoin="round"
        stroke-width="2"
        viewBox="0 0 24 24"
        width="16"
      >
        <path d="M9 6a3 3 0 0 1 3-3v0a3 3 0 0 1 3 3v0a3 3 0 0 1-3 3"/>
        <path d="M9 18a3 3 0 0 0 3 3v0a3 3 0 0 0 3-3v0a3 3 0 0 0-3-3"/>
        <path d="M3 12h3"/>
        <path d="M18 12h3"/>
        <path d="M5.6 5.6l2.1 2.1"/>
        <path d="M16.3 16.3l2.1 2.1"/>
        <path d="M5.6 18.4l2.1-2.1"/>
        <path d="M16.3 7.7l2.1-2.1"/>
      </svg>
      <p class="thinking-indicator__text">
        {{ content || '' }}<span class="thinking-indicator__cursor"/>
      </p>
    </div>
  </div>
</template>

<style scoped>
.thinking-indicator {
  border: 1px solid #bfdbfe;
  background: #dbeafe;
  border-radius: 12px;
  padding: 16px;
}

.thinking-indicator__header {
  display: flex;
  align-items: center;
  gap: 8px;
  color: #2563eb;
}

.thinking-indicator__spinner {
  font-size: 16px;
  animation: spin 1s linear infinite;
}

.thinking-indicator__title {
  font-size: 14px;
  font-weight: 500;
}

.thinking-indicator__duration {
  font-size: 12px;
  color: #2563eb;
  background: #bfdbfe;
  padding: 2px 8px;
  border-radius: 999px;
}

.thinking-indicator__body {
  margin-top: 12px;
  display: flex;
  align-items: flex-start;
  gap: 8px;
  color: #1e40af;
}

.thinking-indicator__brain {
  margin-top: 2px;
  font-size: 16px;
  color: #2563eb;
  flex-shrink: 0;
}

.thinking-indicator__text {
  font-size: 14px;
  line-height: 1.6;
  white-space: pre-wrap;
  word-break: break-word;
  flex: 1;
  min-width: 0;
}

.thinking-indicator__cursor {
  display: inline-block;
  width: 6px;
  height: 16px;
  margin-left: 4px;
  background: #3b82f6;
  vertical-align: middle;
  animation: blink 1s infinite;
}

@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}

@keyframes blink {
  0%,
  100% {
    opacity: 0.3;
  }
  50% {
    opacity: 1;
  }
}
</style>
