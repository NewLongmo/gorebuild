<template>
  <main class="support-page">
    <section class="support-panel">
      <div class="support-heading">
        <span class="brand__mark">G</span>
        <div>
          <h1>自助售后查询</h1>
          <p>输入下单时填写的网课账号，查看订单状态并提交补单刷新。</p>
        </div>
      </div>

      <a-card class="support-search" :bordered="false">
        <a-form layout="inline" :model="form" @finish="search">
          <a-form-item name="account" :rules="[{ required: true, message: '请输入网课账号' }]">
            <a-input v-model:value="form.account" class="support-search__input" placeholder="网课账号" allow-clear />
          </a-form-item>
          <a-form-item>
            <a-button type="primary" html-type="submit" :loading="loading">查询</a-button>
          </a-form-item>
          <a-form-item>
            <RouterLink to="/login">返回登录</RouterLink>
          </a-form-item>
        </a-form>
      </a-card>

      <a-alert
        v-if="supportNotice"
        class="support-alert"
        type="warning"
        show-icon
        :message="supportNotice"
      />

      <a-alert
        v-if="searched && rows.length === 0"
        class="support-alert"
        type="info"
        show-icon
        message="未查询到订单"
        description="请确认输入的是下单时填写的网课账号，订单刚提交后可能需要等待一段时间再查询。"
      />

      <a-table
        v-if="rows.length > 0"
        row-key="id"
        :columns="columns"
        :data-source="rows"
        :loading="loading"
        :pagination="false"
        :scroll="{ x: 980 }"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'status'">
            <a-tag :color="statusColor(record.status)">{{ statusText(record.status) }}</a-tag>
          </template>
          <template v-else-if="column.key === 'progress'">
            {{ record.progress || '-' }}
          </template>
          <template v-else-if="column.key === 'remarks'">
            {{ record.remarks || '-' }}
          </template>
          <template v-else-if="column.key === 'action'">
            <a-space>
              <a-button size="small" @click="openEvents(record)">日志</a-button>
              <a-button size="small" :disabled="isFinalizedOrder(record.status)" @click="openPasswordModal(record)">改密</a-button>
              <a-button size="small" :disabled="isFinalizedOrder(record.status)" :loading="refreshingId === record.id" @click="refresh(record.id)">补单</a-button>
              <a-button size="small" :disabled="isFinalizedOrder(record.status)" :loading="resubmittingId === record.id" @click="resubmit(record.id)">重新上号</a-button>
            </a-space>
          </template>
        </template>
        <template #expandedRowRender="{ record }">
          <div class="order-detail-grid">
            <div>
              <span>平台</span>
              <strong>{{ record.platform || '-' }}</strong>
            </div>
            <div>
              <span>学校</span>
              <strong>{{ record.school || '-' }}</strong>
            </div>
            <div>
              <span>学生姓名</span>
              <strong>{{ record.studentName || '-' }}</strong>
            </div>
            <div>
              <span>对接状态</span>
              <strong>{{ dockingText(record.dockingStatus) }}</strong>
            </div>
            <div>
              <span>分数</span>
              <strong>{{ record.score || '-' }}</strong>
            </div>
            <div>
              <span>学习时长</span>
              <strong>{{ durationText(record.durationMinutes) }}</strong>
            </div>
          </div>
        </template>
      </a-table>

      <a-drawer v-model:open="eventsOpen" width="520" title="订单执行日志">
        <a-spin :spinning="eventsLoading">
          <a-empty v-if="!eventRows.length" description="暂无执行日志" />
          <a-timeline v-else>
            <a-timeline-item v-for="item in eventRows" :key="item.id" :color="eventColor(item.level)">
              <div class="event-row">
                <strong>{{ eventTitle(item) }}</strong>
                <span>{{ formatDate(item.createdAt) }}</span>
                <p>{{ item.content || '-' }}</p>
                <code v-if="item.progress">{{ item.progress }}</code>
              </div>
            </a-timeline-item>
          </a-timeline>
        </a-spin>
      </a-drawer>

      <a-modal v-model:open="passwordOpen" title="改密重刷" :confirm-loading="savingPassword" @ok="savePassword">
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
    </section>
  </main>
</template>

<script setup lang="ts">
import { reactive, ref } from 'vue';
import { message } from 'ant-design-vue';
import {
  fetchPublicOrderEvents,
  refreshPublicOrder,
  resubmitPublicOrder,
  searchPublicOrders,
  updatePublicOrderPassword,
  type OrderEventRow,
  type PublicOrderRow,
} from '@/api/public';
import { dockingStatusLabels, labelOf, orderStatusLabels } from '@/utils/labels';

