<template>
  <section class="page">
    <div class="page-heading">
      <div>
        <h1>工单管理</h1>
      </div>
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
            <a-button size="small" type="primary" @click="openAction(record, 'answer')">回复</a-button>
            <a-button size="small" danger @click="openAction(record, 'reject')">驳回</a-button>
            <a-dropdown>
              <a-button size="small">更多</a-button>
              <template #overlay>
                <a-menu @click="handleMenuClick(record, $event)">
                  <a-menu-item key="close">关闭</a-menu-item>
                  <a-menu-item key="ignore">不处理</a-menu-item>
                  <a-menu-divider />
                  <a-menu-item key="delete" danger>删除</a-menu-item>
                </a-menu>
              </template>
            </a-dropdown>
          </a-space>
        </template>
      </template>
    </a-table>

    <a-modal v-model:open="actionOpen" :title="actionTitle" @ok="submitAction">
      <a-form layout="vertical">
        <a-form-item label="回复内容">
          <a-textarea v-model:value="answer" :rows="6" maxlength="5000" />
        </a-form-item>
        <a-form-item label="处理进度">
          <a-slider v-model:value="actionProgress" :min="0" :max="100" />
        </a-form-item>
        <a-form-item label="附件地址">
          <a-input v-model:value="actionAttachmentUrl" maxlength="500" />
        </a-form-item>
        <a-form-item label="用户可见">
          <a-switch v-model:checked="actionUserVisible" />
        </a-form-item>
      </a-form>
    </a-modal>

    <a-modal v-model:open="detailOpen" title="工单详情" :footer="null">
      <a-descriptions v-if="current" :column="1" bordered size="small">
        <a-descriptions-item label="提交者">{{ current.userAccount || current.userId }}</a-descriptions-item>
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
        <a-descriptions-item label="用户可见">{{ current.userVisible ? '是' : '否' }}</a-descriptions-item>
        <a-descriptions-item label="内容">{{ current.content }}</a-descriptions-item>
        <a-descriptions-item label="回复">{{ current.answer || '-' }}</a-descriptions-item>
      </a-descriptions>
    </a-modal>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue';
import { message, Modal } from 'ant-design-vue';
import DataToolbar from '@/components/DataToolbar.vue';
import { fetchWorkOrders, removeWorkOrder, updateWorkOrder, type WorkOrderRow } from '@/api/admin';

const loading = ref(false);
const rows = ref<WorkOrderRow[]>([]);
const search = ref('');
const status = ref<string>();
const page = ref(1);
const perPage = ref(20);
const total = ref(0);
const current = ref<WorkOrderRow>();
const action = ref('');
const answer = ref('');
const actionProgress = ref(0);
const actionAttachmentUrl = ref('');
const actionUserVisible = ref(true);
const actionOpen = ref(false);
const detailOpen = ref(false);

const statusOptions = [
  { label: '待回复', value: '待回复' },
  { label: '已回复', value: '已回复' },
  { label: '已关闭', value: '已关闭' },
  { label: '已驳回', value: '已驳回' },
  { label: '不做处理', value: '不做处理' },
];

const columns = [
  { title: 'ID', dataIndex: 'id', width: 80 },
  { title: '提交者', dataIndex: 'userAccount', width: 120 },
  { title: '分类', dataIndex: 'category', width: 120 },
  { title: '标题', dataIndex: 'title', ellipsis: true },
  { title: '状态', key: 'status', width: 120 },
  { title: '进度', key: 'progress', width: 150 },
  { title: '创建时间', dataIndex: 'createdAt', width: 190 },
  { title: '操作', key: 'action', width: 300 },
];

const pagination = computed(() => ({
  current: page.value,
  pageSize: perPage.value,
  total: total.value,
  showSizeChanger: true,
}));

const actionTitle = computed(() => (action.value === 'reject' ? '驳回工单' : '回复工单'));

async function load() {
  loading.value = true;
  try {
    const data = await fetchWorkOrders({ q: search.value, status: status.value, page: page.value, perPage: perPage.value });
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

function showDetail(row: WorkOrderRow) {
  current.value = row;
  detailOpen.value = true;
}

function openAction(row: WorkOrderRow, nextAction: string) {
  current.value = row;
  action.value = nextAction;
  answer.value = row.answer || '';
  actionProgress.value = row.progress || 0;
  actionAttachmentUrl.value = row.attachmentUrl || '';
  actionUserVisible.value = row.userVisible !== false;
  actionOpen.value = true;
}

async function submitAction() {
  if (!current.value) return;
  try {
    await updateWorkOrder(current.value.id, {
      action: action.value,
      answer: answer.value,
      progress: actionProgress.value,
      attachmentUrl: actionAttachmentUrl.value.trim(),
      userVisible: actionUserVisible.value,
    });
    message.success('操作成功');
    actionOpen.value = false;
    await load();
  } catch (error) {
    message.error(error instanceof Error ? error.message : '操作失败');
  }
}

function handleMenuClick(row: WorkOrderRow, event: { key: string | number }) {
  handleMenu(row, String(event.key));
}

function handleMenu(row: WorkOrderRow, key: string) {
  if (key === 'delete') {
    Modal.confirm({
      title: '确定删除该工单？',
      onOk: () => deleteRow(row.id),
    });
    return;
  }
  void quickAction(row.id, key);
}

async function quickAction(id: number, nextAction: string) {
  try {
    await updateWorkOrder(id, { action: nextAction });
    message.success('操作成功');
    await load();
  } catch (error) {
    message.error(error instanceof Error ? error.message : '操作失败');
  }
}

async function deleteRow(id: number) {
  try {
    await removeWorkOrder(id);
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
