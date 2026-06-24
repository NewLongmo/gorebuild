<template>
  <section class="page statistics-page">
    <div class="page-heading">
      <div>
        <h1>数据统计</h1>
        <p>查看订单、货源、用户和充值的运行数据。</p>
      </div>
      <a-button :loading="loading" @click="load">刷新</a-button>
    </div>

    <a-tabs v-model:activeKey="activeTab">
      <a-tab-pane key="data" tab="数据统计">
        <a-row :gutter="[12, 12]">
          <a-col v-for="card in summaryCards" :key="card.title" :xs="12" :md="6">
            <a-card :bordered="false" class="metric-card">
              <a-statistic :title="card.title" :value="card.value" :precision="card.precision || 0" :prefix="card.prefix" />
            </a-card>
          </a-col>
        </a-row>
        <a-card :bordered="false" class="chart-card" title="近30日趋势">
          <TrendChart :rows="stats.trend30" metric="orders" />
        </a-card>
      </a-tab-pane>

      <a-tab-pane key="sources" tab="货源统计">
        <a-table row-key="id" :columns="sourceColumns" :data-source="stats.sourceStats" :pagination="false" />
      </a-tab-pane>

      <a-tab-pane key="orders" tab="订单统计">
        <a-row :gutter="[12, 12]">
          <a-col :xs="24" :lg="12">
            <a-card :bordered="false" title="今日用户下单排行">
              <RankList :rows="stats.userOrderRank" value-label="单" />
            </a-card>
          </a-col>
          <a-col :xs="24" :lg="12">
            <a-card :bordered="false" title="今日平台下单排行">
              <RankList :rows="stats.platformRank" value-label="单" />
            </a-card>
          </a-col>
        </a-row>
      </a-tab-pane>

      <a-tab-pane key="marketing" tab="营销统计">
        <a-row :gutter="[12, 12]">
          <a-col :xs="24" :lg="12">
            <a-card :bordered="false" title="今日用户充值排行">
              <RankList :rows="stats.rechargeRank" money />
            </a-card>
          </a-col>
          <a-col :xs="24" :lg="12">
            <a-card :bordered="false" title="邀请用户排行">
              <RankList :rows="stats.inviteRank" value-label="人" />
            </a-card>
          </a-col>
        </a-row>
      </a-tab-pane>

      <a-tab-pane key="users" tab="用户统计">
        <a-row :gutter="[12, 12]">
          <a-col :xs="12" :md="6">
            <a-card :bordered="false" class="metric-card"><a-statistic title="总用户" :value="stats.summary.totalUsers" /></a-card>
          </a-col>
          <a-col :xs="12" :md="6">
            <a-card :bordered="false" class="metric-card"><a-statistic title="今日新增" :value="stats.summary.todayNewUsers" /></a-card>
          </a-col>
          <a-col :xs="12" :md="6">
            <a-card :bordered="false" class="metric-card"><a-statistic title="代理余额" :value="stats.summary.agentBalance" :precision="2" prefix="¥" /></a-card>
          </a-col>
          <a-col :xs="12" :md="6">
            <a-card :bordered="false" class="metric-card"><a-statistic title="今日充值" :value="stats.summary.todayRecharge" :precision="2" prefix="¥" /></a-card>
          </a-col>
        </a-row>
      </a-tab-pane>

      <a-tab-pane key="system" tab="系统信息">
        <a-descriptions bordered :column="{ xs: 1, md: 2 }">
          <a-descriptions-item label="上架课程">{{ stats.summary.onlineClasses }}</a-descriptions-item>
          <a-descriptions-item label="启用货源">{{ stats.summary.activeConnectors }}</a-descriptions-item>
          <a-descriptions-item label="生成时间">{{ stats.generatedAt || '-' }}</a-descriptions-item>
          <a-descriptions-item label="统计口径">订单收费汇总</a-descriptions-item>
        </a-descriptions>
      </a-tab-pane>
    </a-tabs>
  </section>
</template>

<script setup lang="ts">
import { computed, defineComponent, h, onMounted, ref } from 'vue';
import { message } from 'ant-design-vue';
import { fetchDashboardStatistics, type DashboardRankRow, type DashboardStatistics, type DashboardTrendPoint } from '@/api/admin';

const loading = ref(false);
const activeTab = ref('data');
const stats = ref<DashboardStatistics>({
  summary: {
    totalUsers: 0,
    todayNewUsers: 0,
    totalOrders: 0,
    todayOrders: 0,
    pendingOrders: 0,
    doneOrders: 0,
    failedOrders: 0,
    onlineClasses: 0,
    activeConnectors: 0,
    agentBalance: 0,
    todayRecharge: 0,
    totalRecharge: 0,
    todaySpend: 0,
    totalSpend: 0,
    todayRevenue: 0,
    totalRevenue: 0,
    todayProfit: 0,
    totalProfit: 0,
  },
  trend7: [],
  trend30: [],
  userOrderRank: [],
  platformRank: [],
  rechargeRank: [],
  inviteRank: [],
  sourceStats: [],
  generatedAt: '',
});

