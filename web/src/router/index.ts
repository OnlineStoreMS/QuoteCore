import { createRouter, createWebHistory } from 'vue-router'
import AdminLayout from '../layouts/AdminLayout.vue'
import { redirectToPortal, ensureSession, clearToken } from '../utils/auth'

const APP_TITLE = 'QuoteCore - 报价中心'

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
    {
      path: '/auth/callback',
      name: 'AuthCallback',
      component: () => import('../views/AuthCallback.vue'),
      meta: { public: true },
    },
    {
      path: '/auth/logout',
      name: 'AuthLogout',
      component: () => import('../views/AuthLogout.vue'),
      meta: { public: true },
    },
    {
      path: '/',
      component: AdminLayout,
      redirect: '/dashboard',
      children: [
        { path: 'dashboard', name: 'Dashboard', component: () => import('../views/Dashboard.vue'), meta: { title: '工作台' } },
        { path: 'quotes', name: 'Quotes', component: () => import('../views/Quotes.vue'), meta: { title: '报价单' } },
        { path: 'quotes/new', name: 'QuoteCreate', component: () => import('../views/QuoteEdit.vue'), meta: { title: '新建报价' } },
        { path: 'quotes/:id', name: 'QuoteEdit', component: () => import('../views/QuoteEdit.vue'), meta: { title: '编辑报价' } },
        { path: 'templates', name: 'Templates', component: () => import('../views/Templates.vue'), meta: { title: '报价模板' } },
      ],
    },
  ],
})

router.beforeEach(async (to) => {
  if (to.meta.public) return true
  const ok = await ensureSession()
  if (!ok) {
    clearToken()
    redirectToPortal()
    return false
  }
  return true
})

router.afterEach((to) => {
  const page = to.meta.title as string | undefined
  document.title = page ? `${page} - ${APP_TITLE}` : APP_TITLE
})

export default router
