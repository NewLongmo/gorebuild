<template>
  <section class="page menus-page">
    <div class="page-heading">
      <div>
        <h1>菜单管理</h1>
        <p>维护后台左侧导航菜单，保存后管理员刷新即可看到最新结构。</p>
      </div>
      <div class="heading-actions">
        <a-button :loading="loading" @click="load">刷新</a-button>
        <a-button @click="saveSort">保存排序</a-button>
        <a-button type="primary" @click="openCreate">新增菜单</a-button>
      </div>
    </div>

    <a-table
      row-key="id"
      :columns="columns"
      :data-source="rows"
      :loading="loading"
      :pagination="false"
      :scroll="{ x: 980 }"
    >
      <template #bodyCell="{ column, record }">
        <template v-if="column.key === 'name'">
          <span class="menu-name">{{ record.name }}</span>
        </template>
        <a-tag v-else-if="column.key === 'type'" :color="record.type === 'dir' ? 'blue' : 'green'">
          {{ record.type === 'dir' ? '目录' : '菜单' }}
        </a-tag>
        <a-tag v-else-if="column.key === 'visible'" :color="record.visible ? 'green' : 'default'">
          {{ record.visible ? '显示' : '隐藏' }}
        </a-tag>
        <a-input-number
          v-else-if="column.key === 'sortOrder'"
          v-model:value="record.sortOrder"
          :min="0"
          :max="9999"
          size="small"
          class="sort-input"
        />
        <span v-else-if="column.key === 'route'" class="route-text">{{ record.route || '-' }}</span>
        <span v-else-if="column.key === 'actions'" class="table-actions">
          <a-button size="small" @click="openCreate(record.id)">新增子级</a-button>
          <a-button size="small" @click="openEdit(record)">编辑</a-button>
          <a-popconfirm title="删除前请确认该菜单没有子级。" @confirm="deleteRow(record.id)">
            <a-button size="small" danger>删除</a-button>
          </a-popconfirm>
        </span>
      </template>
    </a-table>

    <a-modal v-model:open="modalOpen" :title="editingId ? '编辑菜单' : '新增菜单'" @ok="save" @cancel="modalOpen = false">
      <a-form layout="vertical" :model="form">
        <a-form-item label="上级菜单">
          <a-select v-model:value="form.parentId" :options="parentOptions" />
        </a-form-item>
        <a-form-item label="菜单名称" required>
          <a-input v-model:value="form.name" />
        </a-form-item>
        <a-form-item label="菜单类型">
          <a-segmented v-model:value="form.type" :options="typeOptions" />
        </a-form-item>
        <a-form-item label="路由地址">
          <a-input v-model:value="form.route" placeholder="/dashboard" />
        </a-form-item>
        <a-form-item label="图标">
          <a-select v-model:value="form.icon" allow-clear :options="iconOptions" />
        </a-form-item>
        <a-form-item label="权限标识">
          <a-input v-model:value="form.permission" placeholder="admin" />
        </a-form-item>
        <div class="menu-form-grid">
          <a-form-item label="排序">
            <a-input-number v-model:value="form.sortOrder" :min="0" :max="9999" class="full-input" />
          </a-form-item>
          <a-form-item label="显示">
            <a-switch v-model:checked="form.visible" />
          </a-form-item>
        </div>
      </a-form>
    </a-modal>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue';
import { message } from 'ant-design-vue';
import { createMenu, fetchMenus, removeMenu, sortMenus, updateMenu, type AdminMenuRow } from '@/api/admin';

interface MenuForm {
  parentId: number;
  name: string;
  route: string;
  icon: string;
  type: 'dir' | 'menu';
  sortOrder: number;
  visible: boolean;
  permission: string;
}

const loading = ref(false);
const rows = ref<AdminMenuRow[]>([]);
const modalOpen = ref(false);
const editingId = ref<number | null>(null);

const form = reactive<MenuForm>({
  parentId: 0,
  name: '',
  route: '',
  icon: '',
  type: 'menu',
  sortOrder: 10,
  visible: true,
  permission: 'admin',
});

