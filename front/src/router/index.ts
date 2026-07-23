import {createRouter, createWebHistory} from 'vue-router'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: '/',
      name: 'Home',
      redirect: '/chat'
    },
    {
      path: '/chat',
      name: 'Chat',
      component: () => import('@/views/ChatPage.vue'),
    },
    {
      path: '/chat/:sessionId',
      name: 'ChatSession',
      component: () => import('@/views/ChatPage.vue'),
    },
    {
      path: '/admin',
      component: () => import('@/views/admin/AdminLayoutView.vue'),
      meta: {
        layout: 'fullscreen',
        requiresAdminAuth: true
      },
      children: [
        {
          path: '',
          redirect: '/admin/dashboard'
        },
        {
          path: 'dashboard',
          name: 'AdminDashboard',
          component: () => import('@/views/admin/AdminDashboardView.vue'),
          meta: {
            title: '运营总览'
          }
        },
        {
          path: 'documents',
          name: 'AdminDocuments',
          component: () => import('@/views/admin/AdminDocumentListView.vue'),
          meta: {
            title: '文档接入'
          }
        },
        {
          path: 'documents/:documentId',
          name: 'AdminDocumentDetail',
          component: () => import('@/views/admin/AdminDocumentDetailView.vue'),
          meta: {
            title: '文档详情'
          }
        },
        {
          path: 'knowledge-route',
          name: 'AdminKnowledgeRoute',
          component: () => import('@/views/admin/AdminKnowledgeRouteView.vue'),
          meta: {
            title: '知识路由'
          }
        },
        {
          path: 'knowledge-route/traces',
          name: 'AdminKnowledgeRouteTrace',
          component: () => import('@/views/admin/AdminKnowledgeRouteTraceView.vue'),
          meta: {
            title: '路由追踪'
          }
        },
        {
          path: 'observability',
          name: 'AdminObservabilityList',
          component: () => import('@/views/admin/AdminObservabilityListView.vue'),
          meta: {
            title: '对话观测'
          }
        },
        {
          path: 'observability/:conversationId',
          name: 'AdminObservabilitySession',
          component: () => import('@/views/admin/AdminObservabilitySessionView.vue'),
          meta: {
            title: '会话链路'
          }
        },
        {
          path: 'observability/:conversationId/exchanges/:exchangeId',
          name: 'AdminObservabilityExchangeDetail',
          component: () => import('@/views/admin/AdminObservabilityDetailView.vue'),
          meta: {
            title: '轮次详情'
          }
        }
      ]
    }
  ]
})

export default router
