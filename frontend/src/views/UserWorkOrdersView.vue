<template>
  <section class="page">
    <div class="page-heading">
      <div>
        <h1>工单系统</h1>
      </div>
      <a-button type="primary" @click="openCreate">提交工单</a-button>
    </div>

    <DataToolbar v-model="search" placeholder="搜索工单" @search="load">
      <div class="data-toolbar__filters">
        <a-select v-model:value="status" allow-clear placeholder="状态" class="status-select" :options="statusOptions" @change="load" />
        <a-button @click="load">刷新</a-button>
      </div>
    </DataToolbar>

    <a-table row-key="id" :columns="columns" :data-source="rows" :loading="loading" :pagination="pagination" @change="handleTableChange">
      <template #bodyCell="{ column, record }">
        <template v-if="column.key === 'status'">
          <a-tag :color="statusColor(record.status)">{{ record.status }}</a-tag>
        </template>
        <template v-else-if="column.key === 'progress'">
          <a-progress :percent="record.progress || 0" size="small" />
        </template>
        <template v-else-if="column.key === 'action'">
          <a-space>
            <a-button size="small" @click="showDetail(record)">详情</a-button>
            <a-button size="small" @click="openReply(record)">二次回复</a-button>
            <a-popconfirm title="确定删除该工单？" @confirm="deleteRow(record.id)">
              <a-button size="small" danger>删除</a-button>
            </a-popconfirm>
          </a-space>
        </template>
      </template>
    </a-table>

    <a-modal v-model:open="createOpen" title="提交工单" @ok="submitCreate">
      <a-form layout="vertical">
        <a-form-item label="分类">
          <a-select v-model:value="form.category" :options="categoryOptions" />
        </a-form-item>
        <a-form-item label="标题">
          <a-input v-model:value="form.title" maxlength="160" />
        </a-form-item>
        <a-form-item label="内容">
          <a-textarea v-model:value="form.content" :rows="6" maxlength="5000" />
        </a-form-item>
        <a-form-item label="附件地址">
          <a-input v-model:value="form.attachmentUrl" maxlength="500" placeholder="图片或文件链接，可选" />
        </a-form-item>
      </a-form>
    </a-modal>

    <a-modal v-model:open="replyOpen" title="二次回复" @ok="submitReply">
      <a-textarea v-model:value="replyContent" :rows="6" maxlength="5000" />
    </a-modal>

    <a-modal v-model:open="detailOpen" title="工单详情" :footer="null">
      <a-descriptions v-if="current" :column="1" bordered size="small">
        <a-descriptions-item label="分类">{{ current.category }}</a-descriptions-item>
        <a-descriptions-item label="标题">{{ current.title }}</a-descriptions-item>
        <a-descriptions-item label="状态">
          <a-tag :color="statusColor(current.status)">{{ current.status }}</a-tag>
        </a-descriptions-item>
        <a-descriptions-item label="进度">{{ current.progress || 0 }}%</a-descriptions-item>
        <a-descriptions-item label="附件">
          <a v-if="current.attachmentUrl" :href="current.attachmentUrl" target="_blank" rel="noreferrer">查看附件</a>
          <span v-else>-</span>
        </a-descriptions-item>
        <a-descriptions-item label="内容">{{ current.content }}</a-descriptions-item>
        <a-descriptions-item label="回复">{{ current.answer || '-' }}</a-descriptions-item>
      </a-descriptions>
    </a-modal>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue';
import { message } from 'ant-design-vue';
import DataToolbar from '@/components/DataToolbar.vue';
import {
  createAgentWorkOrder,
  fetchAgentWorkOrders,
  removeAgentWorkOrder,
  replyAgentWorkOrder,
} from '@/api/user';
import type { WorkOrderRow } from '@/api/admin';

const loading = ref(false);
const rows = ref<WorkOrderRow[]>([]);
const search = ref('');
const status = ref<string>();
const page = ref(1);
const perPage = ref(20);
const total = ref(0);
const createOpen = ref(false);
const replyOpen = ref(false);
const detailOpen = ref(false);
const current = ref<WorkOrderRow>();
const replyContent = ref('');
const form = reactive({ category: '订单问题', title: '', content: '', attachmentUrl: '' });

const categoryOptions = [
  { label: '订单问题', value: '订单问题' },
  { label: '充值问题', value: '充值问题' },
  { label: '代理问题', value: '代理问题' },
  { label: '提出意见', value: '提出意见' },
  { label: 'bug反馈', value: 'bug反馈' },
];

const statusOptions = [
  { label: '待回复', value: '待回复' },
  { label: '已回复', value: '已回复' },
  { label: '已关闭', value: '已关闭' },
  { label: '已驳回', value: '已驳回' },
  { label: '不做处理', value: '不做处理' },
];

const columns = [
  { title: 'ID', dataIndex: 'id', width: 80 },
  { title: '分类', dataIndex: 'category', width: 120 },
  { title: '标题', dataIndex: 'title', ellipsis: true },
  { title: '状态', key: 'status', width: 120 },
  { title: '进度', key: 'progress', width: 150 },
  { title: '创建时间', dataIndex: 'createdAt', width: 190 },
  { title: '操作', key: 'action', width: 230 },
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
    const data = await fetchAgentWorkOrders({ q: search.value, status: status.value, page: page.value, perPage: perPage.value });
    rows.value = data.items;
    total.value = data.total;
  } catch (error) {
    message.error(error instanceof Error ? error.message : '工单加载失败');
  } finally {
    loading.value = false;
  }
}

function handleTableChange(next: { current?: number; pageSize?: number }) {
  page.value = next.current || 1;
  perPage.value = next.pageSize || 20;
  void load();
}

function openCreate() {
  form.category = '订单问题';
  form.title = '';
  form.content = '';
  form.attachmentUrl = '';
  createOpen.value = true;
}

async function submitCreate() {
  try {
    await createAgentWorkOrder({ category: form.category, title: form.title, content: form.content, attachmentUrl: form.attachmentUrl.trim() });
    message.success('提交成功');
    createOpen.value = false;
    await load();
  } catch (error) {
    message.error(error instanceof Error ? error.message : '提交失败');
  }
}

function openReply(row: WorkOrderRow) {
  current.value = row;
  replyContent.value = row.content;
  replyOpen.value = true;
}

async function submitReply() {
  if (!current.value) return;
  try {
    await replyAgentWorkOrder(current.value.id, replyContent.value);
    message.success('回复成功');
    replyOpen.value = false;
    await load();
  } catch (error) {
    message.error(error instanceof Error ? error.message : '回复失败');
  }
}

function showDetail(row: WorkOrderRow) {
  current.value = row;
  detailOpen.value = true;
}

async function deleteRow(id: number) {
  try {
    await removeAgentWorkOrder(id);
    message.success('删除成功');
    await load();
  } catch (error) {
    message.error(error instanceof Error ? error.message : '删除失败');
  }
}

function statusColor(value: string) {
  if (value === '已回复') return 'green';
  if (value === '已驳回') return 'red';
  if (value === '已关闭') return 'default';
  if (value === '不做处理') return 'orange';
  return 'blue';
}

onMounted(() => {
  void load();
});
</script>
