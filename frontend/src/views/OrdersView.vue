<template>
  <section class="page">
    <div class="page-heading">
      <div>
        <h1>订单管理</h1>
        <p>管理 MySQL 持久化订单与 Redis 提交/刷新队列。</p>
      </div>
      <div class="heading-actions">
        <a-tag v-if="autoRefreshEnabled" color="processing">自动刷新</a-tag>
        <a-tag v-if="selectedOrderIds.length" color="blue">已选 {{ selectedOrderIds.length }}</a-tag>
        <a-popconfirm title="确定批量刷新选中的订单？" @confirm="batchRefreshSelected">
          <a-button :disabled="!selectedOrderIds.length" :loading="batchRefreshing">批量刷新</a-button>
        </a-popconfirm>
        <a-popconfirm title="确定批量重新上号选中的订单？" @confirm="batchResubmitSelected">
          <a-button :disabled="!selectedOrderIds.length" :loading="batchResubmitting">批量重新上号</a-button>
        </a-popconfirm>
        <a-popconfirm title="确定批量退款并返还订单金额？" ok-text="退款" @confirm="batchRefundSelected">
          <a-button danger :disabled="!selectedOrderIds.length" :loading="batchRefunding">批量退款</a-button>
        </a-popconfirm>
        <a-popconfirm title="确定批量删除选中的订单？" ok-text="删除" @confirm="batchDeleteSelected">
          <a-button danger :disabled="!selectedOrderIds.length" :loading="batchDeleting">批量删除</a-button>
        </a-popconfirm>
        <a-button @click="openCreate(false)">新建订单</a-button>
        <a-button type="primary" @click="openCreate(true)">新建极速单</a-button>
        <a-popconfirm title="确定从 MySQL 中的排队订单重建 Redis 队列？" @confirm="recoverQueues">
          <a-button :loading="recoveringQueue">恢复队列</a-button>
        </a-popconfirm>
      </div>
    </div>

    <DataToolbar v-model="query.q" placeholder="搜索订单、账号、学校或课程" @search="reload">
      <div class="data-toolbar__filters">
        <a-select
          v-model:value="query.status"
          allow-clear
          class="status-select"
          placeholder="状态"
          :options="statusOptions"
          @change="reload"
        />
        <a-segmented
          v-model:value="query.flashMode"
          :options="modeOptions"
          @change="reload"
        />
      </div>
    </DataToolbar>

    <a-table
      row-key="id"
      :columns="columns"
      :data-source="rows"
      :loading="loading"
      :pagination="pagination"
      :row-selection="rowSelection"
      :scroll="{ x: 1320 }"
      @change="onTableChange"
    >
      <template #bodyCell="{ column, record }">
        <a-tag v-if="column.key === 'flashMode'" :color="record.flashMode ? 'gold' : 'default'">
          {{ record.flashMode ? '极速' : '普通' }}
        </a-tag>
        <a-tag v-else-if="column.dataIndex === 'status'" :color="statusColor(record.status)">
          {{ labelOf(orderStatusLabels, record.status) }}
        </a-tag>
        <a-tag v-else-if="column.dataIndex === 'dockingStatus'" :color="dockingColor(record.dockingStatus)">
          {{ labelOf(dockingStatusLabels, record.dockingStatus || 'pending') }}
        </a-tag>
        <a-tag v-else-if="column.dataIndex === 'retryCount'" :color="record.retryCount > 0 ? 'orange' : 'default'">
          {{ record.retryCount }}
        </a-tag>
        <span v-else-if="column.key === 'actions'" class="table-actions">
          <a-button size="small" @click="openEvents(record)">日志</a-button>
          <a-button size="small" @click="refreshRow(record.id)">刷新</a-button>
          <a-button size="small" @click="openEdit(record)">编辑</a-button>
          <a-popconfirm title="确定删除该订单？" @confirm="deleteRow(record.id)">
            <a-button size="small" danger>删除</a-button>
          </a-popconfirm>
          <a-popconfirm v-if="canRefund(record.status)" title="确定退款并返还订单金额？" ok-text="退款" @confirm="refundRow(record.id)">
            <a-button size="small" danger :loading="refundingId === record.id">退款</a-button>
          </a-popconfirm>
        </span>
      </template>
      <template #expandedRowRender="{ record }">
        <div class="order-detail-grid">
          <div>
            <span>远程订单号</span>
            <strong>{{ record.remoteOrderId || '-' }}</strong>
          </div>
          <div>
            <span>对接编码</span>
            <strong>{{ record.dockingCode || '-' }}</strong>
          </div>
          <div>
            <span>课程 ID</span>
            <strong>{{ record.courseId || '-' }}</strong>
          </div>
          <div>
            <span>分数</span>
            <strong>{{ record.score || '-' }}</strong>
          </div>
          <div>
            <span>时长</span>
            <strong>{{ durationText(record.durationMinutes) }}</strong>
          </div>
          <div class="order-detail-grid__wide">
            <span>备注</span>
            <strong>{{ record.remarks || '-' }}</strong>
          </div>
        </div>
      </template>
    </a-table>

    <a-modal v-model:open="modalOpen" :title="modalTitle" @ok="save" @cancel="modalOpen = false">
      <a-form layout="vertical" :model="form">
        <a-form-item label="用户 ID"><a-input-number v-model:value="form.userId" :min="0" class="full-input" /></a-form-item>
        <a-form-item label="课程 ID"><a-input-number v-model:value="form.classId" :min="0" class="full-input" /></a-form-item>
        <a-form-item label="对接通道" required>
          <a-select
            v-model:value="form.connectorId"
            :options="connectorOptions"
            :loading="connectorsLoading"
            placeholder="选择启用中的通道"
          />
        </a-form-item>
        <a-form-item label="平台"><a-input v-model:value="form.platform" /></a-form-item>
        <a-form-item label="学校"><a-input v-model:value="form.school" /></a-form-item>
        <a-form-item label="学生姓名"><a-input v-model:value="form.studentName" /></a-form-item>
        <a-form-item label="学习账号" required><a-input v-model:value="form.account" /></a-form-item>
        <a-form-item label="学习密码"><a-input-password v-model:value="form.accountPassword" /></a-form-item>
        <a-form-item label="课程名称" required><a-input v-model:value="form.courseName" /></a-form-item>
        <a-form-item label="课程标识"><a-input v-model:value="form.courseId" /></a-form-item>
        <a-form-item label="远程订单号"><a-input v-model:value="form.remoteOrderId" /></a-form-item>
        <a-form-item label="对接编码"><a-input v-model:value="form.dockingCode" /></a-form-item>
        <a-form-item label="费用"><a-input-number v-model:value="form.fee" :min="0" :step="0.01" class="full-input" /></a-form-item>
        <a-form-item label="队列模式">
          <a-segmented v-model:value="formQueueMode" :options="formModeOptions" />
        </a-form-item>
        <a-form-item label="订单状态"><a-select v-model:value="form.status" :options="statusOptions" /></a-form-item>
        <a-form-item label="对接状态"><a-select v-model:value="form.dockingStatus" :options="dockingStatusOptions" /></a-form-item>
        <a-form-item label="进度"><a-input v-model:value="form.progress" /></a-form-item>
        <a-form-item label="重试次数"><a-input-number v-model:value="form.retryCount" :min="0" class="full-input" /></a-form-item>
        <a-form-item label="分数"><a-input v-model:value="form.score" /></a-form-item>
        <a-form-item label="学习时长分钟"><a-input-number v-model:value="form.durationMinutes" :min="0" class="full-input" /></a-form-item>
        <a-form-item label="备注"><a-textarea v-model:value="form.remarks" :rows="3" /></a-form-item>
      </a-form>
    </a-modal>

    <a-drawer v-model:open="eventsOpen" width="560" title="订单执行日志">
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
import { computed, onMounted, onUnmounted, reactive, ref } from 'vue';
import { message } from 'ant-design-vue';
import DataToolbar from '@/components/DataToolbar.vue';
import {
  batchDeleteOrders,
  batchRefreshOrders,
  batchResubmitOrders,
  batchRefundOrders,
  createOrder,
  fetchConnectors,
  fetchOrderEvents,
  fetchOrders,
  fetchSettings,
  recoverOrderQueues,
  refreshOrder,
  refundOrder,
  removeOrder,
  updateOrder,
  type ConnectorRow,
  type OrderEventRow,
  type OrderRow,
} from '@/api/admin';
import { dockingStatusLabels, labelOf, orderStatusLabels } from '@/utils/labels';

