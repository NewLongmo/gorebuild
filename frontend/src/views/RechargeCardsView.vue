<template>
  <section class="page">
    <div class="page-heading">
      <div>
        <h1>卡密管理</h1>
      </div>
      <a-button type="primary" @click="openCreate">生成卡密</a-button>
    </div>

    <DataToolbar v-model="search" placeholder="搜索卡密或用户" @search="load">
      <div class="data-toolbar__filters">
        <a-select v-model:value="status" allow-clear placeholder="状态" class="status-select" :options="statusOptions" @change="load" />
        <a-button @click="load">刷新</a-button>
      </div>
    </DataToolbar>

    <a-table row-key="id" :columns="columns" :data-source="rows" :loading="loading" :pagination="pagination" @change="handleTableChange">
      <template #bodyCell="{ column, record }">
        <template v-if="column.key === 'amount'">
          {{ Number(record.amount).toFixed(2) }}
        </template>
        <template v-else-if="column.key === 'status'">
          <a-tag :color="record.status === 'used' ? 'green' : 'blue'">{{ statusLabel(record.status) }}</a-tag>
        </template>
        <template v-else-if="column.key === 'user'">
          {{ record.userAccount || record.userId || '-' }}
        </template>
        <template v-else-if="column.key === 'action'">
          <a-popconfirm title="确定删除该卡密？" @confirm="deleteRow(record.id)">
            <a-button size="small" danger>删除</a-button>
          </a-popconfirm>
        </template>
      </template>
    </a-table>

    <a-modal v-model:open="createOpen" title="生成卡密" @ok="submitCreate">
      <a-form layout="vertical">
        <a-form-item label="数量">
          <a-input-number v-model:value="form.count" :min="1" :max="100" class="full-input" />
        </a-form-item>
        <a-form-item label="金额">
          <a-input-number v-model:value="form.amount" :min="0.01" :step="1" class="full-input" />
        </a-form-item>
        <a-form-item label="指定卡密">
          <a-textarea v-model:value="form.codesText" :rows="5" placeholder="可选，一行一个；留空则随机生成" />
        </a-form-item>
      </a-form>
    </a-modal>

    <a-modal v-model:open="generatedOpen" title="已生成卡密" :footer="null">
      <a-textarea :value="generatedCodes" :rows="8" readonly />
      <div class="modal-actions">
        <a-button type="primary" @click="copyGenerated">复制</a-button>
      </div>
    </a-modal>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue';
import { message } from 'ant-design-vue';
import DataToolbar from '@/components/DataToolbar.vue';
import { createRechargeCards, fetchRechargeCards, removeRechargeCard, type RechargeCardRow } from '@/api/admin';

const loading = ref(false);
const rows = ref<RechargeCardRow[]>([]);
const search = ref('');
const status = ref<string>();
const page = ref(1);
const perPage = ref(20);
const total = ref(0);
const createOpen = ref(false);
const generatedOpen = ref(false);
const generated = ref<RechargeCardRow[]>([]);
const form = reactive({ count: 1, amount: 10, codesText: '' });

const statusOptions = [
  { label: '未使用', value: 'unused' },
  { label: '已使用', value: 'used' },
];

const columns = [
  { title: 'ID', dataIndex: 'id', width: 80 },
  { title: '卡密', dataIndex: 'code', width: 190 },
  { title: '金额', key: 'amount', width: 110 },
  { title: '状态', key: 'status', width: 110 },
  { title: '使用者', key: 'user', width: 130 },
  { title: '添加时间', dataIndex: 'createdAt', width: 190 },
  { title: '使用时间', dataIndex: 'usedAt', width: 190 },
  { title: '操作', key: 'action', width: 100 },
];

const pagination = computed(() => ({
  current: page.value,
  pageSize: perPage.value,
  total: total.value,
  showSizeChanger: true,
}));

const generatedCodes = computed(() => generated.value.map((item) => item.code).join('\n'));

async function load() {
  loading.value = true;
  try {
    const data = await fetchRechargeCards({ q: search.value, status: status.value, page: page.value, perPage: perPage.value });
    rows.value = data.items;
    total.value = data.total;
  } catch (error) {
    message.error(error instanceof Error ? error.message : '卡密加载失败');
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
  form.count = 1;
  form.amount = 10;
  form.codesText = '';
  createOpen.value = true;
}

async function submitCreate() {
  try {
    const codes = form.codesText
      .split(/\r?\n/)
      .map((item) => item.trim())
      .filter(Boolean);
    const data = await createRechargeCards(codes.length > 0 ? { amount: form.amount, codes } : { count: form.count, amount: form.amount });
    generated.value = data.items;
    createOpen.value = false;
    generatedOpen.value = true;
    message.success('生成成功');
    await load();
  } catch (error) {
    message.error(error instanceof Error ? error.message : '生成失败');
  }
}

async function deleteRow(id: number) {
  try {
    await removeRechargeCard(id);
    message.success('删除成功');
    await load();
  } catch (error) {
    message.error(error instanceof Error ? error.message : '删除失败');
  }
}

async function copyGenerated() {
  try {
    await navigator.clipboard.writeText(generatedCodes.value);
    message.success('已复制');
  } catch {
    message.error('复制失败');
  }
}

function statusLabel(value: string) {
  return value === 'used' ? '已使用' : '未使用';
}

onMounted(() => {
  void load();
});
</script>
