<template>
  <a-layout class="admin-shell">
    <a-layout-sider v-model:collapsed="app.collapsed" collapsible width="232">
      <div class="brand">
        <span class="brand__mark">G</span>
        <span v-if="!app.collapsed" class="brand__text">代理中心</span>
      </div>
      <a-menu v-model:selectedKeys="selectedKeys" theme="dark" mode="inline" :items="items" @click="go" />
    </a-layout-sider>

    <a-layout>
      <a-layout-header class="admin-header">
        <a-button type="text" class="icon-button" @click="app.toggleCollapsed">
          <MenuFoldOutlined v-if="!app.collapsed" />
          <MenuUnfoldOutlined v-else />
        </a-button>
        <div class="admin-header__meta">
          <span>{{ app.account || profile.account || 'agent' }}</span>
          <span>余额：{{ profile.balance.toFixed(2) }}</span>
          <span>倍率：{{ profile.priceRate }}</span>
          <a-button size="small" @click="signOut">退出登录</a-button>
        </div>
      </a-layout-header>

      <a-layout-content class="admin-content">
        <RouterView @changed="loadProfile" />
      </a-layout-content>
    </a-layout>
  </a-layout>
</template>

<script setup lang="ts">
import { computed, h, onMounted, reactive } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { message } from 'ant-design-vue';
import {
  AppstoreOutlined,
  CreditCardOutlined,
  PlusOutlined,
  MenuFoldOutlined,
  MenuUnfoldOutlined,
  MessageOutlined,
  ShoppingCartOutlined,
  TeamOutlined,
  UserOutlined,
} from '@ant-design/icons-vue';
import { logout } from '@/api/admin';
import { fetchAgentProfile } from '@/api/user';
import { useAppStore } from '@/stores/app';

const app = useAppStore();
const route = useRoute();
const router = useRouter();
const selectedKeys = computed(() => [route.path]);
const profile = reactive({
  account: '',
  balance: 0,
  priceRate: 1,
});

const items = [
  { key: '/user/dashboard', icon: () => h(AppstoreOutlined), label: '代理概览' },
  { key: '/user/order-submit', icon: () => h(PlusOutlined), label: '订单提交' },
  { key: '/user/orders', icon: () => h(ShoppingCartOutlined), label: '我的订单' },
  { key: '/user/work-orders', icon: () => h(MessageOutlined), label: '工单系统' },
  { key: '/user/recharge', icon: () => h(CreditCardOutlined), label: '充值卡' },
  { key: '/user/agents', icon: () => h(TeamOutlined), label: '下级代理' },
  { key: '/user/account', icon: () => h(UserOutlined), label: '账号安全' },
];

function go(event: { key: string }) {
  router.push(event.key);
}

async function signOut() {
  try {
    await logout();
  } finally {
    app.setAccount('');
    app.setRole('');
    router.replace('/login');
  }
}

async function loadProfile() {
  try {
    const data = await fetchAgentProfile();
    profile.account = data.account;
    profile.balance = data.balance;
    profile.priceRate = data.priceRate;
    app.setAccount(data.account);
    app.setRole(data.role);
  } catch (error) {
    message.error(error instanceof Error ? error.message : '会话校验失败');
  }
}

onMounted(() => {
  void loadProfile();
});
</script>
