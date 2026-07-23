<script lang="ts" setup>
/**
 * ChatSidebar：会话侧边栏。
 * - 品牌区 + 新建对话按钮 + 搜索框 + 会话列表（按时间分组）+ 用户菜单。
 * 复刻 React Sidebar 的核心交互。
 */
import {computed, nextTick, onBeforeUnmount, ref, watch} from 'vue'
import {useRouter} from 'vue-router'
import {
  ChatLineRound,
  Document,
  EditPen,
  Loading as LoadingIcon,
  Message as MessageIcon,
  MoreFilled,
  Plus,
  Search as SearchIcon
} from '@element-plus/icons-vue'
import {ElMessageBox} from 'element-plus'
import {useChatStore} from '@/stores/chat'

const props = defineProps<{
  isOpen: boolean
}>()

const emit = defineEmits<{
  (e: 'close'): void
}>()

const store = useChatStore()
const router = useRouter()

const query = ref('')
const renamingId = ref<string | null>(null)
const renameValue = ref('')
const renameInputRef = ref<HTMLInputElement | null>(null)
const userMenuRef = ref<HTMLElement | null>(null)
const userMenuOpen = ref(false)

watch(() => props.isOpen, (open) => {
  if (open) {
    if (store.sessions.length === 0) {
      store.fetchSessions().catch(() => null)
    }
  }
})

const filteredSessions = computed(() => {
  const keyword = query.value.trim().toLowerCase()
  if (!keyword) return store.sessions
  return store.sessions.filter((session) => {
    return session.title.toLowerCase().includes(keyword) || session.id.toLowerCase().includes(keyword)
  })
})

const groupedSessions = computed(() => {
  const now = new Date()
  const groups = new Map<string, typeof filteredSessions.value>()
  const order: string[] = []
  filteredSessions.value.forEach((session) => {
    const date = session.lastTime ? new Date(session.lastTime) : now
    const valid = !Number.isNaN(date.getTime()) ? date : now
    const diff = Math.max(0, Math.floor((now.getTime() - valid.getTime()) / 86400000))
    let label = '更早'
    if (diff === 0) label = '今天'
    else if (diff <= 7) label = '7天内'
    else if (diff <= 30) label = '30天内'
    if (!groups.has(label)) {
      groups.set(label, [])
      order.push(label)
    }
    groups.get(label)!.push(session)
  })
  return order.map((label) => ({label, items: groups.get(label)!}))
})

watch(renamingId, async (id) => {
  if (id) {
    await nextTick()
    renameInputRef.value?.focus()
    renameInputRef.value?.select()
  }
})

function handleNewChat() {
  store.createSession().catch(() => null)
  router.push('/chat').catch(() => null)
  emit('close')
}

function handleSelectSession(id: string) {
  if (renamingId.value) return
  store.selectSession(id).catch(() => null)
  router.push(`/chat/${id}`).catch(() => null)
  emit('close')
}

function startRename(id: string, title: string) {
  renamingId.value = id
  renameValue.value = title
}

function cancelRename() {
  renamingId.value = null
  renameValue.value = ''
}

function commitRename() {
  if (!renamingId.value) return
  // 当前后端未提供重命名接口，前端仅本地更新标题。
  const next = renameValue.value.trim()
  if (next) {
    store.sessions = store.sessions.map((s) =>
      s.id === renamingId.value ? {...s, title: next} : s
    )
  }
  cancelRename()
}

async function confirmDelete(id: string, title: string) {
  try {
    await ElMessageBox.confirm(`[${title || '该会话'}] 将被永久删除，无法恢复。`, '删除该会话？', {
      type: 'warning',
      confirmButtonText: '删除',
      cancelButtonText: '取消'
    })
    const isCurrent = store.currentSessionId === id
    await store.deleteSession(id)
    if (isCurrent) {
      router.push('/chat').catch(() => null)
    }
  } catch {
    // 取消删除
  }
}

