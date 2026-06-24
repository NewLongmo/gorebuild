<template>
  <section class="page">
    <div class="page-heading">
      <div>
        <h1>用户管理</h1>
        <p>管理账号、角色、余额与价格倍率。</p>
      </div>
      <a-button type="primary" @click="openCreate">新建用户</a-button>
    </div>

    <DataToolbar v-model="query.q" placeholder="搜索账号、名称或 ID" @search="reload" />
    <a-table
      row-key="id"
      :columns="columns"
      :data-source="rows"
      :loading="loading"
      :pagination="pagination"
      @change="onTableChange"
    >
      <template #bodyCell="{ column, record }">
        <a-tag v-if="column.dataIndex === 'status'" :color="record.status === 'active' ? 'green' : 'default'">
          {{ labelOf(userStatusLabels, record.status) }}
        </a-tag>
        <span v-else-if="column.dataIndex === 'role'">
          {{ labelOf(roleLabels, record.role) }}
        </span>
        <span v-else-if="column.key === 'invite'">
          {{ record.inviteCode || '-' }}
        </span>
        <span v-else-if="column.key === 'actions'" class="table-actions">
          <a-button size="small" @click="openEdit(record)">编辑</a-button>
          <a-button size="small" @click="openBalance(record)">余额</a-button>
          <a-popconfirm title="确定重置该用户密码为 1234567？" @confirm="resetPassword(record.id)">
            <a-button size="small">重置密码</a-button>
          </a-popconfirm>
          <a-popconfirm title="确定删除该用户？" @confirm="deleteRow(record.id)">
            <a-button size="small" danger>删除</a-button>
          </a-popconfirm>
        </span>
      </template>
    </a-table>

    <a-modal v-model:open="modalOpen" :title="editingId ? '编辑用户' : '新建用户'" @ok="save" @cancel="modalOpen = false">
      <a-form layout="vertical" :model="form">
        <a-form-item label="账号" required><a-input v-model:value="form.account" /></a-form-item>
        <a-form-item label="密码" :required="!editingId"><a-input-password v-model:value="form.password" /></a-form-item>
        <a-form-item label="名称"><a-input v-model:value="form.name" /></a-form-item>
        <a-form-item label="余额"><a-input-number v-model:value="form.balance" :min="0" class="full-input" /></a-form-item>
        <a-form-item label="价格倍率"><a-input-number v-model:value="form.priceRate" :min="0.0001" :step="0.01" class="full-input" /></a-form-item>
        <a-form-item label="角色">
          <a-select v-model:value="form.role" :options="roleOptions" />
        </a-form-item>
        <a-form-item label="状态">
          <a-select v-model:value="form.status" :options="statusOptions" />
        </a-form-item>
      </a-form>
    </a-modal>

    <a-modal v-model:open="balanceOpen" title="调整余额" :confirm-loading="balanceSaving" @ok="saveBalance">
      <a-form layout="vertical" :model="balanceForm">
        <a-alert
          v-if="balanceTarget"
          type="info"
          show-icon
          :message="balanceTarget.account"
          :description="`当前余额：${Number(balanceTarget.balance).toFixed(2)}`"
        />
        <a-form-item label="变动金额" required>
          <a-input-number v-model:value="balanceForm.amount" class="full-input" :step="1" />
        </a-form-item>
        <a-form-item label="备注">
          <a-input v-model:value="balanceForm.reason" />
        </a-form-item>
      </a-form>
    </a-modal>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue';
import { message } from 'ant-design-vue';
import DataToolbar from '@/components/DataToolbar.vue';
import { adjustUserBalance, createUser, fetchUsers, removeUser, resetUserPassword, updateUser, type UserRow } from '@/api/admin';
import { labelOf, roleLabels, userStatusLabels } from '@/utils/labels';