const loading = ref(false);
const connectorsLoading = ref(false);
const rows = ref<OrderRow[]>([]);
const connectors = ref<ConnectorRow[]>([]);
const total = ref(0);
const modalOpen = ref(false);
const editingId = ref<number | null>(null);
const recoveringQueue = ref(false);
const batchRefreshing = ref(false);
const batchResubmitting = ref(false);
const batchRefunding = ref(false);
const batchDeleting = ref(false);
const refundingId = ref<number>();
const autoRefreshEnabled = ref(false);
const selectedOrderIds = ref<number[]>([]);
const eventsOpen = ref(false);
const eventsLoading = ref(false);
const eventRows = ref<OrderEventRow[]>([]);
let autoRefreshTimer: ReturnType<typeof window.setInterval> | undefined;
const query = reactive({
  q: '',
  status: undefined as string | undefined,
  flashMode: 'all',
  page: 1,
  perPage: 20,
});
const form = reactive({
  userId: 0,
  classId: 0,
  connectorId: undefined as number | undefined,
  platform: '',
  school: '',
  studentName: '',
  account: '',
  accountPassword: '',
  courseName: '',
  courseId: '',
  remoteOrderId: '',
  dockingCode: '',
  fee: 0,
  flashMode: false,
  status: 'queued',
  dockingStatus: 'pending',
  progress: '',
  retryCount: 0,
  remarks: '',
  score: '',
  durationMinutes: 0,
});

