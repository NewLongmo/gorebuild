<template>
  <section class="page">
    <div class="page-heading">
      <div>
        <h1>邀请码</h1>
        <p>生成注册入口，控制可用次数、有效期和注册后的价格倍率。</p>
      </div>
      <a-button type="primary" @click="openCreate">生成邀请码</a-button>
    </div>

    <a-row :gutter="[16, 16]" class="page-note">
      <a-col :xs="24" :sm="12" :lg="6" v-for="card in summaryCards" :key="card.title">
        <a-card :bordered="false" class="metric-card">
          <a-statistic :title="card.title" :value="card.value" />
        </a-card>
      </a-col>
    </a-row>

    <DataToolbar v-model="query.q" placeholder="搜索邀请码、备注或 ID" @search="reload">
      <div class="data-toolbar__filters">
        <a-select
          v-model:value="query.status"
          allow-clear
          class="status-select"
          placeholder="状态"
          :options="statusOptions"
          @change="reload"
        />
        <a-button @click="load" :loading="loading">刷新</a-button>
      </div>
    </DataToolbar>

    <a-table
      row-key="id"
      :columns="columns"
      :data-source="rows"
      :loading="loading"
      :pagination="pagination"
      :scroll="{ x: 1120 }"
      @change="onTableChange"
    >
      <template #bodyCell="{ column, record }">
        <template v-if="column.key === 'code'">
          <span class="invite-code">{{ record.code }}</span>
        </template>
        <template v-else-if="column.key === 'status'">
          <a-tag :color="record.status === 'active' ? 'green' : 'red'">{{ labelOf(userStatusLabels, record.status) }}</a-tag>
        </template>
        <template v-else-if="column.key === 'usage'">
          {{ record.usedCount }} / {{ record.maxUses }}
        </template>
        <template v-else-if="column.key === 'remaining'">
          <a-tag :color="remainingUses(record) > 0 ? 'blue' : 'default'">{{ remainingUses(record) }}</a-tag>
        </template>
        <template v-else-if="column.key === 'expiresAt'">
          {{ formatDate(record.expiresAt) }}
        </template>
        <template v-else-if="column.key === 'actions'">
          <span class="table-actions">
            <a-button size="small" @click="copyCode(record.code)">复制</a-button>
            <a-button size="small" @click="openEdit(record)">编辑</a-button>
            <a-popconfirm title="确定删除该邀请码？" @confirm="deleteRow(record.id)">
              <a-button size="small" danger>删除</a-button>
            </a-popconfirm>
          </span>
        </template>
      </template>
    </a-table>

    <a-modal v-model:open="modalOpen" :title="editingId ? '编辑邀请码' : '生成邀请码'" :confirm-loading="saving" @ok="save">
      <a-form layout="vertical" :model="form">
        <a-form-item label="邀请码">
          <a-input v-model:value="form.code" :disabled="!!editingId" placeholder="留空自动生成" />
        </a-form-item>
        <a-form-item label="备注">
          <a-input v-model:value="form.note" placeholder="例如：6月推广批次" />
        </a-form-item>
        <a-form-item label="可使用次数" required>
          <a-input-number v-model:value="form.maxUses" :min="1" class="full-input" />
        </a-form-item>
        <a-form-item label="注册价格倍率" required>
          <a-input-number v-model:value="form.priceRate" :min="0.0001" :step="0.01" class="full-input" />
        </a-form-item>
        <a-form-item label="状态">
          <a-select v-model:value="form.status" :options="statusOptions" />
        </a-form-item>
        <a-form-item label="过期时间">
          <a-input v-model:value="form.expiresAt" placeholder="例如：2026-12-31T23:59:59+08:00，留空不过期" />
        </a-form-item>
      </a-form>
    </a-modal>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue';
import { message } from 'ant-design-vue';
import DataToolbar from '@/components/DataToolbar.vue';
import {
  createInviteCode,
  fetchInviteCodes,
  removeInviteCode,
  updateInviteCode,
  type InviteCodeRow,
} from '@/api/admin';
import { labelOf, userStatusLabels } from '@/utils/labels';

