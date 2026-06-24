<template>
  <section class="page">
    <div class="page-heading">
      <div>
        <h1>推荐下单</h1>
        <p>配置用户下单页展示的推荐平台和说明。</p>
      </div>
      <a-button type="primary" @click="openCreate">新增推荐</a-button>
    </div>

    <DataToolbar v-model="query.q" placeholder="搜索推荐或平台" @search="load">
      <div class="data-toolbar__filters">
        <a-switch v-model:checked="includeHidden" checked-children="含隐藏" un-checked-children="仅显示" @change="load" />
        <a-button @click="load">刷新</a-button>
      </div>
    </DataToolbar>

    <a-table
      row-key="id"
      :columns="columns"
      :data-source="filteredRows"
      :loading="loading"
      :pagination="pagination"
      @change="handleTableChange"
    >
      <template #bodyCell="{ column, record }">
        <template v-if="column.key === 'visible'">
          <a-tag :color="record.visible ? 'green' : 'default'">{{ record.visible ? '显示' : '隐藏' }}</a-tag>
        </template>
        <template v-else-if="column.key === 'class'">
          <strong>{{ record.className || `#${record.classId}` }}</strong>
          <div class="muted">{{ record.classCategory || '未分类' }} · {{ Number(record.classPrice || 0).toFixed(2) }}</div>
        </template>
        <template v-else-if="column.key === 'note'">
          <span class="note-cell">{{ record.note || '-' }}</span>
        </template>
        <template v-else-if="column.key === 'actions'">
          <div class="table-actions">
            <a-button size="small" @click="openEdit(record)">编辑</a-button>
            <a-popconfirm title="确定删除这条推荐？" @confirm="deleteRow(record.id)">
              <a-button size="small" danger>删除</a-button>
            </a-popconfirm>
          </div>
        </template>
      </template>
    </a-table>

    <a-modal v-model:open="modalOpen" :title="editingId ? '编辑推荐' : '新增推荐'" :confirm-loading="saving" @ok="save">
      <a-form layout="vertical">
        <a-form-item label="推荐平台" required>
          <a-select
            v-model:value="form.classId"
            show-search
            :filter-option="false"
            :options="classOptions"
            :loading="classesLoading"
            placeholder="搜索并选择平台"
            @search="searchClasses"
          />
        </a-form-item>
        <a-form-item label="推荐标题">
          <a-input v-model:value="form.title" maxlength="160" placeholder="留空时使用平台名称" />
        </a-form-item>
        <a-form-item label="推荐说明">
          <a-textarea v-model:value="form.note" :rows="4" maxlength="5000" />
        </a-form-item>
        <a-form-item label="排序">
          <a-input-number v-model:value="form.sortOrder" :min="0" :max="100000" class="full-input" />
        </a-form-item>
        <a-form-item label="显示状态">
          <a-switch v-model:checked="form.visible" />
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
  createRecommendation,
  fetchClasses,
  fetchRecommendations,
  removeRecommendation,
  updateRecommendation,
  type ClassRow,
  type RecommendationRow,
} from '@/api/admin';

const loading = ref(false);
const saving = ref(false);
const classesLoading = ref(false);
const modalOpen = ref(false);
const includeHidden = ref(true);
const rows = ref<RecommendationRow[]>([]);
const classRows = ref<ClassRow[]>([]);
const total = ref(0);
const editingId = ref<number | null>(null);
const query = reactive({ q: '', page: 1, perPage: 20 });
const form = reactive({
  classId: undefined as number | undefined,
  title: '',
  note: '',
  sortOrder: 10,
  visible: true,
});

const columns = [
  { title: 'ID', dataIndex: 'id', width: 80 },
  { title: '排序', dataIndex: 'sortOrder', width: 90 },
  { title: '标题', dataIndex: 'title', width: 180 },
  { title: '平台', key: 'class' },
  { title: '说明', key: 'note' },
  { title: '状态', key: 'visible', width: 100 },
  { title: '操作', key: 'actions', width: 140 },
];

