import { createRouter, createWebHistory } from 'vue-router'
import Layout from '@/layout/index.vue'

const routes = [
  {
    path: '/login',
    component: () => import('@/views/login/index.vue'),
    hidden: true
  },
  {
    path: '/register',
    component: () => import('@/views/register/index.vue'),
    hidden: true
  },
  {
    path: '/oauth/authorize',
    component: () => import('@/views/oauth/authorize.vue'),
    hidden: true,
    meta: { title: '统一身份认证' }
  },
  {
    path: '/oauth/callback',
    component: () => import('@/views/oauth/callback.vue'),
    hidden: true,
    meta: { title: '授权回调' }
  },
  {
    path: '/',
    component: Layout,
    redirect: '/dashboard',
    children: [
      {
        path: 'dashboard',
        name: 'Dashboard',
        component: () => import('@/views/dashboard/index.vue'),
        meta: { title: '首页', icon: 'HomeFilled' }
      }
    ]
  },
  {
    path: '/user',
    component: Layout,
    redirect: '/user/personal',
    meta: { title: '用户管理', icon: 'User' },
    children: [
      {
        path: 'personal',
        name: 'UserPersonal',
        component: () => import('@/views/user/personal.vue'),
        meta: { title: '个体用户', icon: 'UserFilled' }
      },
      {
        path: 'corp',
        name: 'UserCorp',
        component: () => import('@/views/user/corp.vue'),
        meta: { title: '企业用户', icon: 'OfficeBuilding' }
      },
      {
        path: 'audit',
        name: 'UserAudit',
        component: () => import('@/views/user/audit.vue'),
        meta: { title: '审核管理', icon: 'CircleCheck' }
      }
    ]
  },
  {
    path: '/system',
    component: Layout,
    redirect: '/system/user',
    meta: { title: '系统管理', icon: 'Setting' },
    children: [
      {
        path: 'user',
        name: 'SystemUser',
        component: () => import('@/views/system/user.vue'),
        meta: { title: '管理员', icon: 'UserFilled' }
      },
      {
        path: 'role',
        name: 'SystemRole',
        component: () => import('@/views/system/role.vue'),
        meta: { title: '角色', icon: 'Avatar' }
      },
      {
        path: 'menu',
        name: 'SystemMenu',
        component: () => import('@/views/system/menu.vue'),
        meta: { title: '菜单', icon: 'Menu' }
      }
    ]
  },
  {
    path: '/app',
    component: Layout,
    redirect: '/app/list',
    meta: { title: '应用接入', icon: 'Connection' },
    children: [
      {
        path: 'list',
        name: 'AppList',
        component: () => import('@/views/app/list.vue'),
        meta: { title: '应用管理', icon: 'Grid' }
      },
      {
        path: 'grant',
        name: 'AppGrant',
        component: () => import('@/views/app/grant.vue'),
        meta: { title: '授权管理', icon: 'Key' }
      }
    ]
  },
  {
    path: '/stat',
    component: Layout,
    redirect: '/stat/index',
    meta: { title: '统计分析', icon: 'TrendCharts' },
    children: [
      {
        path: 'index',
        name: 'StatIndex',
        component: () => import('@/views/stat/index.vue'),
        meta: { title: '统计概览', icon: 'DataAnalysis' }
      }
    ]
  },
  {
    path: '/log',
    component: Layout,
    redirect: '/log/oper',
    meta: { title: '日志管理', icon: 'Document' },
    children: [
      {
        path: 'oper',
        name: 'LogOper',
        component: () => import('@/views/log/oper.vue'),
        meta: { title: '操作日志', icon: 'Tickets' }
      },
      {
        path: 'login',
        name: 'LogLogin',
        component: () => import('@/views/log/login.vue'),
        meta: { title: '登录日志', icon: 'Document' }
      }
    ]
  },
  {
    path: '/redirect',
    component: Layout,
    hidden: true,
    children: [
      {
        path: '/redirect/:path(.*)',
        name: 'Redirect',
        component: () => import('@/views/redirect/index.vue')
      }
    ]
  },
  {
    path: '/profile',
    component: Layout,
    hidden: true,
    children: [
      {
        path: '',
        name: 'Profile',
        component: () => import('@/views/profile/index.vue'),
        meta: { title: '个人中心', icon: 'User' }
      }
    ]
  },
  {
    path: '/:pathMatch(.*)*',
    component: () => import('@/views/error/404.vue'),
    hidden: true
  }
]

const router = createRouter({
  history: createWebHistory(),
  routes,
  scrollBehavior: () => ({ left: 0, top: 0 })
})

export { routes }

// 路由守卫
const whiteList = ['/login', '/register', '/oauth/authorize', '/oauth/callback']
router.beforeEach((to, from, next) => {
  document.title = (to.meta?.title || '统一身份认证') + ' - UAS'
  const token = localStorage.getItem('uas_token')
  if (token) {
    if (to.path === '/login') {
      next('/')
    } else {
      next()
    }
  } else {
    if (whiteList.includes(to.path)) {
      next()
    } else {
      next(`/login?redirect=${encodeURIComponent(to.fullPath)}`)
    }
  }
})

export default router
