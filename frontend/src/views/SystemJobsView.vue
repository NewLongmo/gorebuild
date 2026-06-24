<template>
  <section class="page">
    <div class="page-heading">
      <div>
        <h1>运行监控</h1>
        <p>查看同步任务和队列恢复的最近运行状态。</p>
      </div>
      <a-button :loading="loading" @click="load">刷新</a-button>
    </div>

    <a-table row-key="name" :columns="columns" :data-source="rows" :loading="loading" :pagination="false">
      <template #bodyCell="{ column, record }">
        <template v-if="column.key === 'status'">
          <a-tag :color="statusColor(record.status)">{{ statusText(record.status) }}</a-tag>
        </template>
        <template v-else-if="column.key === 'enabled'">
          <a-switch
            :checked="record.enabled"
            checked-children="启用"
            un-checked-children="暂停"
            :loading="togglingName === record.name"
            @change="toggleJob(record)"
          />
        </template>
        <template v-else-if="column.key === 'summary'">
          <pre>{{ summaryText(record.lastSummaryJson) }}</pre>
        </template>
        <template v-else-if="column.key === 'error'">
          <span :class="{ danger: record.lastError }">{{ record.lastError || '-' }}</span>
        </template>
        <template v-else-if="column.key === 'actions'">
          <a-button size="small" type="primary" :loading="runningName === record.name" @click="runJob(record.name)">手动执行</a-button>
        </template>
      </template>
    </a-table>
  </section>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue';
import { message } from 'ant-design-vue';
import { fetchSystemJobs, runSystemJob, updateSystemJob, type SystemJobRow } from '@/api/admin';

const loading = ref(false);
const runningName = ref('');
const togglingName = ref('');
const rows = ref<SystemJobRow[]>([]);

const columns = [
  { title: '任务', dataIndex: 'name', width: 180 },
  { title: '状态', key: 'status', width: 110 },
  { title: '开关', key: 'enabled', width: 120 },
  { title: '上次开始', dataIndex: 'lastStartedAt', width: 190 },
  { title: '上次结束', dataIndex: 'lastFinishedAt', width: 190 },
  { title: '耗时', dataIndex: 'lastDurationMs', width: 100 },
  { title: '摘要', key: 'summary' },
  { title: '错误', key: 'error' },
  { title: '操作', key: 'actions', width: 120 },
];

async function load() {
  loading.value = true;
  try {
    const data = await fetchSystemJobs();
    rows.value = ensureKnownJobs(data.items);
  } catch (error) {
    message.error(error instanceof Error ? error.message : '运行状态加载失败');
  } finally {
    loading.value = false;
  }
}

function ensureKnownJobs(items: SystemJobRow[]) {
  const known = ['29wk_order_sync', '29wk_price_sync', 'order_queue_recover'];
  const map = new Map(items.map((item) => [item.name, item]));
  return known.map((name) => map.get(name) || emptyJob(name));
}

function emptyJob(name: string): SystemJobRow {
  return {
    name,
    status: 'idle',
    enabled: true,
    lastStartedAt: null,
    lastFinishedAt: null,
    lastDurationMs: 0,
    lastError: '',
    lastSummaryJson: '{}',
    heartbeatAt: null,
  };
}

async function toggleJob(row: SystemJobRow) {
  togglingName.value = row.name;
  try {
    await updateSystemJob(row.name, !row.enabled);
    message.success(!row.enabled ? '任务已启用' : '任务已暂停');
    await load();
  } catch (error) {
    message.error(error instanceof Error ? error.message : '任务状态更新失败');
  } finally {
    togglingName.value = '';
  }
}

async function runJob(name: string) {
  runningName.value = name;
  try {
    await runSystemJob(name);
    message.success('任务执行完成');
    await load();
  } catch (error) {
    message.error(error instanceof Error ? error.message : '任务执行失败');
    await load();
  } finally {
    runningName.value = '';
  }
}

function statusColor(status: string) {
  if (status === 'success') return 'green';
  if (status === 'failed') return 'red';
  if (status === 'running') return 'processing';
  return 'default';
}

function statusText(status: string) {
  const labels: Record<string, string> = {
    idle: '待运行',
    running: '运行中',
    success: '成功',
    failed: '失败',
  };
  return labels[status] || status || '待运行';
}

function summaryText(raw: string) {
  if (!raw) return '-';
  try {
    return JSON.stringify(JSON.parse(raw), null, 2);
  } catch {
    return raw;
  }
}

onMounted(load);
</script>

<style scoped>
pre {
  max-width: 360px;
  max-height: 160px;
  margin: 0;
  overflow: auto;
  white-space: pre-wrap;
}

.danger {
  color: #cf1322;
}
</style>
