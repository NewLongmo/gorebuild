<template>
  <section class="page wk29-page">
    <div class="page-heading">
      <div>
        <h1>29网课对接</h1>
        <p>拉取 29网课远程商品库，筛选后同步到本地课程。</p>
      </div>
      <a-button @click="loadConnectors">刷新通道</a-button>
    </div>

    <div class="wk29-panel">
      <div class="wk29-panel__title">
        <span class="step-mark">1</span>
        <strong>配置货源并检索全量库</strong>
      </div>
      <div class="wk29-form-grid">
        <a-form-item label="资源列表 ID(HID)">
          <a-select
            v-model:value="form.connectorId"
            :options="connectorOptions"
            :loading="connectorLoading"
            placeholder="选择 29wk 通道"
          />
        </a-form-item>
        <a-form-item label="全局加价倍率">
          <a-input-number v-model:value="form.priceRate" :min="0" :max="1000" :step="0.01" class="full-input" />
        </a-form-item>
        <a-form-item label="跳过已有商品">
          <a-switch v-model:checked="form.skipExisting" />
        </a-form-item>
        <a-form-item label=" ">
          <a-button type="primary" :loading="pulling" class="full-input" @click="pullRemote">
            <template #icon><CloudDownloadOutlined /></template>
            拉取远程资源数据
          </a-button>
        </a-form-item>
      </div>
    </div>

    <div class="wk29-panel">
      <div class="wk29-panel__title">
        <span class="step-mark step-mark--green">2</span>
        <strong>精细化筛选与策略同步确认</strong>
      </div>
      <div class="wk29-actions">
        <a-checkbox v-model:checked="syncOptions.updateName">更新改名字</a-checkbox>
        <a-checkbox v-model:checked="syncOptions.updateDescription">更新商品说明</a-checkbox>
        <a-checkbox v-model:checked="syncOptions.syncCategories">同步分类</a-checkbox>
        <a-button type="primary" :disabled="!preview || selectedProductCount === 0" :loading="syncing" @click="syncSelected">
          <template #icon><CheckCircleOutlined /></template>
          确认勾选并同步入库
        </a-button>
      </div>

      <a-alert
        v-if="preview"
        class="wk29-summary"
        type="info"
        show-icon
        :message="`已拉取 ${preview.totalCategories} 个分类、${preview.totalProducts} 个商品，当前勾选 ${selectedProductCount} 个商品。`"
      />

      <a-empty v-if="!preview" description="请先拉取远程资源数据" />
      <a-table
        v-else
        row-key="upstreamId"
        :columns="categoryColumns"
        :data-source="preview.categories"
        :pagination="false"
        :scroll="{ x: 980 }"
      >
        <template #bodyCell="{ column, record }">
          <a-checkbox v-if="column.key === 'enabled'" v-model:checked="record.enabled" @change="toggleCategory(record)" />
          <a-tag v-else-if="column.key === 'upstreamId'" color="blue">ID: {{ record.upstreamId }}</a-tag>
          <a-badge v-else-if="column.key === 'productCount'" :count="record.productCount" :number-style="{ backgroundColor: '#1677ff' }" />
          <div v-else-if="column.key === 'strategy'" class="strategy-cell">
            <a-segmented v-model:value="record.strategy" :options="strategyOptions" @change="onStrategyChange(record)" />
            <a-select
              v-if="record.strategy === 'bind_existing'"
              v-model:value="record.localCategory"
              show-search
              allow-clear
              placeholder="选择本地分类"
              :options="localCategoryOptions"
              class="strategy-cell__select"
            />
            <span v-else class="strategy-cell__text">{{ strategyCategoryText(record) }}</span>
          </div>
          <span v-else-if="column.key === 'actions'" class="table-actions">
            <a-button size="small" @click="selectCategoryProducts(record, true)">全选商品</a-button>
            <a-button size="small" @click="selectCategoryProducts(record, false)">全不选商品</a-button>
          </span>
        </template>

        <template #expandedRowRender="{ record: category }">
          <div class="wk29-product-toolbar">
            <strong>【{{ category.upstreamName }}】名下商品精确控制</strong>
            <span>{{ categorySelectedCount(category) }} / {{ category.products.length }}</span>
          </div>
          <a-table
            row-key="upstreamId"
            size="small"
            :columns="productColumns"
            :data-source="category.products"
            :pagination="false"
            :scroll="{ x: 960 }"
          >
            <template #bodyCell="{ column, record: product }">
              <a-checkbox
                v-if="column.key === 'sync'"
                v-model:checked="product.sync"
                :disabled="!category.enabled || (form.skipExisting && product.existing)"
              />
              <a-tag v-else-if="column.key === 'upstreamId'">CID: {{ product.upstreamId }}</a-tag>
              <a-tag v-else-if="column.key === 'status'" :color="product.status === 'online' ? 'green' : 'default'">
                {{ product.status === 'online' ? '上架' : '下架' }}
              </a-tag>
              <a-tag v-else-if="column.key === 'existing'" :color="product.existing ? 'green' : 'blue'">
                {{ product.existing ? `已对接 #${product.existingClassId}` : '新商品' }}
              </a-tag>
              <span v-else-if="column.key === 'sourcePrice'">￥{{ formatMoney(product.sourcePrice) }}</span>
              <strong v-else-if="column.key === 'generatedPrice'">￥{{ formatMoney(product.generatedPrice) }}</strong>
              <span v-else-if="column.key === 'description'" class="wk29-description">{{ product.description }}</span>
            </template>
          </a-table>
        </template>
      </a-table>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue';
