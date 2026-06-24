<template>
  <section class="page">
    <div class="page-heading">
      <div>
        <h1>代理层级</h1>
        <p>查看平台代理账号与直属下级关系。</p>
      </div>
      <a-button @click="load">刷新</a-button>
    </div>

    <DataToolbar v-model="search" placeholder="搜索代理账号、名称或 ID" @search="load">
      <div class="data-toolbar__filters">
        <span class="tree-summary">共 {{ total }} 个代理，当前匹配 {{ matched }} 个</span>
      </div>
    </DataToolbar>

    <a-alert
      v-if="truncated"
      class="page-note"
      type="warning"
      show-icon
      message="代理数量超过当前加载上限，树形结果已截断。"
    />

    <a-table
      row-key="id"
      :columns="columns"
      :data-source="rows"
      :loading="loading"
      :pagination="false"
      :scroll="{ x: 980 }"
    >
      <template #bodyCell="{ column, record }">
        <template v-if="column.key === 'account'">
          <span class="agent-account">{{ record.account }}</span>
        </template>
        <template v-else-if="column.key === 'parent'">
          <span v-if="record.parentAccount">{{ record.parentAccount }}</span>
          <span v-else-if="record.parentId">#{{ record.parentId }}</span>
          <span v-else>平台直属</span>
        </template>
        <template v-else-if="column.key === 'balance'">
          {{ Number(record.balance).toFixed(2) }}
        </template>
        <template v-else-if="column.key === 'priceRate'">
          {{ Number(record.priceRate).toFixed(4) }}
        </template>
        <template v-else-if="column.key === 'status'">
          <a-tag :color="record.status === 'active' ? 'green' : 'red'">
            {{ record.status === 'active' ? '启用' : '停用' }}
          </a-tag>
        </template>
        <template v-else-if="column.key === 'children'">
          {{ record.directChildren }}
        </template>
      </template>
    </a-table>
  </section>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue';
import { message } from 'ant-design-vue';
import DataToolbar from '@/components/DataToolbar.vue';
import { fetchAgentTree, type AgentTreeNode } from '@/api/admin';

const loading = ref(false);
const rows = ref<AgentTreeNode[]>([]);
const search = ref('');
const total = ref(0);
const matched = ref(0);
const truncated = ref(false);

const columns = [
  { title: 'ID', dataIndex: 'id', width: 80 },
  { title: '代理账号', key: 'account', dataIndex: 'account', width: 220 },
  { title: '名称', dataIndex: 'name', width: 160 },
  { title: '上级代理', key: 'parent', width: 160 },
  { title: '余额', key: 'balance', width: 120 },
  { title: '价格倍率', key: 'priceRate', width: 120 },
  { title: '直属下级', key: 'children', width: 110 },
  { title: '状态', key: 'status', width: 100 },
  { title: '创建时间', dataIndex: 'createdAt', width: 180 },
];

async function load() {
  loading.value = true;
  try {
    const data = await fetchAgentTree({ q: search.value, limit: 5000 });
    rows.value = data.items;
    total.value = data.total;
    matched.value = data.matched;
    truncated.value = data.truncated;
  } catch (error) {
    message.error(error instanceof Error ? error.message : '代理层级加载失败');
  } finally {
    loading.value = false;
  }
}

onMounted(() => {
  void load();
});
</script>
