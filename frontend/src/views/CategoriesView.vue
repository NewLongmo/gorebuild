<template>
  <section class="page">
    <div class="page-heading">
      <div>
        <h1>分类管理</h1>
      </div>
      <div class="heading-actions">
        <a-segmented v-model:value="statusFilter" :options="statusOptions" @change="reload" />
        <a-button type="primary" @click="openCreate">新增分类</a-button>
      </div>
    </div>

    <DataToolbar v-model="search" placeholder="搜索分类" @search="reload" />

    <a-table row-key="id" :columns="columns" :data-source="rows" :loading="loading" :pagination="pagination" @change="handleTableChange">
      <template #bodyCell="{ column, record }">
        <template v-if="column.key === 'status'">
          <a-tag :color="record.status === 'active' ? 'green' : 'default'">{{ statusLabel(record.status) }}</a-tag>
        </template>
        <template v-else-if="column.key === 'pinned'">
          <a-tag :color="record.pinned ? 'blue' : 'default'">{{ record.pinned ? '置顶' : '普通' }}</a-tag>
        </template>
        <template v-else-if="column.key === 'action'">
          <a-space>
            <a-button size="small" @click="openEdit(record)">编辑</a-button>
            <a-popconfirm title="确定删除该分类？会同时删除该分类下的平台/课程、收藏和密价配置。" @confirm="deleteRow(record.id)">
              <a-button size="small" danger>删除</a-button>
            </a-popconfirm>
          </a-space>
        </template>
      </template>
    </a-table>

    <a-modal v-model:open="modalOpen" :title="editingId ? '编辑分类' : '新增分类'" @ok="save">
      <a-form layout="vertical">
        <a-form-item label="分类名称" required>
          <a-input v-model:value="form.name" maxlength="120" />
        </a-form-item>
        <a-form-item label="排序">
          <a-input-number v-model:value="form.sort" class="full-input" />
        </a-form-item>
        <a-form-item label="状态">
          <a-select v-model:value="form.status" :options="categoryStatusOptions" />
        </a-form-item>
        <a-form-item label="置顶">
          <a-switch v-model:checked="form.pinned" />
        </a-form-item>
        <a-form-item label="说明">
          <a-textarea v-model:value="form.description" :rows="3" maxlength="2000" />
        </a-form-item>
      </a-form>
    </a-modal>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue';
import { message } from 'ant-design-vue';
import DataToolbar from '@/components/DataToolbar.vue';
import { createCategory, fetchCategories, removeCategory, updateCategory, type CascadeDeleteResult, type CategoryRow } from '@/api/admin';

const loading = ref(false);
const modalOpen = ref(false);
const rows = ref<CategoryRow[]>([]);
const search = ref('');
const statusFilter = ref('active');
const page = ref(1);
const perPage = ref(20);
const total = ref(0);
const editingId = ref<number>();
const form = reactive({ name: '', sort: 10, status: 'active', pinned: false, description: '' });

const statusOptions = [
  { label: '启用', value: 'active' },
  { label: '停用', value: 'disabled' },
  { label: '全部', value: 'all' },
];

const categoryStatusOptions = [
  { label: '启用', value: 'active' },
  { label: '停用', value: 'disabled' },
];

const columns = [
  { title: 'ID', dataIndex: 'id', width: 80 },
  { title: '分类名称', dataIndex: 'name' },
  { title: '置顶', key: 'pinned', width: 100 },
  { title: '状态', key: 'status', width: 110 },
  { title: '排序', dataIndex: 'sort', width: 100 },
  { title: '创建时间', dataIndex: 'createdAt', width: 190 },
  { title: '操作', key: 'action', width: 150 },
];

const pagination = computed(() => ({
  current: page.value,
  pageSize: perPage.value,
  total: total.value,
  showSizeChanger: true,
}));

watch(statusFilter, () => reload());

async function load() {
  loading.value = true;
  try {
    const status = statusFilter.value === 'all' ? undefined : statusFilter.value;
    const data = await fetchCategories({ q: search.value, status, page: page.value, perPage: perPage.value });
    rows.value = data.items;
    total.value = data.total;
  } catch (error) {
    message.error(error instanceof Error ? error.message : '分类加载失败');
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
  editingId.value = undefined;
  Object.assign(form, { name: '', sort: 10, status: 'active', pinned: false, description: '' });
  modalOpen.value = true;
}

function openEdit(row: CategoryRow) {
  editingId.value = row.id;
  Object.assign(form, { name: row.name, sort: row.sort, status: row.status, pinned: row.pinned, description: row.description });
  modalOpen.value = true;
}

async function save() {
  const payload = { name: form.name.trim(), sort: form.sort, status: form.status, pinned: form.pinned, description: form.description.trim() };
  if (!payload.name) {
    message.error('分类名称不能为空');
    return;
  }
  try {
    if (editingId.value) {
      await updateCategory(editingId.value, payload);
    } else {
      await createCategory(payload);
    }
    modalOpen.value = false;
    await load();
  } catch (error) {
    message.error(error instanceof Error ? error.message : '分类保存失败');
  }
}

async function deleteRow(id: number) {
  try {
    const result = await removeCategory(id);
    message.success(cascadeDeleteText('分类已删除', result));
    await load();
  } catch (error) {
    message.error(error instanceof Error ? error.message : '分类删除失败');
  }
}

function cascadeDeleteText(prefix: string, result: CascadeDeleteResult) {
  return `${prefix}：平台/课程 ${result.deletedClasses || 0}，收藏 ${result.deletedFavorites || 0}，密价 ${result.deletedSpecialPrices || 0}`;
}

function statusLabel(value: string) {
  return value === 'active' ? '启用' : '停用';
}

onMounted(() => {
  void load();
});
</script>
