<template>
  <section class="page">
    <div class="page-heading">
      <div>
        <h1>密价管理</h1>
        <p>为指定代理设置指定课程的专属价格规则。</p>
      </div>
      <a-button type="primary" @click="openCreate">新增密价</a-button>
    </div>

    <DataToolbar v-model="search" placeholder="搜索代理、课程或 ID" @search="reload">
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
        <template v-if="column.key === 'mode'">
          <a-tag color="blue">{{ modeText(record.mode) }}</a-tag>
        </template>
        <template v-else-if="column.key === 'price'">
          {{ Number(record.price).toFixed(4) }}
        </template>
        <template v-else-if="column.key === 'actions'">
          <span class="table-actions">
            <a-button size="small" @click="openEdit(record)">编辑</a-button>
            <a-popconfirm title="确定删除该密价？" @confirm="deleteRow(record.id)">
              <a-button size="small" danger>删除</a-button>
            </a-popconfirm>
          </span>
        </template>
      </template>
    </a-table>

    <a-modal v-model:open="modalOpen" :title="editingId ? '编辑密价' : '新增密价'" @ok="save" @cancel="modalOpen = false">
      <a-form layout="vertical" :model="form">
        <a-form-item label="用户 ID" required>
          <a-input-number v-model:value="form.userId" :min="1" class="full-input" />
        </a-form-item>
        <a-form-item label="课程 ID" required>
          <a-input-number v-model:value="form.classId" :min="1" class="full-input" />
        </a-form-item>
        <a-form-item label="模式" required>
          <a-select v-model:value="form.mode" :options="modeOptions" />
        </a-form-item>
        <a-form-item label="价格值" required>
          <a-input-number v-model:value="form.price" :min="0" :step="0.01" class="full-input" />
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
  fetchSpecialPrices,
  removeSpecialPrice,
  saveSpecialPrice,
  updateSpecialPrice,
  type SpecialPriceRow,
} from '@/api/admin';

const loading = ref(false);
const modalOpen = ref(false);
const editingId = ref<number | null>(null);
const rows = ref<SpecialPriceRow[]>([]);
const search = ref('');
const page = ref(1);
const perPage = ref(20);
const total = ref(0);
const form = reactive({
  userId: 0,
  classId: 0,
  mode: 0,
  price: 0,
});

const modeOptions = [
  { label: '用户价减价', value: 0 },
  { label: '基础价减价后乘倍率', value: 1 },
  { label: '固定价格', value: 2 },
];

const columns = [
  { title: 'ID', dataIndex: 'id', width: 80 },
  { title: '用户 ID', dataIndex: 'userId', width: 100 },
  { title: '代理账号', dataIndex: 'userAccount', width: 180 },
  { title: '课程 ID', dataIndex: 'classId', width: 100 },
  { title: '课程名称', dataIndex: 'className' },
  { title: '模式', key: 'mode', width: 190 },
  { title: '价格值', key: 'price', width: 120 },
  { title: '操作', key: 'actions', width: 150 },
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
    const data = await fetchSpecialPrices({ q: search.value, page: page.value, perPage: perPage.value });
    rows.value = data.items;
    total.value = data.total;
  } catch (error) {
    message.error(error instanceof Error ? error.message : '密价加载失败');
  } finally {
    loading.value = false;
  }
}

function reload() {
  page.value = 1;
  void load();
}

function handleTableChange(next: { current?: number; pageSize?: number }) {
  page.value = next.current || 1;
  perPage.value = next.pageSize || 20;
  void load();
}

function openCreate() {
  editingId.value = null;
  Object.assign(form, { userId: 0, classId: 0, mode: 0, price: 0 });
  modalOpen.value = true;
}

function openEdit(row: SpecialPriceRow) {
  editingId.value = row.id;
  Object.assign(form, { userId: row.userId, classId: row.classId, mode: row.mode, price: row.price });
  modalOpen.value = true;
}

async function save() {
  if (!form.userId || !form.classId) {
    message.error('用户 ID 和课程 ID 不能为空');
    return;
  }
  if (form.price < 0) {
    message.error('价格值不能小于 0');
    return;
  }
  try {
    if (editingId.value) {
      await updateSpecialPrice(editingId.value, form);
    } else {
      await saveSpecialPrice(form);
    }
    modalOpen.value = false;
    await load();
  } catch (error) {
    message.error(error instanceof Error ? error.message : '密价保存失败');
  }
}

async function deleteRow(id: number) {
  try {
    await removeSpecialPrice(id);
    await load();
  } catch (error) {
    message.error(error instanceof Error ? error.message : '密价删除失败');
  }
}

function modeText(mode: number) {
  return modeOptions.find((item) => item.value === mode)?.label || '未知';
}

onMounted(() => {
  void load();
});
</script>
