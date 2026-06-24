<template>
  <section class="page">
    <div class="page-heading">
      <div>
        <h1>操作日志</h1>
        <p>查看 API 操作与后台任务产生的审计事件。</p>
      </div>
      <a-button :loading="loading" @click="load">刷新</a-button>
    </div>

    <DataToolbar v-model="query.q" placeholder="搜索类型、内容或用户 ID" @search="reload" />
    <a-table
      row-key="id"
      :columns="columns"
      :data-source="rows"
      :loading="loading"
      :pagination="pagination"
      @change="onTableChange"
    />
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue';
import { message } from 'ant-design-vue';
import DataToolbar from '@/components/DataToolbar.vue';
import { fetchLogs, type LogRow } from '@/api/admin';

const loading = ref(false);
const rows = ref<LogRow[]>([]);
const total = ref(0);
const query = reactive({ q: '', page: 1, perPage: 20 });

const columns = [
  { title: 'ID', dataIndex: 'id', width: 80 },
  { title: '用户', dataIndex: 'userId', width: 90 },
  { title: '类型', dataIndex: 'type', width: 150 },
  { title: '内容', dataIndex: 'text' },
  { title: '金额', dataIndex: 'amount', width: 110 },
  { title: '创建时间', dataIndex: 'createdAt', width: 180 },
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
    const data = await fetchLogs(query);
    rows.value = data.items;
    total.value = data.total;
  } catch (error) {
    message.error(error instanceof Error ? error.message : '日志加载失败');
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

onMounted(load);
</script>