import { message } from 'ant-design-vue';
import { CheckCircleOutlined, CloudDownloadOutlined } from '@ant-design/icons-vue';
import {
  fetchConnectors,
  pullWK29Classes,
  syncWK29Classes,
  type ConnectorRow,
  type WK29CategoryPreview,
  type WK29PullResult,
} from '@/api/admin';

const connectorLoading = ref(false);
const pulling = ref(false);
const syncing = ref(false);
const connectors = ref<ConnectorRow[]>([]);
const preview = ref<WK29PullResult | null>(null);

const form = reactive({
  connectorId: undefined as number | undefined,
  priceRate: 1.2,
  skipExisting: true,
});

const syncOptions = reactive({
  updateName: false,
  updateDescription: false,
  syncCategories: true,
});

const strategyOptions = [
  { label: '绑定已有', value: 'bind_existing' },
  { label: '创建新类', value: 'create_new' },
  { label: '独立分类', value: 'independent' },
];

const categoryColumns = [
  { title: '启用分类', key: 'enabled', width: 110 },
  { title: '对方分类 ID', key: 'upstreamId', width: 140 },
  { title: '对方分类名称', dataIndex: 'upstreamName', width: 220 },
  { title: '包含商品数', key: 'productCount', width: 120 },
  { title: '本地分类处理策略', key: 'strategy', width: 360 },
  { title: '操作', key: 'actions', width: 190 },
];

const productColumns = [
  { title: '同步该项', key: 'sync', width: 100 },
  { title: '原始 CID', key: 'upstreamId', width: 120 },
  { title: '商品名称', dataIndex: 'name', width: 260 },
  { title: '对接状态', key: 'existing', width: 120 },
  { title: '资源价格', key: 'sourcePrice', width: 110 },
  { title: '生成售价', key: 'generatedPrice', width: 110 },
  { title: '状态', key: 'status', width: 90 },
  { title: '商品说明', key: 'description', width: 360 },
];

const connectorOptions = computed(() =>
  connectors.value
    .filter((item) => item.kind === '29wk')
    .map((item) => ({ label: `${item.id} - ${item.name}`, value: item.id })),
);

const localCategoryOptions = computed(() => (preview.value?.localCategories || []).map((name) => ({ label: name, value: name })));

const selectedProductCount = computed(() => {
  if (!preview.value) {
    return 0;
  }
  return preview.value.categories.reduce((sum, category) => {
    if (!category.enabled) {
      return sum;
    }
    return sum + category.products.filter((product) => product.sync).length;
  }, 0);
});

watch(
  () => form.skipExisting,
  () => {
    if (!preview.value) {
      return;
    }
    for (const category of preview.value.categories) {
      for (const product of category.products) {
        if (form.skipExisting && product.existing) {
          product.sync = false;
        } else if (category.enabled && !product.existing) {
          product.sync = true;
        }
      }
    }
  },
);

async function loadConnectors() {
  connectorLoading.value = true;
  try {
    const data = await fetchConnectors({ status: 'active', perPage: 100 });
    connectors.value = data.items;
    if (!form.connectorId && connectorOptions.value.length > 0) {
      form.connectorId = connectorOptions.value[0].value;
    }
  } catch (error) {
    message.error(error instanceof Error ? error.message : '通道加载失败');
  } finally {
    connectorLoading.value = false;
  }
}

