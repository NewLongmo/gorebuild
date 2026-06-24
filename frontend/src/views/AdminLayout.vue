<template>
  <a-layout class="admin-shell">
    <a-layout-sider v-model:collapsed="app.collapsed" collapsible width="232">
      <div class="brand">
        <span class="brand__mark">G</span>
        <span v-if="!app.collapsed" class="brand__text">DW0RDWK</span>
      </div>
      <a-menu v-model:selectedKeys="selectedKeys" theme="dark" mode="inline" :items="menuItems" @click="go" />
    </a-layout-sider>

    <a-layout>
      <a-layout-header class="admin-header">
        <a-button type="text" class="icon-button" @click="app.toggleCollapsed">
          <MenuFoldOutlined v-if="!app.collapsed" />
          <MenuUnfoldOutlined v-else />
        </a-button>
        <div class="admin-header__meta">
          <span>{{ app.account || 'admin' }}</span>
          <span>{{ app.role === 'admin' ? '管理员' : app.role || '角色' }}</span>
          <span>Fiber API</span>
          <a-badge status="processing" text="Redis + MySQL" />
          <a-button size="small" @click="signOut">退出登录</a-button>
        </div>
      </a-layout-header>

      <a-layout-content class="admin-content">
        <RouterView />
      </a-layout-content>
    </a-layout>
  </a-layout>
</template>

<script setup lang="ts">
import { computed, h, onMounted, ref } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { message } from 'ant-design-vue';
import {
  AppstoreOutlined,
  ApiOutlined,
  BarChartOutlined,
  ClusterOutlined,
  CreditCardOutlined,
  DatabaseOutlined,
  DollarOutlined,
  FileTextOutlined,
  GiftOutlined,
  HomeOutlined,
  MenuFoldOutlined,
  MenuOutlined,
  MenuUnfoldOutlined,
  MessageOutlined,
  MonitorOutlined,
  NotificationOutlined,
  QuestionCircleOutlined,
  SettingOutlined,
  ShoppingCartOutlined,
  StarOutlined,
  TeamOutlined,
  UserOutlined,
} from '@ant-design/icons-vue';
import { useAppStore } from '@/stores/app';
import { fetchMe, fetchMenus, logout, type AdminMenuRow } from '@/api/admin';

const app = useAppStore();
const route = useRoute();
const router = useRouter();

const selectedKeys = computed(() => [route.path]);

interface AdminMenuItem {
  key: string;
  icon?: () => ReturnType<typeof h>;
  label: string;
  disabled?: boolean;
  children?: AdminMenuItem[];
}

const iconMap = {
  api: ApiOutlined,
  app: AppstoreOutlined,
  'bar-chart': BarChartOutlined,
  cluster: ClusterOutlined,
  'credit-card': CreditCardOutlined,
  database: DatabaseOutlined,
  dollar: DollarOutlined,
  file: FileTextOutlined,
  gift: GiftOutlined,
  home: HomeOutlined,
  menu: MenuOutlined,
  message: MessageOutlined,
  monitor: MonitorOutlined,
  notification: NotificationOutlined,
  plugin: ApiOutlined,
  question: QuestionCircleOutlined,
  settings: SettingOutlined,
  'shopping-cart': ShoppingCartOutlined,
  star: StarOutlined,
  team: TeamOutlined,
  user: UserOutlined,
};

const menuItems = ref<AdminMenuItem[]>(fallbackMenuItems());

function fallbackMenuItems(): AdminMenuItem[] {
  return [
    { key: '/dashboard', icon: renderIcon('home'), label: '首页' },
    {
      key: 'system-management',
      icon: renderIcon('settings'),
      label: '系统管理',
      children: [
        { key: '/settings', icon: renderIcon('settings'), label: '系统配置' },
        { key: '/support', icon: renderIcon('question'), label: '常见问答' },
        { key: '/users', icon: renderIcon('user'), label: '用户管理' },
        { key: '/agents', icon: renderIcon('team'), label: '角色管理' },
        { key: '/menus', icon: renderIcon('menu'), label: '菜单管理' },
        { key: '/recharge-cards', icon: renderIcon('credit-card'), label: '卡密管理' },
        { key: '/system-jobs', icon: renderIcon('monitor'), label: '运行监控' },
        { key: '/logs', icon: renderIcon('file'), label: '操作日志' },
      ],
    },
    {
      key: 'course-management',
      icon: renderIcon('database'),
      label: '网课管理',
      children: [
        { key: '/connectors', icon: renderIcon('api'), label: '货源管理' },
        { key: '/platform-runtime', icon: renderIcon('plugin'), label: '直跑插件' },
        { key: '/categories', icon: renderIcon('database'), label: '分类管理' },
        { key: '/classes', icon: renderIcon('cluster'), label: '平台管理' },
        { key: '/special-prices', icon: renderIcon('dollar'), label: '密价管理' },
        { key: '/orders', icon: renderIcon('shopping-cart'), label: '任务管理' },
        { key: '/recommendations', icon: renderIcon('star'), label: '推荐下单' },
        { key: '/dashboard', icon: renderIcon('bar-chart'), label: '数据统计' },
      ],
    },
  ];
}

function go(event: { key: string }) {
  if (event.key.startsWith('/')) {
    router.push(event.key);
  }
}

function renderIcon(name: string) {
  const component = iconMap[name as keyof typeof iconMap] || AppstoreOutlined;
  return () => h(component);
}

function menuItemFromNode(node: AdminMenuRow): AdminMenuItem | null {
  if (!node.visible) {
    return null;
  }
  const children = (node.children || [])
    .map((child) => menuItemFromNode(child))
    .filter((child): child is AdminMenuItem => Boolean(child));
  const key = node.route || `menu-${node.id}`;
  const item: AdminMenuItem = {
    key,
    icon: renderIcon(node.icon),
    label: node.name,
    disabled: !node.route && children.length === 0,
  };
  if (children.length > 0) {
    item.children = children;
  }
  return item;
}

async function loadMenus() {
  try {
    const data = await fetchMenus();
    const loaded = data.map((item) => menuItemFromNode(item)).filter((item): item is AdminMenuItem => Boolean(item));
    if (loaded.length > 0) {
      menuItems.value = loaded;
    }
  } catch {
    menuItems.value = fallbackMenuItems();
  }
}

async function signOut() {
  try {
    await logout();
  } finally {
    clearSession();
  }
}

function clearSession() {
  app.setAccount('');
  app.setRole('');
  router.replace('/login');
}

async function syncSession() {
  try {
    const me = await fetchMe();
    if (me.role !== 'admin') {
      await logout().catch(() => undefined);
      clearSession();
      message.error('当前账号不是管理员');
      return;
    }
    app.setAccount(me.account);
    app.setRole(me.role);
  } catch (error) {
    if (router.currentRoute.value.path !== '/login') {
      message.error(error instanceof Error ? error.message : '会话校验失败');
    }
  }
}

onMounted(() => {
  void loadMenus();
  void syncSession();
});
</script>