const form = reactive({ account: '' });
const loading = ref(false);
const searched = ref(false);
const refreshingId = ref<number>();
const resubmittingId = ref<number>();
const eventsOpen = ref(false);
const eventsLoading = ref(false);
const eventRows = ref<OrderEventRow[]>([]);
const eventTarget = ref<PublicOrderRow>();
const passwordOpen = ref(false);
const passwordTarget = ref<PublicOrderRow>();
const newPassword = ref('');
const savingPassword = ref(false);
const supportNotice = ref('');
const rows = ref<PublicOrderRow[]>([]);

const columns = [
  { title: '订单号', dataIndex: 'id', width: 90 },
  { title: '账号', dataIndex: 'account', width: 160 },
  { title: '课程', dataIndex: 'courseName' },
  { title: '状态', key: 'status', width: 120 },
  { title: '进度', key: 'progress', width: 140 },
  { title: '备注', key: 'remarks' },
  { title: '下单时间', dataIndex: 'createdAt', width: 180 },
  { title: '操作', key: 'action', width: 300 },
];

async function search() {
  const account = form.account.trim();
  if (!account) {
    message.error('请输入网课账号');
    return;
  }
  loading.value = true;
  try {
    const result = await searchPublicOrders(account);
    rows.value = result.items;
    supportNotice.value = result.notice || '';
    searched.value = true;
  } catch (error) {
    message.error(error instanceof Error ? error.message : '查询失败');
  } finally {
    loading.value = false;
  }
}

async function resubmit(id: number) {
  const account = form.account.trim();
  if (!account) {
    message.error('请先输入网课账号');
    return;
  }
  resubmittingId.value = id;
  try {
    await resubmitPublicOrder(id, account);
    message.success('重新上号任务已提交');
    await search();
  } catch (error) {
    message.error(error instanceof Error ? error.message : '重新上号失败');
  } finally {
    resubmittingId.value = undefined;
  }
}

async function openEvents(row: PublicOrderRow) {
  eventTarget.value = row;
  eventsOpen.value = true;
  eventsLoading.value = true;
  try {
    const result = await fetchPublicOrderEvents(row.id, form.account.trim());
    eventRows.value = result.items;
  } catch (error) {
    message.error(error instanceof Error ? error.message : '日志加载失败');
  } finally {
    eventsLoading.value = false;
  }
}

function openPasswordModal(row: PublicOrderRow) {
  passwordTarget.value = row;
  newPassword.value = '';
  passwordOpen.value = true;
}

async function savePassword() {
  if (!passwordTarget.value) return;
  const password = newPassword.value.trim();
  if (!password) {
    message.error('新密码不能为空');
    return;
  }
  savingPassword.value = true;
  try {
    await updatePublicOrderPassword(passwordTarget.value.id, form.account.trim(), password);
    message.success('密码已更新，刷新任务已入队');
    passwordOpen.value = false;
    await search();
  } catch (error) {
    message.error(error instanceof Error ? error.message : '改密失败');
  } finally {
    savingPassword.value = false;
  }
}

async function refresh(id: number) {
  const account = form.account.trim();
  if (!account) {
    message.error('请先输入网课账号');
    return;
  }
  refreshingId.value = id;
  try {
    await refreshPublicOrder(id, account);
    message.success('补单刷新已提交');
    await search();
  } catch (error) {
    message.error(error instanceof Error ? error.message : '补单提交失败');
  } finally {
    refreshingId.value = undefined;
  }
}

function statusText(status: string) {
  return labelOf(orderStatusLabels, status || 'pending');
}

function dockingText(status: string) {
  return labelOf(dockingStatusLabels, status || 'pending');
}

function statusColor(status: string) {
  if (status === 'done') return 'green';
  if (status === 'failed') return 'red';
  if (status === 'cancelled') return 'default';
  if (status === 'refunded') return 'green';
  if (status === 'processing') return 'blue';
  if (status === 'queued') return 'gold';
  return 'default';
}

function isFinalizedOrder(status: string) {
  return status === 'cancelled' || status === 'refunded';
}

function durationText(minutes: number) {
  return minutes > 0 ? `${minutes} 分钟` : '-';
}

function eventColor(level: string) {
  if (level === 'error') return 'red';
  if (level === 'warning') return 'orange';
  return 'blue';
}

function eventTitle(item: OrderEventRow) {
  return item.eventType || item.source || '执行记录';
}

function formatDate(value: string) {
  return value ? value.replace('T', ' ').replace(/\.\d+Z$/, '') : '-';
}
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