async function pullRemote() {
  if (!form.connectorId) {
    message.error('请选择 29wk 通道');
    return;
  }
  pulling.value = true;
  try {
    preview.value = await pullWK29Classes({
      connectorId: form.connectorId,
      priceRate: form.priceRate,
      skipExisting: form.skipExisting,
    });
  } catch (error) {
    message.error(error instanceof Error ? error.message : '远程资源拉取失败');
  } finally {
    pulling.value = false;
  }
}

async function syncSelected() {
  if (!preview.value || !form.connectorId) {
    return;
  }
  if (selectedProductCount.value === 0) {
    message.error('请至少勾选一个商品');
    return;
  }
  syncing.value = true;
  try {
    const result = await syncWK29Classes({
      connectorId: form.connectorId,
      priceRate: form.priceRate,
      syncCategories: syncOptions.syncCategories,
      updateName: syncOptions.updateName,
      updateDescription: syncOptions.updateDescription,
      skipExisting: form.skipExisting,
      categories: preview.value.categories.map((category) => ({
        upstreamId: category.upstreamId,
        enabled: category.enabled,
        strategy: category.strategy,
        localCategory: category.localCategory,
        products: category.products.map((product) => ({ upstreamId: product.upstreamId, sync: product.sync })),
      })),
    });
    message.success(`同步完成：新增 ${result.inserted}，更新 ${result.updated}，跳过 ${result.skipped}`);
    await pullRemote();
  } catch (error) {
    message.error(error instanceof Error ? error.message : '同步入库失败');
  } finally {
    syncing.value = false;
  }
}

function toggleCategory(category: WK29CategoryPreview) {
  if (category.enabled && category.strategy === 'bind_existing' && !category.localCategory) {
    category.localCategory = preview.value?.localCategories.includes(category.upstreamName) ? category.upstreamName : '';
  }
  selectCategoryProducts(category, category.enabled);
}

function selectCategoryProducts(category: WK29CategoryPreview, checked: boolean) {
  if (checked) {
    category.enabled = true;
  }
  for (const product of category.products) {
    product.sync = checked && !(form.skipExisting && product.existing);
  }
}

function categorySelectedCount(category: WK29CategoryPreview) {
  return category.products.filter((product) => product.sync).length;
}

function onStrategyChange(category: WK29CategoryPreview) {
  if (category.strategy === 'bind_existing') {
    category.localCategory = preview.value?.localCategories.includes(category.upstreamName) ? category.upstreamName : category.localCategory;
  } else {
    category.localCategory = '';
  }
}

function strategyCategoryText(category: WK29CategoryPreview) {
  if (category.strategy === 'independent') {
    return `${category.upstreamName} (ID:${category.upstreamId})`;
  }
  return category.upstreamName;
}

function formatMoney(value: number) {
  return Number(value || 0).toFixed(2);
}

onMounted(() => {
  void loadConnectors();
});
</script>

<style scoped>
.wk29-page {
  display: grid;
  gap: 16px;
}

.wk29-panel {
  padding: 18px;
  border: 1px solid #e5e7eb;
  border-radius: 8px;
  background: #fff;
}

.wk29-panel__title,
.wk29-actions,
.wk29-product-toolbar {
  display: flex;
  align-items: center;
  gap: 12px;
}

.wk29-panel__title {
  margin-bottom: 16px;
}

.step-mark {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 24px;
  height: 24px;
  border-radius: 50%;
  background: #1677ff;
  color: #fff;
  font-weight: 700;
}

.step-mark--green {
  background: #16a34a;
}

.wk29-form-grid {
  display: grid;
  grid-template-columns: minmax(220px, 1fr) minmax(180px, 0.8fr) minmax(120px, 0.45fr) minmax(220px, 1fr);
  gap: 14px;
  align-items: end;
}

.wk29-actions {
  justify-content: flex-end;
  flex-wrap: wrap;
  margin-bottom: 14px;
}

.wk29-summary {
  margin-bottom: 14px;
}

.strategy-cell {
  display: grid;
  grid-template-columns: auto minmax(150px, 1fr);
  gap: 8px;
  align-items: center;
}

.strategy-cell__select {
  min-width: 160px;
}

.strategy-cell__text,
.wk29-description {
  color: #667085;
}

.wk29-description {
  display: inline-block;
  max-width: 340px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.wk29-product-toolbar {
  justify-content: space-between;
  margin-bottom: 10px;
}

@media (max-width: 960px) {
  .wk29-form-grid,
  .strategy-cell {
    grid-template-columns: 1fr;
  }

  .wk29-actions {
    justify-content: flex-start;
  }
}
</style>
