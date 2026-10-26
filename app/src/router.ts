import { createRouter, createWebHashHistory } from 'vue-router'
import { useAccountsStore } from '@/stores/accounts'
import { useUserStore } from '@/stores/user'
import HomeView from '@/views/HomeView.vue'

declare module 'vue-router' {
  interface RouteMeta {
    /** Show the bottom tab bar. */
    tabs?: boolean
    /** Sub-page: show Telegram BackButton. */
    sub?: boolean
  }
}

export const router = createRouter({
  // Hash history: Telegram opens the Mini App at a fixed URL and static hosting needs no rewrites.
  history: createWebHashHistory(),
  routes: [
    { path: '/', name: 'home', component: HomeView, meta: { tabs: true } },
    { path: '/activity', name: 'activity', component: () => import('@/views/ActivityView.vue'), meta: { tabs: true } },
    { path: '/plan', name: 'plan', component: () => import('@/views/PlaceholderView.vue'), meta: { tabs: true }, props: { title: 'tabs.plan', icon: '🎯' } },
    { path: '/insights', name: 'insights', component: () => import('@/views/PlaceholderView.vue'), meta: { tabs: true }, props: { title: 'tabs.insights', icon: '📊' } },
    { path: '/profile', name: 'profile', component: () => import('@/views/ProfileView.vue'), meta: { tabs: true } },
    { path: '/tx/new', name: 'tx-new', component: () => import('@/views/TransactionFormView.vue'), meta: { sub: true } },
    { path: '/tx/:id(\\d+)', name: 'tx-edit', component: () => import('@/views/TransactionFormView.vue'), meta: { sub: true } },
    { path: '/accounts/new', name: 'account-new', component: () => import('@/views/AccountView.vue'), meta: { sub: true } },
    { path: '/accounts/:id(\\d+)', name: 'account-edit', component: () => import('@/views/AccountView.vue'), meta: { sub: true } },
    { path: '/onboarding', name: 'onboarding', component: () => import('@/views/OnboardingView.vue') },
    { path: '/:pathMatch(.*)*', redirect: '/' },
  ],
})

router.beforeEach(async (to) => {
  const user = useUserStore()
  let onboarded: boolean
  try {
    const me = await user.load()
    onboarded = me.onboarded ?? (await useAccountsStore().load()).some((a) => !a.archived)
  } catch (e) {
    user.error = e
    return true // App.vue renders the error state
  }
  if (!onboarded && to.name !== 'onboarding') return { name: 'onboarding', replace: true }
  if (onboarded && to.name === 'onboarding') return { name: 'home', replace: true }
  return true
})