const statusOptions = [
  { label: orderStatusLabels.pending, value: 'pending' },
  { label: orderStatusLabels.queued, value: 'queued' },
  { label: orderStatusLabels.processing, value: 'processing' },
  { label: orderStatusLabels.done, value: 'done' },
  { label: orderStatusLabels.failed, value: 'failed' },
  { label: orderStatusLabels.cancelled, value: 'cancelled' },
  { label: orderStatusLabels.refunded, value: 'refunded' },
];

const dockingStatusOptions = [
  { label: dockingStatusLabels.pending, value: 'pending' },
  { label: dockingStatusLabels.sent, value: 'sent' },
  { label: dockingStatusLabels.refresh_requested, value: 'refresh_requested' },
  { label: dockingStatusLabels.failed, value: 'failed' },
  { label: dockingStatusLabels.queue_failed, value: 'queue_failed' },
  { label: dockingStatusLabels.cancelled, value: 'cancelled' },
  { label: dockingStatusLabels.refunded, value: 'refunded' },
];

const modeOptions = [
  { label: '全部模式', value: 'all' },
  { label: '极速', value: 'true' },
  { label: '普通', value: 'false' },
];

const formModeOptions = [
  { label: '普通队列', value: 'normal' },
  { label: '极速队列', value: 'flash' },
];

const columns = [
  { title: 'ID', dataIndex: 'id', width: 80 },
  { title: '通道', dataIndex: 'connectorId', width: 110 },
  { title: '平台', dataIndex: 'platform', width: 140 },
  { title: '学校', dataIndex: 'school', width: 140 },
  { title: '账号', dataIndex: 'account', width: 140 },
  { title: '课程', dataIndex: 'courseName' },
  { title: '费用', dataIndex: 'fee', width: 90 },
  { title: '模式', key: 'flashMode', width: 100 },
  { title: '状态', dataIndex: 'status', width: 120 },
  { title: '对接', dataIndex: 'dockingStatus', width: 140 },
  { title: '进度', dataIndex: 'progress', width: 120 },
  { title: '重试', dataIndex: 'retryCount', width: 90 },
  { title: '创建时间', dataIndex: 'createdAt', width: 180 },
  { title: '操作', key: 'actions', width: 220 },
];

const pagination = computed(() => ({
  current: query.page,
  pageSize: query.perPage,
  total: total.value,
  showSizeChanger: true,
}));

const rowSelection = computed(() => ({
  selectedRowKeys: selectedOrderIds.value,
  onChange: (keys: Array<string | number>) => {
    selectedOrderIds.value = keys.map((key) => Number(key)).filter((key) => Number.isFinite(key));
  },
}));

const connectorOptions = computed(() =>
  connectors.value.map((connector) => ({
    label: `${connector.name} #${connector.id}`,
    value: connector.id,
  })),
);

const modalTitle = computed(() => {
  if (editingId.value) {
    return '编辑订单';
  }
  return form.flashMode ? '新建极速单' : '新建订单';
});

const formQueueMode = computed({
  get: () => (form.flashMode ? 'flash' : 'normal'),
  set: (value: string) => {
    form.flashMode = value === 'flash';
  },
});

const orderQuery = computed(() => {
  const params: Record<string, unknown> = {
    q: query.q,
    status: query.status,
    page: query.page,
    perPage: query.perPage,
  };
  if (query.flashMode !== 'all') {
    params.flashMode = query.flashMode;
  }
  return params;
});

async function load() {
  loading.value = true;
  try {
    const data = await fetchOrders(orderQuery.value);
    rows.value = data.items;
    total.value = data.total;
    const visibleIds = new Set(data.items.map((row) => row.id));
    selectedOrderIds.value = selectedOrderIds.value.filter((id) => visibleIds.has(id));
  } catch (error) {
    message.error(error instanceof Error ? error.message : '订单加载失败');
  } finally {
    loading.value = false;
  }
}

