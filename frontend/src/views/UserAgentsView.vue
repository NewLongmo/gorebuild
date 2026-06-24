<template>
  <section class="page">
    <div class="page-heading">
      <div>
        <h1>下级代理</h1>
        <p>创建直属代理、调整状态并管理下级余额。</p>
      </div>
      <a-button type="primary" @click="openCreate">创建代理</a-button>
    </div>

    <DataToolbar v-model="search" placeholder="搜索代理" @search="load">
      <div class="data-toolbar__filters">
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
        <template v-if="column.key === 'balance'">
          {{ Number(record.balance).toFixed(2) }}
        </template>
        <template v-else-if="column.key === 'status'">
          <a-tag :color="record.status === 'active' ? 'green' : 'red'">{{ labelOf(userStatusLabels, record.status) }}</a-tag>
        </template>
        <template v-else-if="column.key === 'action'">
          <div class="table-actions">
            <a-button size="small" @click="openEdit(record)">编辑</a-button>
            <a-button size="small" @click="openTransfer(record)">调余额</a-button>
            <a-popconfirm title="确定重置该代理密码为 1234567？" @confirm="resetPassword(record.id)">
              <a-button size="small">重置密码</a-button>
            </a-popconfirm>
          </div>
        </template>
      </template>
    </a-table>

    <a-modal v-model:open="editorOpen" :title="editing ? '编辑代理' : '创建代理'" :confirm-loading="saving" @ok="saveAgent">
      <a-form layout="vertical" :model="form">
        <a-form-item label="账号" :required="!editing">
          <a-input v-model:value="form.account" :disabled="editing" autocomplete="off" />
        </a-form-item>
        <a-form-item :label="editing ? '新密码' : '密码'" :required="!editing">
          <a-input-password v-model:value="form.password" autocomplete="new-password" />
        </a-form-item>
        <a-form-item label="名称">
          <a-input v-model:value="form.name" />
        </a-form-item>
        <a-form-item label="初始余额" v-if="!editing">
          <a-input-number v-model:value="form.balance" :min="0" class="full-input" />
        </a-form-item>
        <a-form-item label="价格倍率">
          <a-input-number v-model:value="form.priceRate" :min="0.0001" :step="0.1" class="full-input" />
        </a-form-item>
        <a-form-item label="状态">
          <a-select v-model:value="form.status" :options="statusOptions" />
        </a-form-item>
      </a-form>
    </a-modal>

    <a-modal v-model:open="transferOpen" title="调整下级余额" :confirm-loading="transferring" @ok="transferBalance">
      <a-form layout="vertical">
        <a-alert v-if="selected" type="info" show-icon :message="selected.account" :description="`当前余额：${Number(selected.balance).toFixed(2)}`" />
        <a-form-item label="操作类型" required>
          <a-radio-group v-model:value="transferMode">
            <a-radio-button value="recharge">给下级充值</a-radio-button>
            <a-radio-button value="deduct">从下级扣除</a-radio-button>
          </a-radio-group>
        </a-form-item>
        <a-form-item label="金额" required>
          <a-input-number v-model:value="transferAmount" :min="0.01" :step="1" class="full-input" />
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
  createChildAgent,
  fetchChildAgents,
  resetChildAgentPassword,
  transferChildBalance,
  updateChildAgent,
} from '@/api/user';
import type { UserRow } from '@/api/admin';
import { labelOf, userStatusLabels } from '@/utils/labels';

const emit = defineEmits<{ changed: [] }>();
const loading = ref(false);
const saving = ref(false);
const transferring = ref(false);
const editorOpen = ref(false);
const transferOpen = ref(false);
const editing = ref<UserRow>();
const selected = ref<UserRow>();
const rows = ref<UserRow[]>([]);
const search = ref('');
const page = ref(1);
const perPage = ref(20);
const total = ref(0);
const transferAmount = ref(0);
const transferMode = ref<'recharge' | 'deduct'>('recharge');

