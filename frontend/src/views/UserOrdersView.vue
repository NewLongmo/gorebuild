<template>
  <section class="page">
    <div class="page-heading">
      <div>
        <h1>我的订单</h1>
        <p>查看已提交订单，并发起刷新、改密或取消操作。</p>
      </div>
    </div>

    <DataToolbar v-model="search" placeholder="搜索订单" @search="load">
      <div class="data-toolbar__filters">
        <a-select v-model:value="status" allow-clear placeholder="状态" class="status-select" :options="statusOptions" @change="load" />
        <a-button @click="load">刷新</a-button>
      </div>
    </DataToolbar>

    <a-table
      row-key="id"
      :columns="columns"
      :data-source="rows"
      :loading="loading"
      :pagination="pagination"
      @change="handleTableChange"
    >
      <template #bodyCell="{ column, record }">
        <template v-if="column.key === 'status'">
          <a-tag :color="statusColor(record.status)">{{ labelOf(orderStatusLabels, record.status) }}</a-tag>
        </template>
        <template v-else-if="column.key === 'flash'">
          <a-tag v-if="record.flashMode" color="orange">极速</a-tag>
          <span v-else>普通</span>
        </template>
        <template v-else-if="column.key === 'fee'">
          {{ Number(record.fee).toFixed(2) }}
        </template>
        <template v-else-if="column.key === 'action'">
          <div class="table-actions">
            <a-button size="small" @click="openEvents(record)">日志</a-button>
            <a-button size="small" :disabled="isFinalizedOrder(record.status)" :loading="refreshingId === record.id" @click="refreshOrder(record.id)">刷新</a-button>
            <a-button size="small" :disabled="isFinalizedOrder(record.status)" @click="openPasswordModal(record)">改密</a-button>
            <a-popconfirm
              v-if="canCancel(record.status)"
              title="确定取消该订单？"
              ok-text="取消订单"
              cancel-text="再想想"
              @confirm="cancelOrder(record.id)"
            >
              <a-button size="small" danger :loading="cancellingId === record.id">取消</a-button>
            </a-popconfirm>
          </div>
        </template>
      </template>
    </a-table>

    <a-modal v-model:open="passwordModalOpen" title="改密重刷" :confirm-loading="savingPassword" @ok="saveOrderPassword">
      <a-form layout="vertical">
        <a-alert
          v-if="passwordTarget"
          type="info"
          show-icon
          :message="passwordTarget.account"
          :description="`课程：${passwordTarget.courseName}`"
        />
        <a-form-item label="新密码" required>
          <a-input-password v-model:value="newPassword" autocomplete="new-password" />
        </a-form-item>
      </a-form>
    </a-modal>

    <a-drawer v-model:open="eventsOpen" width="520" title="订单执行日志">
      <a-spin :spinning="eventsLoading">
        <a-empty v-if="!eventRows.length" description="暂无执行日志" />
        <a-timeline v-else>
          <a-timeline-item v-for="item in eventRows" :key="item.id" :color="eventColor(item.level)">
            <div class="event-row">
              <strong>{{ item.eventType || item.source }}</strong>
              <span>{{ formatDate(item.createdAt) }}</span>
              <p>{{ item.content || '-' }}</p>
              <code v-if="item.progress">{{ item.progress }}</code>
            </div>
          </a-timeline-item>
        </a-timeline>
      </a-spin>
    </a-drawer>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue';
import { message } from 'ant-design-vue';
import DataToolbar from '@/components/DataToolbar.vue';
import { cancelAgentOrder, fetchAgentOrderEvents, fetchAgentOrders, refreshAgentOrder, updateAgentOrderPassword } from '@/api/user';
import type { OrderEventRow, OrderRow } from '@/api/admin';
import { labelOf, orderStatusLabels } from '@/utils/labels';

const loading = ref(false);
const savingPassword = ref(false);
const refreshingId = ref<number>();
const cancellingId = ref<number>();
const rows = ref<OrderRow[]>([]);
const search = ref('');
const status = ref<string>();
const page = ref(1);
const perPage = ref(20);
const total = ref(0);
const passwordModalOpen = ref(false);
const passwordTarget = ref<OrderRow>();
const newPassword = ref('');
const eventsOpen = ref(false);
const eventsLoading = ref(false);
const eventRows = ref<OrderEventRow[]>([]);