function handleDocumentClick(e: MouseEvent) {
  if (!userMenuOpen.value) return
  const target = e.target as Node | null
  if (userMenuRef.value && target && !userMenuRef.value.contains(target)) {
    userMenuOpen.value = false
  }
}

onBeforeUnmount(() => {
  document.removeEventListener('click', handleDocumentClick)
})

watch(userMenuOpen, (open) => {
  if (open) {
    document.addEventListener('click', handleDocumentClick)
  } else {
    document.removeEventListener('click', handleDocumentClick)
  }
})
</script>

<template>
  <div
    :class="{ 'sidebar-mask--open': isOpen }"
    class="sidebar-mask"
    @click="emit('close')"
  />
  <aside :class="{ 'chat-sidebar--open': isOpen }" class="chat-sidebar">
    <div class="chat-sidebar__brand">
      <div class="chat-sidebar__logo">
        <el-icon>
          <ChatLineRound/>
        </el-icon>
      </div>
      <div>
        <p class="chat-sidebar__title">KonwRag</p>
        <p class="chat-sidebar__subtitle">Powered by AI</p>
      </div>
    </div>

    <div class="chat-sidebar__quick">
      <div class="quick-card" @click="handleNewChat">
        <span class="quick-card__icon">
          <el-icon><Plus/></el-icon>
        </span>
        <span class="quick-card__body">
          <span class="quick-card__title">新建对话</span>
          <span class="quick-card__desc">从空白开始</span>
        </span>
      </div>
    </div>

    <div class="chat-sidebar__search">
      <el-icon class="chat-sidebar__search-icon">
        <SearchIcon/>
      </el-icon>
      <input
        v-model="query"
        class="chat-sidebar__search-input"
        placeholder="搜索对话..."
        type="text"
      />
    </div>

    <div class="chat-sidebar__list">
      <div v-if="store.sessions.length === 0 && store.isLoading" class="chat-sidebar__loading">
        <el-icon class="is-loading">
          <LoadingIcon/>
        </el-icon>
        <span>加载会话中</span>
      </div>
      <div v-else-if="filteredSessions.length === 0" class="chat-sidebar__empty">
        <el-icon>
          <MessageIcon/>
        </el-icon>
        <p>暂无对话记录</p>
      </div>
      <div v-else>
        <div
          v-for="(group, index) in groupedSessions"
          :key="group.label"
          :class="{ 'chat-sidebar__group--first': index === 0 }"
          class="chat-sidebar__group"
        >
          <p class="chat-sidebar__group-label">{{ group.label }}</p>
          <div
            v-for="session in group.items"
            :key="session.id"
            :class="{ 'chat-sidebar__item--active': store.currentSessionId === session.id }"
            class="chat-sidebar__item"
            role="button"
            tabindex="0"
            @click="handleSelectSession(session.id)"
            @keydown.enter="handleSelectSession(session.id)"
          >
            <input
              v-if="renamingId === session.id"
              ref="renameInputRef"
              v-model="renameValue"
              class="chat-sidebar__rename-input"
              @blur="commitRename"
              @click.stop
              @keydown.enter.prevent="commitRename"
              @keydown.esc.prevent="cancelRename"
            />
            <span v-else class="chat-sidebar__item-title">{{ session.title || '新对话' }}</span>
            <el-dropdown
              v-if="renamingId !== session.id"
              trigger="click"
              @command="(cmd: string) => {
                if (cmd === 'rename') startRename(session.id, session.title || '新对话')
                if (cmd === 'delete') confirmDelete(session.id, session.title || '新对话')
              }"
              @click.stop
            >
              <button
                :class="{ 'chat-sidebar__item-action--show': store.currentSessionId === session.id }"
                aria-label="会话操作"
                class="chat-sidebar__item-action"
                type="button"
                @click.stop
              >
                <el-icon>
                  <MoreFilled/>
                </el-icon>
              </button>
              <template #dropdown>
                <el-dropdown-menu>
                  <el-dropdown-item command="rename">
                    <el-icon>
                      <EditPen/>
                    </el-icon>
                    重命名
                  </el-dropdown-item>
                  <el-dropdown-item command="delete" divided>
                    <el-icon style="color: #ef4444">
                      <Document/>
                    </el-icon>
                    <span style="color: #ef4444">删除</span>
                  </el-dropdown-item>
                </el-dropdown-menu>
              </template>
            </el-dropdown>
          </div>
        </div>
      </div>
    </div>

    <div ref="userMenuRef" class="chat-sidebar__user">
      <button
        class="chat-sidebar__user-trigger"
        type="button"
        @click="userMenuOpen = !userMenuOpen"
      >
        <span class="chat-sidebar__user-avatar">U</span>
        <span class="chat-sidebar__user-name">用户</span>
        <el-icon class="chat-sidebar__user-caret">
          <MoreFilled/>
        </el-icon>
      </button>
    </div>
  </aside>