const loading = ref(false);
const rows = ref<UserRow[]>([]);
const total = ref(0);
const modalOpen = ref(false);
const balanceOpen = ref(false);
const balanceSaving = ref(false);
const balanceTarget = ref<UserRow>();
const editingId = ref<number | null>(null);
const query = reactive({ q: '', page: 1, perPage: 20 });
const form = reactive({
  account: '',
  password: '',
  name: '',
  balance: 0,
  priceRate: 1,
  role: 'agent',
  status: 'active',
});
const balanceForm = reactive({
  amount: 0,
  reason: '',
});

const roleOptions = [
  { label: roleLabels.admin, value: 'admin' },
  { label: roleLabels.agent, value: 'agent' },
];

const statusOptions = [
  { label: userStatusLabels.active, value: 'active' },
  { label: userStatusLabels.disabled, value: 'disabled' },
];

const columns = [
  { title: 'ID', dataIndex: 'id', width: 80 },
  { title: '账号', dataIndex: 'account' },
  { title: '名称', dataIndex: 'name' },
  { title: '余额', dataIndex: 'balance', width: 120 },
  { title: '倍率', dataIndex: 'priceRate', width: 100 },
  { title: '邀请码', key: 'invite', width: 130 },
  { title: '角色', dataIndex: 'role', width: 110 },
  { title: '状态', dataIndex: 'status', width: 110 },
  { title: '创建时间', dataIndex: 'createdAt', width: 180 },
  { title: '操作', key: 'actions', width: 320 },
];

const pagination = computed(() => ({
  current: query.page,
  pageSize: query.perPage,
  total: total.value,
  showSizeChanger: true,
}));

async function load() {
  loading.value = true;
  try {
    const data = await fetchUsers(query);
    rows.value = data.items;
    total.value = data.total;
  } catch (error) {
    message.error(error instanceof Error ? error.message : '用户加载失败');
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
  Object.assign(form, { account: '', password: '', name: '', balance: 0, priceRate: 1, role: 'agent', status: 'active' });
}

function openCreate() {
  editingId.value = null;
  resetForm();
  modalOpen.value = true;
}

function openEdit(row: UserRow) {
  editingId.value = row.id;
  Object.assign(form, {
    account: row.account,
    password: '',
    name: row.name,
    balance: row.balance,
    priceRate: row.priceRate,
    role: row.role,
    status: row.status,
  });
  modalOpen.value = true;
}

function openBalance(row: UserRow) {
  balanceTarget.value = row;
  Object.assign(balanceForm, { amount: 0, reason: '' });
  balanceOpen.value = true;
}

async function save() {
  const payload = {
    ...form,
    account: form.account.trim(),
    password: form.password.trim(),
    name: form.name.trim(),
  };
  if (!payload.account || (!editingId.value && !payload.password)) {
    message.error('账号和密码不能为空');
    return;
  }
  if (payload.balance < 0) {
    message.error('余额不能小于 0');
    return;
  }
  if (payload.priceRate <= 0) {
    message.error('价格倍率必须大于 0');
    return;
  }
  try {
    if (editingId.value) {
      await updateUser(editingId.value, payload);
    } else {
      await createUser(payload);
    }
    modalOpen.value = false;
    load();
  } catch (error) {
    message.error(error instanceof Error ? error.message : '用户保存失败');
  }
}

async function saveBalance() {
  if (!balanceTarget.value) return;
  if (balanceForm.amount === 0) {
    message.error('变动金额不能为 0');
    return;
  }
  balanceSaving.value = true;
  try {
    await adjustUserBalance(balanceTarget.value.id, balanceForm.amount, balanceForm.reason.trim());
    balanceOpen.value = false;
    message.success('余额已更新');
    await load();
  } catch (error) {
    message.error(error instanceof Error ? error.message : '余额更新失败');
  } finally {
    balanceSaving.value = false;
  }
}

async function deleteRow(id: number) {
  try {
    await removeUser(id);
    load();
  } catch (error) {
    message.error(error instanceof Error ? error.message : '用户删除失败');
  }
}

async function resetPassword(id: number) {
  try {
    const result = await resetUserPassword(id);
    message.success(`密码已重置为 ${result.password}`);
  } catch (error) {
    message.error(error instanceof Error ? error.message : '密码重置失败');
  }
}

onMounted(load);
</script>