const statusOptions = [
  { label: orderStatusLabels.pending, value: 'pending' },
  { label: orderStatusLabels.queued, value: 'queued' },
  { label: orderStatusLabels.processing, value: 'processing' },
  { label: orderStatusLabels.done, value: 'done' },
  { label: orderStatusLabels.failed, value: 'failed' },
  { label: orderStatusLabels.cancelled, value: 'cancelled' },
  { label: orderStatusLabels.refunded, value: 'refunded' },
];

const columns = [
  { title: 'ID', dataIndex: 'id', width: 80 },
  { title: '平台', dataIndex: 'platform', width: 160 },
  { title: '账号', dataIndex: 'account', width: 160 },
  { title: '课程', dataIndex: 'courseName' },
  { title: '费用', key: 'fee', width: 100 },
  { title: '模式', key: 'flash', width: 100 },
  { title: '状态', key: 'status', width: 120 },
  { title: '进度', dataIndex: 'progress', width: 140 },
  { title: '备注', dataIndex: 'remarks' },
  { title: '操作', key: 'action', width: 300 },
];

const pagination = computed(() => ({
  current: page.value,
  pageSize: perPage.value,
  total: total.value,
  showSizeChanger: true,
}));

async function load() {
  loading.value = true;
  try {
    const data = await fetchAgentOrders({ q: search.value, status: status.value, page: page.value, perPage: perPage.value });
    rows.value = data.items;
    total.value = data.total;
  } catch (error) {
    message.error(error instanceof Error ? error.message : '订单加载失败');
  } finally {
    loading.value = false;
  }
}

function handleTableChange(next: { current?: number; pageSize?: number }) {
  page.value = next.current || 1;
  perPage.value = next.pageSize || 20;
  void load();
}

async function refreshOrder(id: number) {
  refreshingId.value = id;
  try {
    await refreshAgentOrder(id);
    message.success('刷新任务已入队');
    await load();
  } catch (error) {
    message.error(error instanceof Error ? error.message : '订单刷新失败');
  } finally {
    refreshingId.value = undefined;
  }
}

async function openEvents(row: OrderRow) {
  eventsOpen.value = true;
  eventsLoading.value = true;
  try {
    const result = await fetchAgentOrderEvents(row.id);
    eventRows.value = result.items;
  } catch (error) {
    message.error(error instanceof Error ? error.message : '日志加载失败');
  } finally {
    eventsLoading.value = false;
  }
}

async function cancelOrder(id: number) {
  cancellingId.value = id;
  try {
    await cancelAgentOrder(id);
    message.success('订单已取消');
    await load();
  } catch (error) {
    message.error(error instanceof Error ? error.message : '订单取消失败');
  } finally {
    cancellingId.value = undefined;
  }
}

function openPasswordModal(row: OrderRow) {
  passwordTarget.value = row;
  newPassword.value = '';
  passwordModalOpen.value = true;
}

async function saveOrderPassword() {
  if (!passwordTarget.value) return;
  const password = newPassword.value.trim();
  if (!password) {
    message.error('新密码不能为空');
    return;
  }
  if (password.length > 160) {
    message.error('新密码不能超过 160 个字符');
    return;
  }
  savingPassword.value = true;
  try {
    await updateAgentOrderPassword(passwordTarget.value.id, password);
    passwordModalOpen.value = false;
    message.success('密码已更新，刷新任务已入队');
    await load();
  } catch (error) {
    message.error(error instanceof Error ? error.message : '改密失败');
  } finally {
    savingPassword.value = false;
  }
}

function canCancel(value: string) {
  return value === 'pending' || value === 'queued' || value === 'processing';
}

function isFinalizedOrder(value: string) {
  return value === 'cancelled' || value === 'refunded';
}

function statusColor(value: string) {
  if (value === 'done') return 'green';
  if (value === 'failed') return 'red';
  if (value === 'cancelled') return 'default';
  if (value === 'refunded') return 'green';
  if (value === 'processing') return 'blue';
  return 'gold';
}

function eventColor(level: string) {
  if (level === 'error') return 'red';
  if (level === 'warning') return 'orange';
  return 'blue';
}

function formatDate(value: string) {
  return value ? value.replace('T', ' ').replace(/\.\d+Z$/, '') : '-';
}

onMounted(() => {
  void load();
});
</script>

<style scoped>
.event-row {
  display: grid;
  gap: 4px;
}

.event-row span {
  color: #667085;
  font-size: 12px;
}

.event-row p {
  margin: 0;
}

.event-row code {
  color: #475467;
  white-space: pre-wrap;
}
</style>