const columns = [
  { title: '菜单名称', key: 'name', dataIndex: 'name', width: 220 },
  { title: '类型', key: 'type', width: 90 },
  { title: '路由', key: 'route', width: 220 },
  { title: '图标', dataIndex: 'icon', width: 120 },
  { title: '排序', key: 'sortOrder', width: 110 },
  { title: '显示', key: 'visible', width: 90 },
  { title: '权限', dataIndex: 'permission', width: 120 },
  { title: '操作', key: 'actions', width: 220 },
];

const typeOptions = [
  { label: '菜单', value: 'menu' },
  { label: '目录', value: 'dir' },
];

const iconOptions = [
  'home',
  'settings',
  'notification',
  'question',
  'user',
  'team',
  'menu',
  'credit-card',
  'api',
  'database',
  'cluster',
  'dollar',
  'shopping-cart',
  'bar-chart',
  'file',
].map((value) => ({ label: value, value }));

const parentOptions = computed(() => [
  { label: '顶级菜单', value: 0 },
  ...flattenMenus(rows.value)
    .filter((item) => item.type === 'dir' && item.id !== editingId.value)
    .map((item) => ({ label: `${'　'.repeat(item.depth)}${item.name}`, value: item.id })),
]);

async function load() {
  loading.value = true;
  try {
    rows.value = await fetchMenus({ all: true });
  } catch (error) {
    message.error(error instanceof Error ? error.message : '菜单加载失败');
  } finally {
    loading.value = false;
  }
}

function openCreate(parentId = 0) {
  editingId.value = null;
  Object.assign(form, {
    parentId,
    name: '',
    route: '',
    icon: '',
    type: 'menu',
    sortOrder: 10,
    visible: true,
    permission: 'admin',
  });
  modalOpen.value = true;
}

function openEdit(row: AdminMenuRow) {
  editingId.value = row.id;
  Object.assign(form, {
    parentId: row.parentId,
    name: row.name,
    route: row.route,
    icon: row.icon,
    type: row.type === 'dir' ? 'dir' : 'menu',
    sortOrder: row.sortOrder,
    visible: row.visible,
    permission: row.permission || 'admin',
  });
  modalOpen.value = true;
}

async function save() {
  const payload = {
    parentId: form.parentId || 0,
    name: form.name.trim(),
    route: form.type === 'dir' ? form.route.trim() : form.route.trim(),
    icon: form.icon,
    type: form.type,
    sortOrder: form.sortOrder || 0,
    visible: form.visible,
    permission: form.permission.trim() || 'admin',
  };
  if (!payload.name) {
    message.error('菜单名称不能为空');
    return;
  }
  try {
    if (editingId.value) {
      await updateMenu(editingId.value, payload);
    } else {
      await createMenu(payload);
    }
    modalOpen.value = false;
    await load();
  } catch (error) {
    message.error(error instanceof Error ? error.message : '菜单保存失败');
  }
}

async function saveSort() {
  try {
    await sortMenus(flattenMenus(rows.value).map((item) => ({ id: item.id, parentId: item.parentId, sortOrder: item.sortOrder })));
    message.success('排序已保存');
    await load();
  } catch (error) {
    message.error(error instanceof Error ? error.message : '排序保存失败');
  }
}

async function deleteRow(id: number) {
  try {
    await removeMenu(id);
    await load();
  } catch (error) {
    message.error(error instanceof Error ? error.message : '菜单删除失败');
  }
}

function flattenMenus(items: AdminMenuRow[], depth = 0): Array<AdminMenuRow & { depth: number }> {
  const result: Array<AdminMenuRow & { depth: number }> = [];
  for (const item of items) {
    result.push({ ...item, depth });
    result.push(...flattenMenus(item.children || [], depth + 1));
  }
  return result;
}

onMounted(load);
</script>

<style scoped>
.menus-page {
  display: grid;
  gap: 16px;
}

.menu-name {
  font-weight: 600;
}

.route-text {
  font-family: "SFMono-Regular", Consolas, "Liberation Mono", monospace;
  color: #475467;
}

.sort-input {
  width: 86px;
}

.menu-form-grid {
  display: grid;
  grid-template-columns: 1fr 120px;
  gap: 14px;
  align-items: center;
}
</style>
