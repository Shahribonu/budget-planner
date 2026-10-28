import { createRouter, createWebHashHistory } from 'vue-router'
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
    { path: '/plan', name: 'plan', component: () => import('@/views/PlanView.vue'), meta: { tabs: true } },
    { path: '/accounts', name: 'accounts', component: () => import('@/views/AccountsView.vue'), meta: { sub: true } },
    { path: '/categories', name: 'categories', component: () => import('@/views/CategoriesView.vue'), meta: { sub: true } },
    { path: '/premium', name: 'premium', component: () => import('@/views/PremiumView.vue'), meta: { sub: true } },
    { path: '/recurring', name: 'recurring', component: () => import('@/views/RecurringView.vue'), meta: { sub: true } },
    { path: '/debts', name: 'debts', component: () => import('@/views/DebtsView.vue'), meta: { sub: true } },
    { path: '/debts/new', name: 'debt-new', component: () => import('@/views/DebtFormView.vue'), meta: { sub: true } },
    { path: '/debts/:id(\\d+)', name: 'debt', component: () => import('@/views/DebtView.vue'), meta: { sub: true } },
    { path: '/installments', name: 'installments', component: () => import('@/views/InstallmentsView.vue'), meta: { sub: true } },
    { path: '/installments/new', name: 'installment-new', component: () => import('@/views/InstallmentFormView.vue'), meta: { sub: true } },
    { path: '/installments/:id(\\d+)', name: 'installment', component: () => import('@/views/InstallmentView.vue'), meta: { sub: true } },
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
    onboarded = (await user.load()).onboarded
  } catch (e) {
    user.error = e
    return true // App.vue renders the error state
  }
  if (!onboarded && to.name !== 'onboarding') return { name: 'onboarding', replace: true }
  if (onboarded && to.name === 'onboarding') return { name: 'home', replace: true }
  return true
})