const pagination = computed(() => ({
  current: query.page,
  pageSize: query.perPage,
  total: total.value,
  showSizeChanger: true,
}));

const filteredRows = computed(() => {
  const keyword = query.q.trim().toLowerCase();
  if (!keyword) return rows.value;
  return rows.value.filter((item) => [item.title, item.note, item.className, item.classCategory].some((value) => String(value || '').toLowerCase().includes(keyword)));
});

const classOptions = computed(() =>
  classRows.value.map((item) => ({
    label: `${item.name} · ${item.category || '未分类'} · ${Number(item.price || 0).toFixed(2)}`,
    value: item.id,
  })),
);

async function load() {
  loading.value = true;
  try {
    const data = await fetchRecommendations({
      includeHidden: includeHidden.value,
      page: query.page,
      perPage: query.perPage,
    });
    rows.value = data.items;
    total.value = data.total;
  } catch (error) {
    message.error(error instanceof Error ? error.message : '推荐加载失败');
  } finally {
    loading.value = false;
  }
}

async function loadClasses(keyword = '') {
  classesLoading.value = true;
  try {
    const data = await fetchClasses({ q: keyword, status: 'online', page: 1, perPage: 100 });
    classRows.value = data.items;
  } catch (error) {
    message.error(error instanceof Error ? error.message : '平台加载失败');
  } finally {
    classesLoading.value = false;
  }
}

function searchClasses(value: string) {
  void loadClasses(value);
}

function handleTableChange(next: { current?: number; pageSize?: number }) {
  query.page = next.current || 1;
  query.perPage = next.pageSize || 20;
  void load();
}

function resetForm() {
  Object.assign(form, { classId: undefined, title: '', note: '', sortOrder: 10, visible: true });
}

function openCreate() {
  editingId.value = null;
  resetForm();
  modalOpen.value = true;
  void loadClasses();
}

function openEdit(row: RecommendationRow) {
  editingId.value = row.id;
  Object.assign(form, {
    classId: row.classId,
    title: row.title,
    note: row.note,
    sortOrder: row.sortOrder,
    visible: row.visible,
  });
  if (!classRows.value.some((item) => item.id === row.classId)) {
    classRows.value.unshift({
      id: row.classId,
      name: row.className,
      category: row.classCategory,
      price: row.classPrice,
      status: row.classStatus,
      sort: 0,
      dockingCode: '',
      queryParam: '',
      queryPlatform: row.classQuery,
      dockingPlatform: row.classDocking,
      priceOperator: '*',
      description: row.classDescription,
      bridgeEnabled: row.classBridgeEnabled,
    });
  }
  modalOpen.value = true;
}

async function save() {
  if (!form.classId) {
    message.error('请选择推荐平台');
    return;
  }
  if (form.sortOrder < 0 || form.sortOrder > 100000) {
    message.error('排序必须在 0 到 100000 之间');
    return;
  }
  saving.value = true;
  try {
    const payload = {
      classId: form.classId,
      title: form.title.trim(),
      note: form.note.trim(),
      sortOrder: form.sortOrder,
      visible: form.visible,
    };
    if (editingId.value) {
      await updateRecommendation(editingId.value, payload);
    } else {
      await createRecommendation(payload);
    }
    modalOpen.value = false;
    message.success('推荐已保存');
    await load();
  } catch (error) {
    message.error(error instanceof Error ? error.message : '推荐保存失败');
  } finally {
    saving.value = false;
  }
}

async function deleteRow(id: number) {
  try {
    await removeRecommendation(id);
    message.success('推荐已删除');
    await load();
  } catch (error) {
    message.error(error instanceof Error ? error.message : '删除失败');
  }
}

onMounted(() => {
  void load();
  void loadClasses();
});
</script>

<style scoped>
.muted {
  color: #667085;
  font-size: 12px;
}

.note-cell {
  display: inline-block;
  max-width: 460px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
</style>