</template>

<style scoped>
.sidebar-mask {
  position: fixed;
  inset: 0;
  background: rgba(15, 23, 42, 0.3);
  backdrop-filter: blur(2px);
  opacity: 0;
  pointer-events: none;
  transition: opacity 0.2s ease;
  z-index: 30;
}

.sidebar-mask--open {
  opacity: 1;
  pointer-events: auto;
}

@media (min-width: 1024px) {
  .sidebar-mask {
    display: none;
  }
}

.chat-sidebar {
  position: fixed;
  left: 0;
  top: 0;
  bottom: 0;
  width: 280px;
  display: flex;
  flex-direction: column;
  padding: 12px;
  background: #fafafa;
  border-right: 1px solid #f0f0f0;
  z-index: 40;
  transform: translateX(-100%);
  transition: transform 0.24s ease;
}

.chat-sidebar--open {
  transform: translateX(0);
}

@media (min-width: 1024px) {
  .chat-sidebar {
    position: static;
    transform: translateX(0);
    height: 100vh;
  }
}

.chat-sidebar__brand {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 8px 4px 12px;
  border-bottom: 1px solid #f0f0f0;
}

.chat-sidebar__logo {
  width: 40px;
  height: 40px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: #3b82f6;
  border-radius: 10px;
  color: #ffffff;
  font-size: 20px;
}

.chat-sidebar__title {
  margin: 0;
  font-size: 16px;
  font-weight: 600;
  color: #1a1a1a;
}

.chat-sidebar__subtitle {
  margin: 2px 0 0;
  font-size: 12px;
  color: #999999;
}

.chat-sidebar__quick {
  margin: 12px 0;
}

.quick-card {
  display: flex;
  align-items: center;
  gap: 12px;
  width: 100%;
  padding: 12px 16px;
  border: 0;
  border-radius: 16px;
  background: linear-gradient(135deg, #f0f9ff, #fef3c7);
  cursor: pointer;
  text-align: left;
  transition: transform 0.15s ease, box-shadow 0.15s ease;
}

.quick-card:hover {
  transform: translateY(-1px);
  box-shadow: 0 8px 20px rgba(15, 23, 42, 0.08);
}

.quick-card__icon {
  width: 44px;
  height: 44px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, #60a5fa, #2563eb);
  border-radius: 16px;
  color: #ffffff;
  font-size: 18px;
  box-shadow: 0 6px 14px rgba(37, 99, 235, 0.3);
}

.quick-card__body {
  display: flex;
  flex-direction: column;
}

.quick-card__title {
  font-size: 14px;
  font-weight: 600;
  color: #1f2937;
}

.quick-card__desc {
  font-size: 12px;
  color: #94a3b8;
}

.chat-sidebar__search {
  display: flex;
  align-items: center;
  gap: 8px;
  margin: 0 0 12px;
  padding: 8px 12px;
  background: #ffffff;
  border: 1px solid #e5e7eb;
  border-radius: 12px;
  transition: border-color 0.15s ease;
}

.chat-sidebar__search:focus-within {
  border-color: #93c5fd;
}

.chat-sidebar__search-icon {
  font-size: 16px;
  color: #9ca3af;
}

.chat-sidebar__search-input {
  flex: 1;
  border: 0;
  outline: 0;
  background: transparent;
  font-size: 13px;
  color: #1f2937;
}

.chat-sidebar__search-input::placeholder {
  color: #9ca3af;
}

.chat-sidebar__list {
  flex: 1;
  min-height: 0;
  position: relative;
  overflow-y: auto;
}

.chat-sidebar__list::after {
  content: '';
  position: absolute;
  left: 0;
  right: 0;
  bottom: 0;
  height: 20px;
  background: linear-gradient(180deg, transparent, #fafafa);
  pointer-events: none;
}

.chat-sidebar__loading,
.chat-sidebar__empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 8px;
  padding: 40px 20px;
  color: #999999;
  font-size: 14px;
}

.chat-sidebar__empty .el-icon {
  font-size: 48px;
}

.chat-sidebar__group {
  margin-top: 16px;
}

.chat-sidebar__group--first {
  margin-top: 0;
}

.chat-sidebar__group-label {
  margin: 0 0 6px 12px;
  font-size: 12px;
  color: #999999;
}

.chat-sidebar__item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 12px;
  border-radius: 8px;
  cursor: pointer;
  transition: background 0.15s ease, color 0.15s ease;
  color: #333333;
  font-size: 14px;
  line-height: 22px;
}

