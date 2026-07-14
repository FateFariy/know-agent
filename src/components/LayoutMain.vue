<script setup lang="ts">
import { ref } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import {
  ChatDotRound,
  Setting,
  Menu,
  Close,
  Cpu
} from '@element-plus/icons-vue'

const router = useRouter()
const route = useRoute()
const isCollapsed = ref(false)
const isMobileMenuOpen = ref(false)

const menuItems = [
  { path: '/chat', name: '智能问答', icon: ChatDotRound },
]

function isActive(path: string): boolean {
  return route.path === path
}

function toggleSidebar() {
  isCollapsed.value = !isCollapsed.value
}

function toggleMobileMenu() {
  isMobileMenuOpen.value = !isMobileMenuOpen.value
}

function navigate(path: string) {
  router.push(path)
  isMobileMenuOpen.value = false
}

function goToAdmin() {
  router.push('/admin')
}
</script>

<template>
  <div class="layout-container">
    <aside class="sidebar" :class="{ 'is-collapsed': isCollapsed, 'is-mobile-open': isMobileMenuOpen }">
      <div class="sidebar-header">
        <div class="logo">
          <Cpu class="logo-icon" />
          <span v-show="!isCollapsed" class="logo-text">智能知识库</span>
        </div>
        <button v-show="!isCollapsed" class="collapse-btn" @click="toggleSidebar">
          <Close :size="18" />
        </button>
      </div>

      <nav class="sidebar-nav">
        <ul>
          <li v-for="item in menuItems" :key="item.path" class="nav-item" :class="{ active: isActive(item.path) }"
            @click="navigate(item.path)">
            <component :is="item.icon" class="nav-icon" />
            <span v-show="!isCollapsed" class="nav-text">{{ item.name }}</span>
          </li>
        </ul>
      </nav>

      <div class="sidebar-footer">
        <button class="admin-btn" @click="goToAdmin">
          <Setting :size="18" />
          <span v-show="!isCollapsed" class="admin-text">管理</span>
        </button>
      </div>
    </aside>

    <div class="mobile-overlay" v-show="isMobileMenuOpen" @click="toggleMobileMenu"></div>

    <main class="main-content">
      <header class="main-header">
        <button class="mobile-menu-btn" @click="toggleMobileMenu">
          <Menu :size="20" />
        </button>
        <h1 class="page-title">{{menuItems.find(item => isActive(item.path))?.name || '智能知识库'}}</h1>
        <button class="desktop-admin-btn" @click="goToAdmin">
          <Setting :size="18" />
          <span>管理</span>
        </button>
      </header>
      <router-view />
    </main>
  </div>
</template>

<style scoped>
.layout-container {
  display: flex;
  height: 100%;
  overflow: hidden;
}

.sidebar {
  width: 200px;
  background: linear-gradient(180deg, #0f172a 0%, #1e293b 100%);
  color: #f1f5f9;
  display: flex;
  flex-direction: column;
  transition: width 0.3s ease;
  position: fixed;
  left: 0;
  top: 0;
  bottom: 0;
  z-index: 100;
  box-shadow: 4px 0 20px rgba(0, 0, 0, 0.15);
}

.sidebar.is-collapsed {
  width: 60px;
}

.sidebar.is-mobile-open {
  transform: translateX(0);
}

.sidebar-header {
  padding: 16px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  border-bottom: 1px solid #334155;
}

.logo {
  display: flex;
  align-items: center;
  gap: 10px;
}

.logo-icon {
  width: 32px;
  height: 32px;
  color: #60a5fa;
}

.logo-text {
  font-size: 18px;
  font-weight: 700;
  color: #f8fafc;
}

.collapse-btn {
  background: transparent;
  border: none;
  color: #94a3b8;
  cursor: pointer;
  padding: 4px;
  border-radius: 4px;
  transition: all 0.2s;
}

.collapse-btn:hover {
  background: rgba(255, 255, 255, 0.1);
  color: #f1f5f9;
}

.sidebar-nav {
  flex: 1;
  padding: 12px 0;
}

.sidebar-nav ul {
  list-style: none;
  padding: 0;
  margin: 0;
}

.nav-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px 16px;
  cursor: pointer;
  transition: all 0.2s;
  border-left: 3px solid transparent;
  margin: 0 4px;
  border-radius: 0 8px 8px 0;
}

.nav-item:hover {
  background: rgba(255, 255, 255, 0.05);
}

.nav-item.active {
  background: rgba(96, 165, 250, 0.15);
  border-left-color: #60a5fa;
  color: #bfdbfe;
}

.nav-icon {
  width: 20px;
  height: 20px;
  flex-shrink: 0;
}

.nav-text {
  font-size: 14px;
  font-weight: 500;
}

.sidebar-footer {
  padding: 12px;
  border-top: 1px solid #334155;
}

.admin-btn {
  display: flex;
  align-items: center;
  gap: 10px;
  width: 100%;
  padding: 10px 12px;
  background: rgba(255, 255, 255, 0.05);
  border: 1px solid #334155;
  border-radius: 8px;
  color: #94a3b8;
  cursor: pointer;
  transition: all 0.2s;
  font-size: 13px;
}

.admin-btn:hover {
  background: rgba(96, 165, 250, 0.15);
  border-color: #60a5fa;
  color: #f1f5f9;
}

.main-content {
  flex: 1;
  display: flex;
  flex-direction: column;
  margin-left: 200px;
  transition: margin-left 0.3s ease;
}

.sidebar.is-collapsed+.main-content {
  margin-left: 60px;
}

.main-header {
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 14px 24px;
  background: #fff;
  border-bottom: 1px solid #e2e8f0;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.05);
}

.mobile-menu-btn {
  display: none;
  background: transparent;
  border: none;
  color: #475569;
  cursor: pointer;
  padding: 8px;
  border-radius: 6px;
  transition: all 0.2s;
}

.mobile-menu-btn:hover {
  background: #f1f5f9;
}

.page-title {
  flex: 1;
  font-size: 18px;
  font-weight: 600;
  color: #1e293b;
  margin: 0;
}

.desktop-admin-btn {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 8px 14px;
  background: #f1f5f9;
  border: 1px solid #e2e8f0;
  border-radius: 8px;
  font-size: 13px;
  color: #475569;
  cursor: pointer;
  transition: all 0.2s;
}

.desktop-admin-btn:hover {
  background: #eff6ff;
  border-color: #3b82f6;
  color: #3b82f6;
}

.mobile-overlay {
  display: none;
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.5);
  z-index: 99;
}

@media (max-width: 768px) {
  .sidebar {
    transform: translateX(-100%);
  }

  .sidebar.is-mobile-open {
    width: 200px;
  }

  .main-content {
    margin-left: 0;
  }

  .mobile-menu-btn {
    display: block;
  }

  .desktop-admin-btn {
    display: none;
  }

  .mobile-overlay {
    display: block;
  }
}

@media (max-width: 480px) {
  .main-header {
    padding: 12px 16px;
  }

  .page-title {
    font-size: 16px;
  }
}
</style>
