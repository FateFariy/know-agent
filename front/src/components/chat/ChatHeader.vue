<script lang="ts" setup>
/**
 * ChatHeader：会话头部。
 * - 移动端侧边栏开关 + 当前会话标题 + 工具按钮。
 */
import {computed} from 'vue'
import {ChatLineRound, Fold, Setting} from '@element-plus/icons-vue'
import {useChatStore} from '@/stores/chat'

defineProps<{
  showMenuButton?: boolean
}>()

const emit = defineEmits<{
  (e: 'toggleSidebar'): void
}>()

const store = useChatStore()

const title = computed(() => {
  const current = store.sessions.find((s) => s.id === store.currentSessionId)
  return current?.title || '新对话'
})
</script>

<template>
  <header class="chat-header">
    <div class="chat-header__left">
      <button
        v-if="showMenuButton"
        aria-label="切换侧边栏"
        class="chat-header__menu"
        type="button"
        @click="emit('toggleSidebar')"
      >
        <el-icon>
          <Fold/>
        </el-icon>
      </button>
      <div class="chat-header__title">
        <el-icon class="chat-header__title-icon">
          <ChatLineRound/>
        </el-icon>
        <span>{{ title }}</span>
      </div>
    </div>
    <div class="chat-header__right">
      <a
        aria-label="管理后台"
        class="chat-header__action"
        href="/admin"
        rel="noreferrer"
        target="_blank"
      >
        <el-icon>
          <Setting/>
        </el-icon>
        <span>管理后台</span>
      </a>
    </div>
  </header>
</template>

<style scoped>
.chat-header {
  position: sticky;
  top: 0;
  z-index: 20;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 16px 24px;
  background: rgba(255, 255, 255, 0.92);
  backdrop-filter: blur(12px);
  border-bottom: 1px solid #f0f0f0;
}

.chat-header__left {
  display: flex;
  align-items: center;
  gap: 12px;
}

.chat-header__menu {
  display: none;
  align-items: center;
  justify-content: center;
  width: 36px;
  height: 36px;
  border: 0;
  border-radius: 8px;
  background: transparent;
  color: #6b7280;
  cursor: pointer;
  transition: background 0.15s ease;
}

.chat-header__menu:hover {
  background: #f3f4f6;
}

@media (max-width: 1023px) {
  .chat-header__menu {
    display: inline-flex;
  }
}

.chat-header__title {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 16px;
  font-weight: 500;
  color: #1f2937;
}

.chat-header__title-icon {
  color: #3b82f6;
  font-size: 18px;
}

.chat-header__right {
  display: flex;
  align-items: center;
  gap: 8px;
}

.chat-header__action {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 6px 12px;
  border: 1px solid #e5e7eb;
  border-radius: 999px;
  background: #ffffff;
  color: #374151;
  font-size: 13px;
  text-decoration: none;
  transition: background 0.15s ease, border-color 0.15s ease;
}

.chat-header__action:hover {
  background: #f9fafb;
  border-color: #93c5fd;
  color: #2563eb;
}

@media (max-width: 640px) {
  .chat-header {
    padding: 12px 16px;
  }

  .chat-header__action span {
    display: none;
  }
}
</style>
