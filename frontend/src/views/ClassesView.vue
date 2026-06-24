<template>
  <section class="page">
    <div class="page-heading">
      <div>
        <h1>课程管理</h1>
        <p>维护课程商品、价格和对接参数。</p>
      </div>
      <div class="heading-actions">
        <a-segmented v-model:value="statusFilter" :options="statusOptions" @change="reload" />
        <a-button type="primary" @click="openCreate">新建课程</a-button>
      </div>
    </div>

    <DataToolbar v-model="query.q" placeholder="搜索课程名称、对接编码或分类" @search="reload">
      <div class="data-toolbar__filters">
        <a-select v-model:value="query.category" allow-clear placeholder="分类" class="status-select" :options="categoryOptions" @change="reload" />
        <a-button @click="reload">刷新</a-button>
      </div>
    </DataToolbar>

    <div class="bulk-toolbar">
      <span>已选 {{ selectedClassIds.length }}</span>
      <a-button :disabled="!selectedClassIds.length" @click="batchStatus('online')">批量上架</a-button>
      <a-button :disabled="!selectedClassIds.length" @click="batchStatus('offline')">批量下架</a-button>
      <a-button :disabled="!selectedClassIds.length" @click="moveOpen = true">移动分类</a-button>
      <a-button :disabled="!selectedClassIds.length" @click="patchOpen = true">批量字段</a-button>
      <a-button @click="keywordOpen = true">关键词替换</a-button>
      <a-button @click="prefixOpen = true">批量前缀</a-button>
      <a-button @click="dedupOpen = true">重复商品</a-button>
      <a-button danger :disabled="!selectedClassIds.length" @click="batchDelete">批量删除</a-button>
    </div>

    <a-table
      row-key="id"
      :columns="columns"
      :data-source="rows"
      :loading="loading"
      :pagination="pagination"
      :row-selection="rowSelection"
      @change="onTableChange"
    >
      <template #bodyCell="{ column, record }">
        <a-tag v-if="column.dataIndex === 'status'" :color="record.status === 'online' ? 'green' : 'default'">
          {{ labelOf(classStatusLabels, record.status) }}
        </a-tag>
        <span v-else-if="column.key === 'actions'" class="table-actions">
          <a-button size="small" @click="openEdit(record)">编辑</a-button>
          <a-popconfirm title="确定删除该课程？" @confirm="deleteRow(record.id)">
            <a-button size="small" danger>删除</a-button>
          </a-popconfirm>
        </span>
      </template>
    </a-table>

    <a-modal v-model:open="modalOpen" :title="editingId ? '编辑课程' : '新建课程'" @ok="save" @cancel="modalOpen = false">
      <a-form layout="vertical" :model="form">
        <a-form-item label="课程名称" required><a-input v-model:value="form.name" /></a-form-item>
        <a-form-item label="对接编码"><a-input v-model:value="form.dockingCode" /></a-form-item>
        <a-form-item label="查询参数"><a-input v-model:value="form.queryParam" /></a-form-item>
        <a-form-item label="查询接口"><a-input v-model:value="form.queryPlatform" /></a-form-item>
        <a-form-item label="对接接口"><a-input v-model:value="form.dockingPlatform" /></a-form-item>
        <a-form-item label="分类"><a-select v-model:value="form.category" allow-clear show-search :options="categoryOptions" /></a-form-item>
        <a-form-item label="价格"><a-input-number v-model:value="form.price" :min="0" :step="0.01" class="full-input" /></a-form-item>
        <a-form-item label="价格运算"><a-select v-model:value="form.priceOperator" :options="priceOperatorOptions" /></a-form-item>
        <a-form-item label="排序"><a-input-number v-model:value="form.sort" class="full-input" /></a-form-item>
        <a-form-item label="状态"><a-select v-model:value="form.status" :options="classStatusOptions" /></a-form-item>
        <a-form-item label="允许代理下单"><a-switch v-model:checked="form.bridgeEnabled" /></a-form-item>
        <a-form-item label="说明"><a-textarea v-model:value="form.description" :rows="3" maxlength="500" /></a-form-item>
      </a-form>
    </a-modal>

    <a-modal v-model:open="moveOpen" title="移动分类" @ok="submitMove">
      <a-select v-model:value="moveCategory" show-search allow-clear class="full-input" :options="categoryOptions" />
    </a-modal>

    <a-modal v-model:open="patchOpen" title="批量字段" @ok="submitPatch">
      <a-form layout="vertical">
        <a-form-item label="价格"><a-input-number v-model:value="patchForm.price" :min="0" :step="0.01" class="full-input" /></a-form-item>
        <a-form-item label="排序"><a-input-number v-model:value="patchForm.sort" class="full-input" /></a-form-item>
        <a-form-item label="查询接口"><a-input v-model:value="patchForm.queryPlatform" /></a-form-item>
        <a-form-item label="对接接口"><a-input v-model:value="patchForm.dockingPlatform" /></a-form-item>
        <a-form-item label="价格运算"><a-select v-model:value="patchForm.priceOperator" allow-clear :options="priceOperatorOptions" /></a-form-item>
        <a-form-item label="允许代理下单"><a-select v-model:value="patchForm.bridgeEnabled" allow-clear :options="bridgeOptions" /></a-form-item>
      </a-form>
    </a-modal>

    <a-modal v-model:open="keywordOpen" title="关键词替换" @ok="submitKeyword">
      <a-form layout="vertical">
        <a-form-item label="范围"><a-select v-model:value="keywordForm.scope" :options="scopeOptions" /></a-form-item>
        <a-form-item v-if="keywordForm.scope === 'category'" label="分类"><a-select v-model:value="keywordForm.scopeId" show-search :options="categoryOptions" /></a-form-item>
        <a-form-item v-if="keywordForm.scope === 'docking'" label="货源ID"><a-input v-model:value="keywordForm.scopeId" /></a-form-item>
        <a-form-item label="原关键词" required><a-input v-model:value="keywordForm.oldKeyword" /></a-form-item>
        <a-form-item label="新关键词"><a-input v-model:value="keywordForm.newKeyword" /></a-form-item>
      </a-form>
    </a-modal>

    <a-modal v-model:open="prefixOpen" title="批量前缀" @ok="submitPrefix">
      <a-form layout="vertical">
        <a-form-item label="范围"><a-select v-model:value="prefixForm.scope" :options="scopeOptions" /></a-form-item>
        <a-form-item v-if="prefixForm.scope === 'category'" label="分类"><a-select v-model:value="prefixForm.scopeId" show-search :options="categoryOptions" /></a-form-item>
        <a-form-item v-if="prefixForm.scope === 'docking'" label="货源ID"><a-input v-model:value="prefixForm.scopeId" /></a-form-item>
        <a-form-item label="前缀" required><a-input v-model:value="prefixForm.prefix" /></a-form-item>
      </a-form>
    </a-modal>

    <a-modal v-model:open="dedupOpen" title="重复商品" width="760px" @ok="applyDedup">
      <a-form layout="inline" class="dedup-form">
        <a-form-item label="范围"><a-select v-model:value="dedupForm.scope" :options="scopeOptions" class="status-select" /></a-form-item>
        <a-form-item v-if="dedupForm.scope === 'category'" label="分类"><a-select v-model:value="dedupForm.scopeId" show-search :options="categoryOptions" class="status-select" /></a-form-item>
        <a-form-item v-if="dedupForm.scope === 'docking'" label="货源ID"><a-input v-model:value="dedupForm.scopeId" class="status-select" /></a-form-item>
        <a-form-item label="保留"><a-select v-model:value="dedupForm.strategy" :options="dedupStrategyOptions" class="status-select" /></a-form-item>
        <a-button @click="previewDedup">预览</a-button>
      </a-form>
      <a-alert class="dedup-summary" type="info" :message="`将删除 ${dedupDeleteCount} 个重复商品`" />
      <a-table size="small" row-key="key" :columns="dedupColumns" :data-source="dedupGroups" :pagination="{ pageSize: 5 }" />
    </a-modal>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue';