const summaryCards = computed(() => [
  { title: '今日订单', value: stats.value.summary.todayOrders },
  { title: '总订单', value: stats.value.summary.totalOrders },
  { title: '今日新增', value: stats.value.summary.todayNewUsers },
  { title: '总用户', value: stats.value.summary.totalUsers },
  { title: '今日充值', value: stats.value.summary.todayRecharge, precision: 2, prefix: '¥' },
  { title: '今日消费', value: stats.value.summary.todaySpend, precision: 2, prefix: '¥' },
  { title: '今日利润', value: stats.value.summary.todayProfit, precision: 2, prefix: '¥' },
  { title: '总利润', value: stats.value.summary.totalProfit, precision: 2, prefix: '¥' },
]);

const sourceColumns = [
  { title: 'ID', dataIndex: 'id', width: 80 },
  { title: '货源名称', dataIndex: 'name' },
  { title: '类型', dataIndex: 'kind', width: 120 },
  { title: '状态', dataIndex: 'status', width: 100 },
  { title: '今日订单', dataIndex: 'todayOrders', width: 120 },
  { title: '总订单', dataIndex: 'orders', width: 120 },
];

async function load() {
  loading.value = true;
  try {
    stats.value = await fetchDashboardStatistics();
  } catch (error) {
    message.error(error instanceof Error ? error.message : '统计加载失败');
  } finally {
    loading.value = false;
  }
}

const TrendChart = defineComponent({
  props: {
    rows: { type: Array<DashboardTrendPoint>, required: true },
    metric: { type: String, required: true },
  },
  setup(props) {
    return () => {
      const rows = props.rows as DashboardTrendPoint[];
      const values = rows.map((row) => Number((row as unknown as Record<string, number>)[props.metric] || 0));
      const max = Math.max(1, ...values);
      const width = 840;
      const height = 260;
      const points = values.map((value, index) => {
        const x = rows.length <= 1 ? 0 : (index / (rows.length - 1)) * width;
        const y = height - (value / max) * (height - 24) - 12;
        return `${x},${y}`;
      }).join(' ');
      return h('div', { class: 'trend-chart' }, [
        h('svg', { viewBox: `0 0 ${width} ${height}`, preserveAspectRatio: 'none' }, [
          h('polyline', { points, fill: 'none', stroke: '#1677ff', 'stroke-width': 3 }),
          ...values.map((value, index) => {
            const x = rows.length <= 1 ? 0 : (index / (rows.length - 1)) * width;
            const y = height - (value / max) * (height - 24) - 12;
            return h('circle', { cx: x, cy: y, r: 4, fill: '#1677ff' });
          }),
        ]),
        h('div', { class: 'trend-labels' }, rows.map((row) => h('span', { key: row.date }, row.date))),
      ]);
    };
  },
});

const RankList = defineComponent({
  props: {
    rows: { type: Array<DashboardRankRow>, required: true },
    money: { type: Boolean, default: false },
    valueLabel: { type: String, default: '' },
  },
  setup(props) {
    return () => {
      const rows = props.rows as DashboardRankRow[];
      if (!rows.length) {
        return h('div', { class: 'rank-empty' }, '暂无数据');
      }
      return h('ol', { class: 'rank-list' }, rows.map((row, index) => h('li', { key: `${row.id}-${row.name}-${index}` }, [
        h('span', { class: 'rank-index' }, String(index + 1)),
        h('strong', row.name || String(row.id || '-')),
        h('em', props.money ? `¥${Number(row.amount).toFixed(2)}` : `${row.count}${props.valueLabel}`),
      ])));
    };
  },
});

onMounted(load);
</script>

<style scoped>
.chart-card {
  margin-top: 12px;
  border-radius: 8px;
}

.trend-chart {
  width: 100%;
  min-height: 300px;
}

.trend-chart svg {
  width: 100%;
  height: 260px;
  border-bottom: 1px solid #edf0f3;
}

.trend-labels {
  display: flex;
  justify-content: space-between;
  gap: 4px;
  margin-top: 8px;
  color: #667085;
  font-size: 12px;
}

.rank-list {
  display: grid;
  gap: 8px;
  padding: 0;
  margin: 0;
  list-style: none;
}

.rank-list li {
  display: grid;
  grid-template-columns: 28px minmax(0, 1fr) auto;
  gap: 10px;
  align-items: center;
  padding: 8px 10px;
  border: 1px solid #edf0f3;
  border-radius: 8px;
}

.rank-index {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 22px;
  height: 22px;
  border-radius: 6px;
  background: #e6f4ff;
  color: #1677ff;
  font-weight: 700;
}

.rank-list strong {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.rank-list em {
  color: #1677ff;
  font-style: normal;
  font-weight: 700;
}

.rank-empty {
  display: grid;
  min-height: 180px;
  place-items: center;
  color: #667085;
}
</style>
