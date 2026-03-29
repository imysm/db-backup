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
        path: '/jobs/new',
        name: 'JobWizard',
        component: () => import('@/views/JobWizard.vue'),
        meta: { title: '创建任务' }
      },
      {
        path: '/jobs/:id',
        name: 'JobDetail',
        component: () => import('@/views/JobDetail.vue'),
        meta: { title: '任务详情' }
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
        path: '/storage',
        name: 'Storage',
        component: () => import('@/views/Storage.vue'),
        meta: { title: '存储管理' }
      },
      {
        path: '/alert-rules',
        name: 'AlertRules',
        component: () => import('@/views/AlertRules.vue'),
        meta: { title: '告警规则' }
      },
      {
        path: '/health',
        name: 'HealthCheck',
        component: () => import('@/views/HealthCheck.vue'),
        meta: { title: '系统体检' }
      },
      {
        path: '/storage-forecast',
        name: 'StorageForecast',
        component: () => import('@/views/StorageForecast.vue'),
        meta: { title: '容量预测' }
      },
      {
        path: '/backup-impact',
        name: 'BackupImpact',
        component: () => import('@/views/BackupImpact.vue'),
        meta: { title: '备份影响分析' }
      },
      {
        path: '/alerts',
        name: 'Alerts',
        component: () => import('@/views/Alerts.vue'),
        meta: { title: '告警中心' }
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
