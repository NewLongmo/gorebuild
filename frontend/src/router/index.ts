import { createRouter, createWebHistory } from 'vue-router';
import { getAuthToken } from '@/api/http';

export const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/login', component: () => import('@/views/LoginView.vue') },
    { path: '/register', component: () => import('@/views/RegisterView.vue') },
    { path: '/support', component: () => import('@/views/SupportView.vue') },
    {
      path: '/',
      component: () => import('@/views/AdminLayout.vue'),
      children: [
        { path: '', redirect: '/dashboard' },
        { path: 'dashboard', component: () => import('@/views/DashboardView.vue') },
        { path: 'statistics', component: () => import('@/views/StatisticsView.vue') },
        { path: 'order-submit', component: () => import('@/views/UserOrderSubmitView.vue') },
        { path: 'users', component: () => import('@/views/UsersView.vue') },
        { path: 'agents', component: () => import('@/views/AgentTreeView.vue') },
        { path: 'invite-codes', component: () => import('@/views/InviteCodesView.vue') },
        { path: 'categories', component: () => import('@/views/CategoriesView.vue') },
        { path: 'classes', component: () => import('@/views/ClassesView.vue') },
        { path: 'special-prices', component: () => import('@/views/SpecialPricesView.vue') },
        { path: 'orders', component: () => import('@/views/OrdersView.vue') },
        { path: 'work-orders', component: () => import('@/views/WorkOrdersView.vue') },
        { path: 'recharge-cards', component: () => import('@/views/RechargeCardsView.vue') },
        { path: 'recommendations', component: () => import('@/views/RecommendationsView.vue') },
        { path: 'menus', component: () => import('@/views/MenusView.vue') },
        { path: 'system-jobs', component: () => import('@/views/SystemJobsView.vue') },
        { path: 'connectors', component: () => import('@/views/ConnectorsView.vue') },
        { path: 'connector-29wk', component: () => import('@/views/Connector29WKView.vue') },
        { path: 'platform-runtime', component: () => import('@/views/PlatformRuntimeView.vue') },
        { path: 'settings', component: () => import('@/views/SettingsView.vue') },
        { path: 'logs', component: () => import('@/views/LogsView.vue') },
      ],
    },
    {
      path: '/user',
      component: () => import('@/views/UserLayout.vue'),
      children: [
        { path: '', redirect: '/user/dashboard' },
        { path: 'dashboard', component: () => import('@/views/UserDashboardView.vue') },
        { path: 'order-submit', component: () => import('@/views/UserOrderSubmitView.vue') },
        { path: 'classes', component: () => import('@/views/UserOrderSubmitView.vue') },
        { path: 'orders', component: () => import('@/views/UserOrdersView.vue') },
        { path: 'work-orders', component: () => import('@/views/UserWorkOrdersView.vue') },
        { path: 'recharge', component: () => import('@/views/UserRechargeView.vue') },
        { path: 'agents', component: () => import('@/views/UserAgentsView.vue') },
        { path: 'account', component: () => import('@/views/UserAccountView.vue') },
      ],
    },
  ],
});

const publicPaths = new Set(['/login', '/register', '/support']);
const guestOnlyPaths = new Set(['/login', '/register']);

router.beforeEach((to) => {
  if (!publicPaths.has(to.path) && !getAuthToken()) {
    return '/login';
  }
  if (guestOnlyPaths.has(to.path) && getAuthToken()) {
    return localStorage.getItem('dw0rdwk_role') === 'admin' ? '/dashboard' : '/user/dashboard';
  }
  if (getAuthToken() && !to.path.startsWith('/user') && localStorage.getItem('dw0rdwk_role') !== 'admin') {
    return '/user/dashboard';
  }
  return true;
});