import { message, Modal } from 'ant-design-vue';
import DataToolbar from '@/components/DataToolbar.vue';
import {
  addClassPrefix,
  applyClassDeduplicate,
  batchDeleteClasses,
  batchMoveClasses,
  batchPatchClasses,
  batchUpdateClassStatus,
  createClass,
  fetchCategories,
  fetchClasses,
  previewClassDeduplicate,
  removeClass,
  replaceClassKeywords,
  updateClass,
  type CategoryRow,
  type ClassDeduplicateGroup,
  type ClassRow,
} from '@/api/admin';
import { classStatusLabels, labelOf } from '@/utils/labels';

const loading = ref(false);
const rows = ref<ClassRow[]>([]);
const categories = ref<CategoryRow[]>([]);
const selectedClassIds = ref<number[]>([]);
const moveOpen = ref(false);
const patchOpen = ref(false);
const keywordOpen = ref(false);
const prefixOpen = ref(false);
const dedupOpen = ref(false);
const moveCategory = ref<string>();
const dedupGroups = ref<ClassDeduplicateGroup[]>([]);
const dedupDeleteCount = ref(0);
const total = ref(0);
const modalOpen = ref(false);
const editingId = ref<number | null>(null);
const statusFilter = ref('online');
const query = reactive<{ q: string; page: number; perPage: number; status?: string; category?: string }>({
  q: '',
  page: 1,
  perPage: 20,
  status: 'online',
  category: undefined,
});
const form = reactive({
  name: '',
  dockingCode: '',
  queryParam: '',
  queryPlatform: '',
  dockingPlatform: '',
  category: '',
  price: 0,
  priceOperator: '*',
  description: '',
  sort: 10,
  status: 'online',
  bridgeEnabled: true,
});
const patchForm = reactive<{ price?: number; sort?: number; queryPlatform: string; dockingPlatform: string; priceOperator?: string; bridgeEnabled?: boolean }>({
  price: undefined,
  sort: undefined,
  queryPlatform: '',
  dockingPlatform: '',
  priceOperator: undefined,
  bridgeEnabled: undefined,
});
const keywordForm = reactive({ scope: 'all', scopeId: '', oldKeyword: '', newKeyword: '' });
const prefixForm = reactive({ scope: 'all', scopeId: '', prefix: '' });
const dedupForm = reactive({ scope: 'all', scopeId: '', strategy: 'keep_older' });

