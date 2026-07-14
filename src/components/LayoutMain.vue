<script setup lang="ts">
import { useRouter, useRoute } from 'vue-router'
import { Setting, Menu, Cpu } from '@element-plus/icons-vue'

const router = useRouter()
const route = useRoute()

function goToAdmin() {
  router.push('/admin')
}

function isAdminPage(): boolean {
  return route.path === '/admin'
}
</script>

<template>
  <div class="layout-container">
    <router-view />
  </div>
</template>

<style scoped>
.layout-container {
  height: 100%;
  display: flex;
  flex-direction: column;
}
</style>