async function loadConnectors() {
  connectorsLoading.value = true;
  try {
    const data = await fetchConnectors({ status: 'active', page: 1, perPage: 100 });
    connectors.value = data.items;
  } catch (error) {
    message.error(error instanceof Error ? error.message : '通道加载失败');
  } finally {
    connectorsLoading.value = false;
  }
}

async function loadPageSettings() {
  try {
    const settings = await fetchSettings();
    autoRefreshEnabled.value = settings.order_auto_refresh === 'true';
    syncAutoRefreshTimer();
  } catch (error) {
    message.error(error instanceof Error ? error.message : '页面设置加载失败');
  }
}

function syncAutoRefreshTimer() {
  if (autoRefreshTimer) {
    window.clearInterval(autoRefreshTimer);
    autoRefreshTimer = undefined;
  }
  if (!autoRefreshEnabled.value) {
    return;
  }
  autoRefreshTimer = window.setInterval(() => {
    if (modalOpen.value || loading.value || recoveringQueue.value || batchRefreshing.value || batchResubmitting.value || batchRefunding.value || batchDeleting.value) {
      return;
    }
    load();
  }, 15000);
}

function reload() {
  query.page = 1;
  load();
}

function onTableChange(next: { current?: number; pageSize?: number }) {
  query.page = next.current || 1;
  query.perPage = next.pageSize || 20;
  load();
}

function resetForm(flashMode = false) {
  Object.assign(form, {
    userId: 0,
    classId: 0,
    connectorId: connectors.value[0]?.id,
    platform: '',
    school: '',
    studentName: '',
    account: '',
    accountPassword: '',
    courseName: '',
    courseId: '',
    remoteOrderId: '',
    dockingCode: '',
    fee: 0,
    flashMode,
    status: 'queued',
    dockingStatus: 'pending',
    progress: '',
    retryCount: 0,
    remarks: '',
    score: '',
    durationMinutes: 0,
  });
}

function openCreate(flashMode = false) {
  editingId.value = null;
  resetForm(flashMode);
  modalOpen.value = true;
}

function openEdit(row: OrderRow) {
  editingId.value = row.id;
  Object.assign(form, {
    userId: row.userId,
    classId: row.classId,
    connectorId: row.connectorId,
    platform: row.platform,
    school: row.school,
    studentName: row.studentName,
    account: row.account,
    accountPassword: '',
    courseName: row.courseName,
    courseId: row.courseId,
    remoteOrderId: row.remoteOrderId,
    dockingCode: row.dockingCode,
    fee: row.fee,
    flashMode: row.flashMode,
    status: row.status,
    dockingStatus: row.dockingStatus || 'pending',
    progress: row.progress,
    retryCount: row.retryCount,
    remarks: row.remarks,
    score: row.score,
    durationMinutes: row.durationMinutes,
  });
  modalOpen.value = true;
}

function statusColor(status: string) {
  const colors: Record<string, string> = {
    pending: 'default',
    queued: 'blue',
    processing: 'processing',
    done: 'green',
    failed: 'red',
    cancelled: 'default',
    refunded: 'green',
  };
  return colors[status] || 'default';
}

function dockingColor(status: string) {
  const colors: Record<string, string> = {
    pending: 'default',
    sent: 'blue',
    refresh_requested: 'purple',
    failed: 'red',
    queue_failed: 'red',
    cancelled: 'default',
    refunded: 'green',
  };
  return colors[status] || 'default';
}

function durationText(minutes: number) {
  if (!minutes) {
    return '-';
  }
  return `${minutes} 分钟`;
}

async function save() {
  const payload = {
    ...form,
    platform: form.platform.trim(),
    school: form.school.trim(),
    studentName: form.studentName.trim(),
    account: form.account.trim(),
    accountPassword: form.accountPassword.trim(),
    courseName: form.courseName.trim(),
    courseId: form.courseId.trim(),
    remoteOrderId: form.remoteOrderId.trim(),
    dockingCode: form.dockingCode.trim(),
    dockingStatus: form.dockingStatus.trim(),
    progress: form.progress.trim(),
    remarks: form.remarks.trim(),
    score: form.score.trim(),
  };
  if (!payload.account || !payload.courseName) {
    message.error('学习账号和课程名称不能为空');
    return;
  }
  if (!payload.connectorId) {
    message.error('请选择对接通道');
    return;
  }
  if (payload.fee < 0) {
    message.error('费用不能小于 0');
    return;
  }
  if (payload.retryCount < 0 || payload.durationMinutes < 0) {
    message.error('重试次数和学习时长不能小于 0');
    return;
  }
  try {
    if (editingId.value) {
      await updateOrder(editingId.value, payload);
    } else {
      await createOrder(payload);
    }
    modalOpen.value = false;
    load();
  } catch (error) {
    message.error(error instanceof Error ? error.message : '订单保存失败');
  }
}