const statusOptions = [
  { label: '上架', value: 'online' },
  { label: '下架', value: 'offline' },
  { label: '全部', value: 'all' },
];

const classStatusOptions = [
  { label: classStatusLabels.online, value: 'online' },
  { label: classStatusLabels.offline, value: 'offline' },
];

const priceOperatorOptions = [
  { label: '倍率', value: '*' },
  { label: '固定加价', value: '+' },
];

const scopeOptions = [
  { label: '全部', value: 'all' },
  { label: '分类', value: 'category' },
  { label: '货源', value: 'docking' },
];

const dedupStrategyOptions = [
  { label: '保留旧商品', value: 'keep_older' },
  { label: '保留新商品', value: 'keep_newer' },
];

const bridgeOptions = [
  { label: '允许', value: true },
  { label: '禁止', value: false },
];

const categoryOptions = computed(() => categories.value.map((item) => ({ label: item.name, value: item.name })));

const columns = [
  { title: 'ID', dataIndex: 'id', width: 80 },
  { title: '课程名称', dataIndex: 'name' },
  { title: '对接编码', dataIndex: 'dockingCode', width: 150 },
  { title: '分类', dataIndex: 'category', width: 120 },
  { title: '价格', dataIndex: 'price', width: 100 },
  { title: '状态', dataIndex: 'status', width: 110 },
  { title: '排序', dataIndex: 'sort', width: 88 },
  { title: '操作', key: 'actions', width: 150 },
];

const dedupColumns = [
  { title: '分类', dataIndex: 'category' },
  { title: '货源', dataIndex: 'dockingPlatform', width: 100 },
  { title: '编码', dataIndex: 'dockingCode', width: 140 },
  { title: '数量', dataIndex: 'count', width: 80 },
  { title: '保留ID', dataIndex: 'keepId', width: 90 },
];

const rowSelection = computed(() => ({
  selectedRowKeys: selectedClassIds.value,
  onChange: (keys: Array<string | number>) => {
    selectedClassIds.value = keys.map(Number).filter(Boolean);
  },
}));

const pagination = computed(() => ({
  current: query.page,
  pageSize: query.perPage,
  total: total.value,
  showSizeChanger: true,
}));

watch(statusFilter, (value) => {
  query.status = value === 'all' ? undefined : value;
});

async function load() {
  loading.value = true;
  try {
    const data = await fetchClasses(query);
    rows.value = data.items;
    total.value = data.total;
    selectedClassIds.value = selectedClassIds.value.filter((id) => data.items.some((item) => item.id === id));
  } catch (error) {
    message.error(error instanceof Error ? error.message : '课程加载失败');
  } finally {
    loading.value = false;
  }
}