.chat-sidebar__item:hover {
  background: #f5f5f5;
}

.chat-sidebar__item--active {
  background: #dbeafe;
  color: #2563eb;
}

.chat-sidebar__item-title {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.chat-sidebar__item-action {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 24px;
  height: 24px;
  border: 0;
  border-radius: 4px;
  background: transparent;
  color: #666666;
  cursor: pointer;
  opacity: 0;
  transition: opacity 0.15s ease, background 0.15s ease;
}

.chat-sidebar__item:hover .chat-sidebar__item-action,
.chat-sidebar__item-action--show {
  opacity: 1;
}

.chat-sidebar__item-action:hover {
  background: rgba(0, 0, 0, 0.06);
}

.chat-sidebar__rename-input {
  flex: 1;
  height: 24px;
  border: 1px solid #e5e5e5;
  border-radius: 4px;
  padding: 0 6px;
  font-size: 14px;
  outline: none;
}

.chat-sidebar__rename-input:focus {
  border-color: #2563eb;
}

.chat-sidebar__user {
  position: relative;
  margin-top: 8px;
  padding: 8px 4px 0;
  border-top: 1px solid #f0f0f0;
}

.chat-sidebar__user-trigger {
  display: flex;
  align-items: center;
  gap: 8px;
  width: 100%;
  padding: 8px;
  border: 0;
  border-radius: 8px;
  background: transparent;
  cursor: pointer;
  text-align: left;
  transition: background 0.15s ease;
}

.chat-sidebar__user-trigger:hover {
  background: #f5f5f5;
}

.chat-sidebar__user-avatar {
  width: 32px;
  height: 32px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: #3b82f6;
  border-radius: 50%;
  color: #ffffff;
  font-size: 14px;
  font-weight: 500;
}

.chat-sidebar__user-name {
  flex: 1;
  font-size: 14px;
  color: #1a1a1a;
  font-weight: 500;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.chat-sidebar__user-caret {
  font-size: 16px;
  color: #999999;
}

.chat-sidebar__user-menu {
  position: absolute;
  left: 4px;
  right: 4px;
  bottom: calc(100% + 4px);
  background: #ffffff;
  border: 1px solid #f0f0f0;
  border-radius: 8px;
  box-shadow: 0 8px 24px rgba(15, 23, 42, 0.12);
  padding: 4px 0;
  z-index: 5;
}

.chat-sidebar__user-menu-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 12px;
  font-size: 14px;
  color: #333333;
  text-decoration: none;
  transition: background 0.15s ease;
}

.chat-sidebar__user-menu-item:hover {
  background: #f5f5f5;
}
</style>