const loading = ref(false);
const saving = ref(false);
const rows = ref<InviteCodeRow[]>([]);
const total = ref(0);
const modalOpen = ref(false);
const editingId = ref<number | null>(null);
const query = reactive({ q: '', status: undefined as string | undefined, page: 1, perPage: 20 });
const form = reactive({
  code: '',
  note: '',
  maxUses: 1,
  priceRate: 1,
  status: 'active',
  expiresAt: '',
});

const statusOptions = [
  { label: userStatusLabels.active, value: 'active' },
  { label: userStatusLabels.disabled, value: 'disabled' },
];

const columns = [
  { title: 'ID', dataIndex: 'id', width: 80 },
  { title: '邀请码', key: 'code', dataIndex: 'code', width: 180 },
  { title: '备注', dataIndex: 'note' },
  { title: '使用次数', key: 'usage', width: 120 },
  { title: '剩余', key: 'remaining', width: 90 },
  { title: '倍率', dataIndex: 'priceRate', width: 100 },
  { title: '状态', key: 'status', width: 100 },
  { title: '过期时间', key: 'expiresAt', width: 210 },
  { title: '创建时间', dataIndex: 'createdAt', width: 190 },
  { title: '操作', key: 'actions', width: 190 },
];

const pagination = computed(() => ({
  current: query.page,
  pageSize: query.perPage,
  total: total.value,
  showSizeChanger: true,
}));

const summaryCards = computed(() => {
  const active = rows.value.filter((item) => item.status === 'active').length;
  const remaining = rows.value.reduce((sum, item) => sum + remainingUses(item), 0);
  const used = rows.value.reduce((sum, item) => sum + item.usedCount, 0);
  return [
    { title: '当前页邀请码', value: rows.value.length },
    { title: '启用中', value: active },
    { title: '剩余可用次数', value: remaining },
    { title: '已注册次数', value: used },
  ];
});

async function load() {
  loading.value = true;
  try {
    const data = await fetchInviteCodes(query);
    rows.value = data.items;
    total.value = data.total;
  } catch (error) {
    message.error(error instanceof Error ? error.message : '邀请码加载失败');
  } finally {
    loading.value = false;
  }
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

function resetForm() {
  Object.assign(form, { code: '', note: '', maxUses: 1, priceRate: 1, status: 'active', expiresAt: '' });
}

function openCreate() {
  editingId.value = null;
  resetForm();
  modalOpen.value = true;
}

function openEdit(row: InviteCodeRow) {
  editingId.value = row.id;
  Object.assign(form, {
    code: row.code,
    note: row.note,
    maxUses: row.maxUses,
    priceRate: row.priceRate,
    status: row.status,
    expiresAt: row.expiresAt || '',
  });
  modalOpen.value = true;
}

async function save() {
  if (form.maxUses < 1) {
    message.error('可使用次数至少为 1');
    return;
  }
  if (form.priceRate <= 0) {
    message.error('价格倍率必须大于 0');
    return;
  }
  saving.value = true;
  try {
    const payload = {
      code: form.code.trim(),
      note: form.note.trim(),
      maxUses: form.maxUses,
      priceRate: form.priceRate,
      status: form.status,
      expiresAt: form.expiresAt.trim(),
    };
    if (editingId.value) {
      await updateInviteCode(editingId.value, payload);
    } else {
      await createInviteCode(payload);
    }
    modalOpen.value = false;
    message.success('邀请码已保存');
    await load();
  } catch (error) {
    message.error(error instanceof Error ? error.message : '邀请码保存失败');
  } finally {
    saving.value = false;
  }
}

async function deleteRow(id: number) {
  try {
    await removeInviteCode(id);
    message.success('邀请码已删除');
    await load();
  } catch (error) {
    message.error(error instanceof Error ? error.message : '邀请码删除失败');
  }
}

function remainingUses(row: InviteCodeRow) {
  return Math.max(row.maxUses - row.usedCount, 0);
}

function formatDate(value: string | null) {
  if (!value) {
    return '不过期';
  }
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return value;
  }
  return date.toLocaleString('zh-CN', { hour12: false });
}

async function copyCode(code: string) {
  try {
    await navigator.clipboard.writeText(code);
    message.success('邀请码已复制');
  } catch {
    message.info(code);
  }
}

onMounted(load);
</script>
