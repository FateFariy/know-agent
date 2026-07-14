import { createRouter, createWebHistory } from 'vue-router'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: '/',
      name: 'Home',
      redirect: '/chat',
    },
    {
      path: '/chat',
      name: 'Chat',
      component: () => import('@/views/ChatView.vue'),
    },
    {
      path: '/documents',
      name: 'Documents',
      component: () => import('@/views/DocumentView.vue'),
    },
    {
      path: '/knowledge',
      name: 'Knowledge',
      component: () => import('@/views/KnowledgeView.vue'),
    },
  ],
})

export default router
