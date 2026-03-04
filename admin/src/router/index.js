/**
 * 管理后台路由配置
 *
 * 路由结构：
 * - /login → 管理员登录页（无需认证）
 * - / → 后台主布局（需要认证 + admin 角色）
 *   - /dashboard → 仪表盘
 *   - /user/list → 用户管理列表
 *   - /user/detail/:id → 用户详情
 *
 * 路由守卫：
 * - 未登录 → 重定向到 /login
 * - 已登录访问 /login → 重定向到 /dashboard
 */
import { createRouter, createWebHashHistory } from 'vue-router'

const routes = [
  {
    path: '/login',
    name: 'Login',
    component: () => import('@/views/login/index.vue'),
    meta: { title: '管理员登录', requiresAuth: false }
  },
  {
    path: '/',
    component: () => import('@/views/layout/index.vue'),
    redirect: '/dashboard',
    meta: { requiresAuth: true },
    children: [
      {
        path: 'dashboard',
        name: 'Dashboard',
        component: () => import('@/views/dashboard/index.vue'),
        meta: { title: '仪表盘' }
      },
      {
        path: 'user/list',
        name: 'UserList',
        component: () => import('@/views/user/list.vue'),
        meta: { title: '用户列表' }
      },
      {
        path: 'user/detail/:id',
        name: 'UserDetail',
        component: () => import('@/views/user/detail.vue'),
        meta: { title: '用户详情' }
      },
      {
        path: 'monitor/online',
        name: 'OnlineMonitor',
        component: () => import('@/views/monitor/online.vue'),
        meta: { title: '在线监控' }
      },
      {
        path: 'contact/list',
        name: 'ContactManage',
        component: () => import('@/views/contact/list.vue'),
        meta: { title: '好友管理' }
      },
      {
        path: 'group/list',
        name: 'GroupManage',
        component: () => import('@/views/group/list.vue'),
        meta: { title: '群组管理' }
      }
    ]
  }
]

const router = createRouter({
  history: createWebHashHistory(),
  routes
})

/**
 * 全局前置守卫
 * 检查登录状态，控制页面访问权限
 */
router.beforeEach((to, from, next) => {
  const token = localStorage.getItem('admin_token')

  document.title = to.meta.title
    ? `${to.meta.title} - EchoChat 管理后台`
    : 'EchoChat 管理后台'

  if (to.meta.requiresAuth === false) {
    if (token && to.path === '/login') {
      next('/dashboard')
    } else {
      next()
    }
    return
  }

  if (!token) {
    next({ path: '/login', query: { redirect: to.fullPath } })
    return
  }

  next()
})

export default router
