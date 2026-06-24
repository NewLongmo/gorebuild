<template>
  <section class="page">
    <div class="page-heading">
      <div>
        <h1>课程列表</h1>
        <p>浏览可下单课程并提交学习订单。</p>
      </div>
    </div>

    <DataToolbar v-model="search" placeholder="搜索课程" @search="load">
      <div class="data-toolbar__filters">
        <a-select v-model:value="category" allow-clear placeholder="分类" class="status-select" :options="categoryOptions" @change="reload" />
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
        <template v-if="column.key === 'price'">
          {{ Number(record.userPrice).toFixed(2) }}
        </template>
        <template v-else-if="column.key === 'status'">
          <a-tag color="green">{{ labelOf(classStatusLabels, record.status) }}</a-tag>
        </template>
        <template v-else-if="column.key === 'action'">
          <a-button type="primary" size="small" :disabled="!record.bridgeEnabled" @click="openSubmit(record)">下单</a-button>
        </template>
      </template>
    </a-table>

    <a-modal v-model:open="submitOpen" title="提交订单" :confirm-loading="submitting" @ok="submitOrder">
      <a-form layout="vertical" :model="form">
        <a-alert v-if="selectedClass" type="info" show-icon :message="selectedClass.name" :description="`费用：${Number(selectedClass.userPrice).toFixed(2)}`" />
        <a-form-item label="学校">
          <a-input v-model:value="form.school" />
        </a-form-item>
        <a-form-item label="学生姓名">
          <a-input v-model:value="form.studentName" />
        </a-form-item>
        <a-form-item label="学习账号" required>
          <a-input v-model:value="form.account" />
        </a-form-item>
        <a-form-item label="学习密码">
          <a-input-password v-model:value="form.accountPassword" />
        </a-form-item>
        <a-form-item label="课程 ID">
          <a-input v-model:value="form.courseId" />
        </a-form-item>
        <a-form-item label="课程名称" required>
          <a-input v-model:value="form.courseName" />
        </a-form-item>
        <a-form-item label="学习时长分钟">
          <a-input-number v-model:value="form.durationMinutes" :min="0" class="full-input" />
        </a-form-item>
        <a-form-item label="极速模式">
          <a-switch v-model:checked="form.flashMode" />
        </a-form-item>
      </a-form>
    </a-modal>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue';
import { message } from 'ant-design-vue';
import DataToolbar from '@/components/DataToolbar.vue';
import { createAgentOrder, fetchAgentCategories, fetchAgentClasses, type AgentClassRow } from '@/api/user';
import type { CategoryRow } from '@/api/admin';
import { classStatusLabels, labelOf } from '@/utils/labels';

const emit = defineEmits<{ changed: [] }>();
const loading = ref(false);
const submitting = ref(false);
const submitOpen = ref(false);
const rows = ref<AgentClassRow[]>([]);
const categories = ref<CategoryRow[]>([]);
const selectedClass = ref<AgentClassRow>();
const search = ref('');
const category = ref<string>();
const page = ref(1);
const perPage = ref(20);
const total = ref(0);

const form = reactive({
  school: '',
  studentName: '',
  account: '',
  accountPassword: '',
  courseId: '',
  courseName: '',
  durationMinutes: 0,
  flashMode: false,
});

const columns = [
  { title: '课程名称', dataIndex: 'name' },
  { title: '分类', dataIndex: 'category', width: 140 },
  { title: '价格', key: 'price', width: 120 },
  { title: '状态', key: 'status', width: 100 },
  { title: '操作', key: 'action', width: 110 },
];

const pagination = computed(() => ({
  current: page.value,
  pageSize: perPage.value,
  total: total.value,
  showSizeChanger: true,
}));

const categoryOptions = computed(() => categories.value.map((item) => ({ label: item.name, value: item.name })));

async function load() {
  loading.value = true;
  try {
    const data = await fetchAgentClasses({ q: search.value, category: category.value, page: page.value, perPage: perPage.value });
    rows.value = data.items;
    total.value = data.total;
  } catch (error) {
    message.error(error instanceof Error ? error.message : '课程加载失败');
  } finally {
    loading.value = false;
  }
}

function reload() {
  page.value = 1;
  void load();
}

async function loadCategories() {
  try {
    const data = await fetchAgentCategories();
    categories.value = data.items;
  } catch (error) {
    message.error(error instanceof Error ? error.message : '分类加载失败');
  }
}

function handleTableChange(next: { current?: number; pageSize?: number }) {
  page.value = next.current || 1;
  perPage.value = next.pageSize || 20;
  void load();
}

function openSubmit(row: AgentClassRow) {
  selectedClass.value = row;
  Object.assign(form, {
    school: '',
    studentName: '',
    account: '',
    accountPassword: '',
    courseId: '',
    courseName: row.name,
    durationMinutes: 0,
    flashMode: false,
  });
  submitOpen.value = true;
}

async function submitOrder() {
  if (!selectedClass.value) return;
  if (!form.account.trim() || !form.courseName.trim()) {
    message.error('学习账号和课程名称不能为空');
    return;
  }
  submitting.value = true;
  try {
    await createAgentOrder({
      classId: selectedClass.value.id,
      school: form.school,
      studentName: form.studentName,
      account: form.account,
      accountPassword: form.accountPassword,
      courseId: form.courseId,
      courseName: form.courseName,
      durationMinutes: form.durationMinutes,
      flashMode: form.flashMode,
    });
    submitOpen.value = false;
    message.success('订单已提交');
    emit('changed');
  } catch (error) {
    message.error(error instanceof Error ? error.message : '订单提交失败');
  } finally {
    submitting.value = false;
  }
}

onMounted(() => {
  void loadCategories();
  void load();
});
</script>