async function deleteRow(id: number) {
  try {
    await removeOrder(id);
    load();
  } catch (error) {
    message.error(error instanceof Error ? error.message : '订单删除失败');
  }
}

async function refreshRow(id: number) {
  try {
    await refreshOrder(id);
    load();
  } catch (error) {
    message.error(error instanceof Error ? error.message : '订单刷新失败');
  }
}

async function openEvents(row: OrderRow) {
  eventsOpen.value = true;
  eventsLoading.value = true;
  try {
    const result = await fetchOrderEvents(row.id);
    eventRows.value = result.items;
  } catch (error) {
    message.error(error instanceof Error ? error.message : '日志加载失败');
  } finally {
    eventsLoading.value = false;
  }
}

async function refundRow(id: number) {
  refundingId.value = id;
  try {
    await refundOrder(id);
    message.success('订单已退款，金额已返还');
    load();
  } catch (error) {
    message.error(error instanceof Error ? error.message : '订单退款失败');
  } finally {
    refundingId.value = undefined;
  }
}

async function batchRefreshSelected() {
  if (!selectedOrderIds.value.length) {
    return;
  }
  batchRefreshing.value = true;
  try {
    const result = await batchRefreshOrders(selectedOrderIds.value);
    message.success(batchResultText('已请求刷新', result.requested || 0, result));
    selectedOrderIds.value = [];
    load();
  } catch (error) {
    message.error(error instanceof Error ? error.message : '批量刷新失败');
  } finally {
    batchRefreshing.value = false;
  }
}

async function batchResubmitSelected() {
  if (!selectedOrderIds.value.length) {
    return;
  }
  batchResubmitting.value = true;
  try {
    const result = await batchResubmitOrders(selectedOrderIds.value);
    message.success(batchResultText('已重新上号', result.requeued || 0, result));
    selectedOrderIds.value = [];
    load();
  } catch (error) {
    message.error(error instanceof Error ? error.message : '批量重新上号失败');
  } finally {
    batchResubmitting.value = false;
  }
}

async function batchRefundSelected() {
  if (!selectedOrderIds.value.length) {
    return;
  }
  batchRefunding.value = true;
  try {
    const result = await batchRefundOrders(selectedOrderIds.value);
    message.success(batchResultText('已退款', result.refunded || 0, result));
    selectedOrderIds.value = [];
    load();
  } catch (error) {
    message.error(error instanceof Error ? error.message : '批量退款失败');
  } finally {
    batchRefunding.value = false;
  }
}

async function batchDeleteSelected() {
  if (!selectedOrderIds.value.length) {
    return;
  }
  batchDeleting.value = true;
  try {
    const result = await batchDeleteOrders(selectedOrderIds.value);
    message.success(batchResultText('已删除', result.deleted || 0, result));
    selectedOrderIds.value = [];
    load();
  } catch (error) {
    message.error(error instanceof Error ? error.message : '批量删除失败');
  } finally {
    batchDeleting.value = false;
  }
}

function batchResultText(label: string, count: number, result: { skipped: number; failed: number }) {
  return `${label} ${count} 个，跳过 ${result.skipped} 个，失败 ${result.failed} 个`;
}

function canRefund(status: string) {
  return status !== 'refunded';
}

function eventColor(level: string) {
  if (level === 'error') return 'red';
  if (level === 'warning') return 'orange';
  return 'blue';
}

function formatDate(value: string) {
  return value ? value.replace('T', ' ').replace(/\.\d+Z$/, '') : '-';
}

async function recoverQueues() {
  recoveringQueue.value = true;
  try {
    const result = await recoverOrderQueues();
    message.success(`已恢复 ${result.recovered} 个排队订单`);
    load();
  } catch (error) {
    message.error(error instanceof Error ? error.message : '队列恢复失败');
  } finally {
    recoveringQueue.value = false;
  }
}

onMounted(() => {
  loadConnectors();
  load();
  loadPageSettings();
});

onUnmounted(() => {
  if (autoRefreshTimer) {
    window.clearInterval(autoRefreshTimer);
  }
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
