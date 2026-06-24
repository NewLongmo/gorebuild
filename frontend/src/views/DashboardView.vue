<template>
  <section class="page">
    <div class="page-heading">
      <div>
        <h1>控制台</h1>
        <p>查看订单、队列、用户和课程的实时运行概览。</p>
      </div>
      <a-button :loading="loading" @click="load">刷新</a-button>
    </div>

    <a-row :gutter="[16, 16]">
      <a-col :xs="24" :sm="12" :lg="6" v-for="card in cards" :key="card.title">
        <a-card :bordered="false" class="metric-card">
          <a-statistic :title="card.title" :value="card.value" />
        </a-card>
      </a-col>
    </a-row>

    <a-alert
      class="page-note"
      type="info"
      show-icon
      :message="cacheMessage"
    />
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue';
import { message } from 'ant-design-vue';
import { fetchDashboard, fetchSettings, type DashboardStats } from '@/api/admin';

const loading = ref(false);
const cacheSeconds = ref(30);
const stats = ref<DashboardStats>({
  users: 0,
  classes: 0,
  orders: 0,
  pending: 0,
  flashOrders: 0,
  flashPending: 0,
  queueOrders: 0,
  queueRefreshes: 0,
  queueSubmit: 0,
  queueSubmitFlash: 0,
  queueRefresh: 0,
  queueRefreshFlash: 0,
  activeUsers: 0,
  onlineClasses: 0,
});

const cards = computed(() => [
  { title: '用户总数', value: stats.value.users },
  { title: '启用用户', value: stats.value.activeUsers },
  { title: '课程总数', value: stats.value.classes },
  { title: '订单总数', value: stats.value.orders },
  { title: '待处理订单', value: stats.value.pending },
  { title: '极速订单', value: stats.value.flashOrders },
  { title: '待处理极速单', value: stats.value.flashPending },
  { title: '提交队列', value: stats.value.queueOrders },
  { title: '刷新队列', value: stats.value.queueRefreshes },
  { title: '普通提交队列', value: stats.value.queueSubmit },
  { title: '极速提交队列', value: stats.value.queueSubmitFlash },
  { title: '普通刷新队列', value: stats.value.queueRefresh },
  { title: '极速刷新队列', value: stats.value.queueRefreshFlash },
  { title: '上架课程', value: stats.value.onlineClasses },
]);

const cacheMessage = computed(
  () => `Redis 会缓存控制台指标 ${cacheSeconds.value} 秒；列表数据仍直接读取 MySQL，保证页面信息新鲜。`,
);

async function load() {
  loading.value = true;
  try {
    const [dashboard, settings] = await Promise.all([fetchDashboard(), fetchSettings()]);
    stats.value = dashboard;
    cacheSeconds.value = normalizeCacheSeconds(settings.dashboard_cache_seconds);
  } catch (error) {
    message.error(error instanceof Error ? error.message : '控制台加载失败');
  } finally {
    loading.value = false;
  }
}

function normalizeCacheSeconds(value: string | undefined) {
  const parsed = Number(value || 30);
  if (!Number.isFinite(parsed)) {
    return 30;
  }
  return Math.min(Math.max(Math.trunc(parsed), 5), 3600);
}

onMounted(load);
</script>