async function loadCategories() {
  try {
    const data = await fetchCategories({ status: 'active', perPage: 500 });
    categories.value = data.items;
  } catch (error) {
    message.error(error instanceof Error ? error.message : '分类加载失败');
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
  Object.assign(form, {
    name: '',
    dockingCode: '',
    queryParam: '',
    queryPlatform: '',
    dockingPlatform: '',
    category: '',
    price: 0,
    priceOperator: '*',
    description: '',
    sort: 10,
    status: 'online',
    bridgeEnabled: true,
  });
}

function openCreate() {
  editingId.value = null;
  resetForm();
  modalOpen.value = true;
}

function openEdit(row: ClassRow) {
  editingId.value = row.id;
  Object.assign(form, {
    name: row.name,
    dockingCode: row.dockingCode,
    queryParam: row.queryParam,
    queryPlatform: row.queryPlatform,
    dockingPlatform: row.dockingPlatform,
    category: row.category,
    price: row.price,
    priceOperator: row.priceOperator || '*',
    description: row.description,
    sort: row.sort,
    status: row.status,
    bridgeEnabled: row.bridgeEnabled,
  });
  modalOpen.value = true;
}

async function save() {
  const payload = {
    ...form,
    name: form.name.trim(),
    dockingCode: form.dockingCode.trim(),
    queryParam: form.queryParam.trim(),
    queryPlatform: form.queryPlatform.trim(),
    dockingPlatform: form.dockingPlatform.trim(),
    category: String(form.category || '').trim(),
    priceOperator: form.priceOperator,
    description: form.description.trim(),
  };
  if (!payload.name) {
    message.error('课程名称不能为空');
    return;
  }
  if (payload.price < 0) {
    message.error('价格不能小于 0');
    return;
  }
  try {
    if (editingId.value) {
      await updateClass(editingId.value, payload);
    } else {
      await createClass(payload);
    }
    modalOpen.value = false;
    load();
  } catch (error) {
    message.error(error instanceof Error ? error.message : '课程保存失败');
  }
}

function batchStatus(status: string) {
  Modal.confirm({
    title: status === 'online' ? '批量上架课程？' : '批量下架课程？',
    onOk: async () => {
      try {
        const result = await batchUpdateClassStatus(selectedClassIds.value, status);
        message.success(`已更新 ${result.affected} 个课程`);
        await load();
      } catch (error) {
        message.error(error instanceof Error ? error.message : '批量状态更新失败');
      }
    },
  });
}

function batchDelete() {
  Modal.confirm({
    title: '确定批量删除选中课程？',
    okButtonProps: { danger: true },
    onOk: async () => {
      try {
        const result = await batchDeleteClasses(selectedClassIds.value);
        selectedClassIds.value = [];
        message.success(`已删除 ${result.affected} 个课程`);
        await load();
      } catch (error) {
        message.error(error instanceof Error ? error.message : '批量删除失败');
      }
    },
  });
}

async function submitMove() {
  try {
    const result = await batchMoveClasses(selectedClassIds.value, moveCategory.value || '');
    moveOpen.value = false;
    message.success(`已移动 ${result.affected} 个课程`);
    await load();
  } catch (error) {
    message.error(error instanceof Error ? error.message : '批量移动失败');
  }
}

async function submitPatch() {
  const update: Record<string, unknown> = {};
  if (patchForm.price !== undefined) update.price = patchForm.price;
  if (patchForm.sort !== undefined) update.sort = patchForm.sort;
  if (patchForm.queryPlatform.trim()) update.queryPlatform = patchForm.queryPlatform.trim();
  if (patchForm.dockingPlatform.trim()) update.dockingPlatform = patchForm.dockingPlatform.trim();
  if (patchForm.priceOperator) update.priceOperator = patchForm.priceOperator;
  if (patchForm.bridgeEnabled !== undefined) update.bridgeEnabled = patchForm.bridgeEnabled;
  const updates = selectedClassIds.value.map((id) => ({ id, ...update }));
  try {
    const result = await batchPatchClasses(updates);
    patchOpen.value = false;
    message.success(`已更新 ${result.updated} 个课程`);
    await load();
  } catch (error) {
    message.error(error instanceof Error ? error.message : '批量字段更新失败');
  }
}

async function submitKeyword() {
  try {
    const result = await replaceClassKeywords({ ...keywordForm });
    keywordOpen.value = false;
    message.success(`已替换 ${result.affected} 个课程`);
    await load();
  } catch (error) {
    message.error(error instanceof Error ? error.message : '关键词替换失败');
  }
}

async function submitPrefix() {
  try {
    const result = await addClassPrefix({ ...prefixForm });
    prefixOpen.value = false;
    message.success(`已处理 ${result.affected} 个课程`);
    await load();
  } catch (error) {
    message.error(error instanceof Error ? error.message : '批量前缀失败');
  }
}

async function previewDedup() {
  try {
    const result = await previewClassDeduplicate({ ...dedupForm, limit: 200 });
    dedupGroups.value = result.groups;
    dedupDeleteCount.value = result.deleteCount;
  } catch (error) {
    message.error(error instanceof Error ? error.message : '重复商品预览失败');
  }
}

async function applyDedup() {
  if (!dedupDeleteCount.value) {
    dedupOpen.value = false;
    return;
  }
  try {
    const result = await applyClassDeduplicate({ ...dedupForm, limit: 200 });
    dedupOpen.value = false;
    message.success(`已删除 ${result.affected} 个重复商品`);
    await load();
  } catch (error) {
    message.error(error instanceof Error ? error.message : '去重失败');
  }
}

async function deleteRow(id: number) {
  try {
    await removeClass(id);
    load();
  } catch (error) {
    message.error(error instanceof Error ? error.message : '课程删除失败');
  }
}

onMounted(() => {
  void loadCategories();
  void load();
});
</script>

<style scoped>
.bulk-toolbar {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  align-items: center;
  margin-bottom: 12px;
}

.dedup-form {
  margin-bottom: 12px;
}

.dedup-summary {
  margin-bottom: 12px;
}
</style>