const form = reactive({
  account: '',
  password: '',
  name: '',
  balance: 0,
  priceRate: 1,
  status: 'active',
});

const statusOptions = [
  { label: userStatusLabels.active, value: 'active' },
  { label: userStatusLabels.disabled, value: 'disabled' },
];

const columns = [
  { title: 'ID', dataIndex: 'id', width: 80 },
  { title: '账号', dataIndex: 'account' },
  { title: '名称', dataIndex: 'name' },
  { title: '余额', key: 'balance', width: 120 },
  { title: '倍率', dataIndex: 'priceRate', width: 100 },
  { title: '状态', key: 'status', width: 110 },
  { title: '创建时间', dataIndex: 'createdAt', width: 190 },
  { title: '操作', key: 'action', width: 260 },
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
    const data = await fetchChildAgents({ q: search.value, page: page.value, perPage: perPage.value });
    rows.value = data.items;
    total.value = data.total;
  } catch (error) {
    message.error(error instanceof Error ? error.message : '下级代理加载失败');
  } finally {
    loading.value = false;
  }
}

function handleTableChange(next: { current?: number; pageSize?: number }) {
  page.value = next.current || 1;
  perPage.value = next.pageSize || 20;
  void load();
}

function resetForm() {
  Object.assign(form, {
    account: '',
    password: '',
    name: '',
    balance: 0,
    priceRate: 1,
    status: 'active',
  });
}

function openCreate() {
  editing.value = undefined;
  resetForm();
  editorOpen.value = true;
}

function openEdit(row: UserRow) {
  editing.value = row;
  Object.assign(form, {
    account: row.account,
    password: '',
    name: row.name,
    balance: row.balance,
    priceRate: row.priceRate,
    status: row.status,
  });
  editorOpen.value = true;
}

async function saveAgent() {
  if (!editing.value && (!form.account.trim() || !form.password.trim())) {
    message.error('账号和密码不能为空');
    return;
  }
  saving.value = true;
  try {
    if (editing.value) {
      await updateChildAgent(editing.value.id, {
        password: form.password || undefined,
        name: form.name,
        priceRate: form.priceRate,
        status: form.status,
      });
    } else {
      await createChildAgent({ ...form });
      emit('changed');
    }
    editorOpen.value = false;
    message.success('代理已保存');
    await load();
  } catch (error) {
    message.error(error instanceof Error ? error.message : '代理保存失败');
  } finally {
    saving.value = false;
  }
}

function openTransfer(row: UserRow) {
  selected.value = row;
  transferAmount.value = 0;
  transferMode.value = 'recharge';
  transferOpen.value = true;
}

async function transferBalance() {
  if (!selected.value || transferAmount.value <= 0) {
    message.error('金额必须大于 0');
    return;
  }
  if (transferMode.value === 'deduct' && transferAmount.value > Number(selected.value.balance)) {
    message.error('扣除金额不能超过下级当前余额');
    return;
  }
  transferring.value = true;
  try {
    const amount = transferMode.value === 'deduct' ? -transferAmount.value : transferAmount.value;
    const result = await transferChildBalance(selected.value.id, amount);
    transferOpen.value = false;
    if (transferMode.value === 'deduct') {
      message.success('余额已扣除');
    } else {
      message.success(`已给下级充值 ${Number(result.creditedAmount).toFixed(2)}，实际扣费 ${Number(result.chargedAmount).toFixed(2)}`);
    }
    emit('changed');
    await load();
  } catch (error) {
    message.error(error instanceof Error ? error.message : '余额分配失败');
  } finally {
    transferring.value = false;
  }
}

async function resetPassword(id: number) {
  try {
    const result = await resetChildAgentPassword(id);
    message.success(`密码已重置为 ${result.password}`);
  } catch (error) {
    message.error(error instanceof Error ? error.message : '密码重置失败');
  }
}

onMounted(() => {
  void load();
});
</script>
