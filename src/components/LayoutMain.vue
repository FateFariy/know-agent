<script setup lang="ts">
import { ref } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import {
  Reading,
  ChatRound,
  FolderOpened,
  Menu,
  Close,
} from '@element-plus/icons-vue'

const router = useRouter()
const route = useRoute()
const isCollapsed = ref(false)

const menuItems = [
  { path: '/chat', name: '智能问答', icon: ChatRound },
  { path: '/documents', name: '文档管理', icon: FolderOpened },
  { path: '/knowledge', name: '知识库', icon: Reading },
]

function navigateTo(path: string) {
  router.push(path)
}
</script>

<template>
  <div class="layout-container">
    <aside :class="['sidebar', { collapsed: isCollapsed }]">
      <div class="logo">
        <Reading :size="28" />
        <span v-show="!isCollapsed" class="logo-text">知识助手</span>
      </div>
      <nav class="menu">
        <button
          v-for="item in menuItems"
          :key="item.path"
          :class="['menu-item', { active: route.path === item.path }]"
          @click="navigateTo(item.path)"
        >
          <component :is="item.icon" :size="20" />
          <span v-show="!isCollapsed" class="menu-text">{{ item.name }}</span>
        </button>
      </nav>
      <button class="collapse-btn" @click="isCollapsed = !isCollapsed">
        <Menu v-if="isCollapsed" :size="18" />
        <Close v-else :size="18" />
      </button>
    </aside>
    <main class="main-content">
      <router-view />
    </main>
  </div>
</template>

<style scoped>
.layout-container {
  display: flex;
  height: 100%;
}

.sidebar {
  width: 240px;
  background: linear-gradient(180deg, #1e293b 0%, #0f172a 100%);
  color: #fff;
  display: flex;
  flex-direction: column;
  transition: width 0.3s ease;
  position: fixed;
  left: 0;
  top: 0;
  height: 100%;
  z-index: 100;
}

.sidebar.collapsed {
  width: 64px;
}

.logo {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 10px;
  padding: 20px;
  border-bottom: 1px solid rgba(255, 255, 255, 0.1);
}

.logo-text {
  font-size: 18px;
  font-weight: 600;
}

.menu {
  flex: 1;
  padding: 16px 8px;
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.menu-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px 16px;
  border: none;
  background: transparent;
  color: #94a3b8;
  cursor: pointer;
  border-radius: 8px;
  transition: all 0.2s ease;
  font-size: 14px;
}

.menu-item:hover {
  background: rgba(255, 255, 255, 0.1);
  color: #fff;
}

.menu-item.active {
  background: #3b82f6;
  color: #fff;
}

.menu-text {
  flex: 1;
  text-align: left;
}

.collapse-btn {
  padding: 12px;
  border: none;
  background: rgba(255, 255, 255, 0.1);
  color: #94a3b8;
  cursor: pointer;
  transition: all 0.2s ease;
}

.collapse-btn:hover {
  background: rgba(255, 255, 255, 0.2);
  color: #fff;
}

.main-content {
  flex: 1;
  margin-left: 240px;
  transition: margin-left 0.3s ease;
  background: #f8fafc;
  overflow-y: auto;
}

.sidebar.collapsed + .main-content {
  margin-left: 64px;
}
</style>
