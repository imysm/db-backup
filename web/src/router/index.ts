import { createRouter, createWebHistory } from 'vue-router'
import type { RouteRecordRaw } from 'vue-router'

const routes: RouteRecordRaw[] = [
  {
    path: '/login',
    name: 'Login',
    component: () => import('@/views/Login.vue'),
    meta: { requiresAuth: false }
  },
  {
    path: '/',
    component: () => import('@/views/Layout.vue'),
    redirect: '/dashboard',
    meta: { requiresAuth: true },
    children: [
      {
        path: '/dashboard',
        name: 'Dashboard',
        component: () => import('@/views/Dashboard.vue'),
        meta: { title: '仪表盘' }
      },
      {
        path: '/jobs',
        name: 'Jobs',
        component: () => import('@/views/Jobs.vue'),
        meta: { title: '备份任务' }
      },
      {
        path: '/records',
        name: 'Records',
        component: () => import('@/views/Records.vue'),
        meta: { title: '备份记录' }
      },
      {
        path: '/verify',
        name: 'Verify',
        component: () => import('@/views/Verify.vue'),
        meta: { title: '备份验证' }
      },
      {
        path: '/restore',
        name: 'Restore',
        component: () => import('@/views/Restore.vue'),
        meta: { title: '恢复管理' }
      },
      {
        path: '/settings',
        name: 'Settings',
        component: () => import('@/views/Settings.vue'),
        meta: { title: '系统设置' }
      }
    ]
  }
]

const router = createRouter({
  history: createWebHistory(),
  routes
})

// 路由守卫：未认证重定向到 /login
router.beforeEach((to, _from, next) => {
  const apiKey = localStorage.getItem('api_key')
  if (to.meta.requiresAuth !== false && !apiKey) {
    next('/login')
  } else {
    next()
  }
})

export default router
