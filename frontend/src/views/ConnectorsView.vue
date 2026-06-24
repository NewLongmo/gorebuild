<template>
  <section class="page connectors-page">
    <div class="page-heading">
      <div>
        <h1>货源管理</h1>
        <p>统一维护网课货源接口、29 对接配置、商品拉取与同步策略。</p>
      </div>
      <a-button type="primary" @click="openCreate">新增货源</a-button>
    </div>

    <DataToolbar v-model="query.q" placeholder="搜索货源名称、URL 或类型" @search="reload">
      <a-select
        v-model:value="query.status"
        allow-clear
        class="status-select"
        placeholder="状态"
        :options="statusOptions"
        @change="reload"
      />
    </DataToolbar>

    <a-table
      row-key="id"
      :columns="columns"
      :data-source="rows"
      :loading="loading"
      :pagination="pagination"
      :scroll="{ x: 1280 }"
      @change="onTableChange"
    >
      <template #bodyCell="{ column, record }">
        <a-tag v-if="column.key === 'status'" :color="record.status === 'active' ? 'green' : 'default'">
          {{ record.status === 'active' ? '启用' : '停用' }}
        </a-tag>
        <a-tag v-else-if="column.key === 'kind'" :color="record.kind === '29wk' ? 'blue' : 'default'">
          {{ accessLabel(record.kind) }}
        </a-tag>
        <span v-else-if="column.key === 'price'">{{ priceRuleText(record) }}</span>
        <span v-else-if="column.key === 'sync'">
          <a-tag :color="record.orderSyncEnabled ? 'green' : 'default'">订单{{ record.orderSyncEnabled ? '开' : '关' }}</a-tag>
          <a-tag :color="record.sourceSyncEnabled ? 'green' : 'default'">货源{{ record.sourceSyncEnabled ? '开' : '关' }}</a-tag>
        </span>
        <span v-else-if="column.key === 'actions'" class="table-actions connector-actions">
          <a-button size="small" @click="queryBalance(record)">
            <template #icon><DollarOutlined /></template>
            金额
          </a-button>
          <a-button size="small" :disabled="record.kind !== '29wk'" @click="openPull(record)">
            <template #icon><CloudDownloadOutlined /></template>
            拉取
          </a-button>
          <a-button size="small" @click="openSecret(record)">
            <template #icon><KeyOutlined /></template>
            改密
          </a-button>
          <a-button size="small" @click="toggleStatus(record)">
            <template #icon><PauseCircleOutlined /></template>
            {{ record.status === 'active' ? '暂停' : '启用' }}
          </a-button>
          <a-button size="small" @click="openConfig(record)">
            <template #icon><SettingOutlined /></template>
            同步配置
          </a-button>
          <a-button size="small" :disabled="record.kind !== '29wk'" @click="openPriceSync(record)">
            <template #icon><SyncOutlined /></template>
            更新价格
          </a-button>
          <a-button
            size="small"
            :disabled="record.kind !== '29wk'"
            :loading="orderSyncingIds.has(record.id)"
            @click="runOrderSync(record)"
          >
            <template #icon><SyncOutlined /></template>
            同步订单
          </a-button>
          <a-button size="small" @click="openEdit(record)">编辑</a-button>
          <a-popconfirm title="确定删除该货源？会同时删除该货源下的平台/课程，并清理空分类。" @confirm="deleteRow(record.id)">
            <a-button size="small" danger>删除</a-button>
          </a-popconfirm>
        </span>
      </template>
    </a-table>

    <a-modal v-model:open="modalOpen" :title="editingId ? '编辑货源' : '新增货源'" @ok="save" @cancel="modalOpen = false">
      <a-form layout="vertical" :model="form">
        <a-form-item label="接口名称" required>
          <a-input v-model:value="form.name" />
        </a-form-item>
        <a-form-item label="接入方式">
          <a-segmented v-model:value="form.accessType" :options="accessOptions" />
        </a-form-item>
        <a-form-item label="接口地址" :required="form.status === 'active'">
          <a-input v-model:value="form.baseUrl" />
        </a-form-item>
        <div class="connector-form-grid">
          <a-form-item label="UID">
            <a-input v-model:value="form.appKey" />
          </a-form-item>
          <a-form-item label="Key">
            <a-input-password v-model:value="form.appSecret" :placeholder="editingId ? '留空则不覆盖' : ''" />
          </a-form-item>
        </div>
        <div class="connector-form-grid connector-form-grid--small">
          <a-form-item label="排序">
            <a-input-number v-model:value="form.sortOrder" :min="0" :max="9999" class="full-input" />
          </a-form-item>
          <a-form-item label="状态">
            <a-select v-model:value="form.status" :options="statusOptions" />
          </a-form-item>
        </div>
      </a-form>
    </a-modal>

    <a-modal v-model:open="secretOpen" title="修改货源密钥" @ok="saveSecret" @cancel="secretOpen = false">
      <a-form layout="vertical">
        <a-form-item label="新 Key" required>
          <a-input-password v-model:value="secretForm.appSecret" />
        </a-form-item>
      </a-form>
    </a-modal>

    <a-modal
      v-model:open="configOpen"
      title="货源同步配置"
      width="860px"
      :confirm-loading="configSaving"
      @ok="saveConfigOnly"
      @cancel="configOpen = false"
    >
      <template #footer>
        <a-button @click="configOpen = false">取消</a-button>
        <a-button :loading="configSaving" @click="saveConfigOnly">保存</a-button>
        <a-button type="primary" :loading="syncingNow" @click="saveAndSync">保存并同步</a-button>
      </template>
      <a-tabs v-model:activeKey="configTab">
        <a-tab-pane key="source" tab="货源级别">
          <div class="sync-config-grid">
            <a-form-item label="订单同步">
              <a-switch v-model:checked="configForm.orderSyncEnabled" />
            </a-form-item>
            <a-form-item label="货源同步">
              <a-switch v-model:checked="configForm.sourceSyncEnabled" />
            </a-form-item>
            <a-form-item label="价格模式">
              <a-select v-model:value="configForm.priceMode" :options="priceModeOptions" />
            </a-form-item>
            <a-form-item :label="configForm.priceMode === 'fixed_add' ? '固定加价' : '价格倍数'">
              <a-input-number v-model:value="configForm.priceValue" :min="0" :max="100000" :step="0.01" class="full-input" />
            </a-form-item>
            <a-form-item label="价格取整">
              <a-select v-model:value="configForm.priceRounding" :options="roundingOptions" />
            </a-form-item>
          </div>
          <div class="config-section">
            <div class="section-title">
              <strong>词替换规则</strong>
              <a-button size="small" @click="addReplaceRule">
                <template #icon><PlusOutlined /></template>
                新增
              </a-button>
            </div>
            <div v-for="(rule, index) in configForm.replaceRules" :key="index" class="replace-row">
              <a-input v-model:value="rule.from" placeholder="原词" />
              <a-input v-model:value="rule.to" placeholder="替换为" />
              <a-button danger @click="removeReplaceRule(index)">
                <template #icon><DeleteOutlined /></template>
              </a-button>
            </div>
            <a-empty v-if="configForm.replaceRules.length === 0" description="暂无词替换规则" />
          </div>
          <div class="config-section">
            <div class="section-title">
              <strong>分类价格配置</strong>
              <a-button size="small" @click="addCategoryRule">
                <template #icon><PlusOutlined /></template>
                新增
              </a-button>
            </div>
            <a-table
              row-key="upstreamId"
              size="small"
              :columns="categoryRuleColumns"
              :data-source="configForm.categoryRules"
              :pagination="false"
            >
              <template #bodyCell="{ column, record, index }">
                <a-input v-if="column.key === 'upstreamId'" v-model:value="record.upstreamId" placeholder="上游分类 ID" />
                <a-select v-else-if="column.key === 'priceMode'" v-model:value="record.priceMode" :options="priceModeOptions" />
                <a-input-number
                  v-else-if="column.key === 'priceValue'"
                  v-model:value="record.priceValue"
                  :min="0"
                  :max="100000"
                  :step="0.01"
                  class="full-input"
                />
                <a-select v-else-if="column.key === 'priceRounding'" v-model:value="record.priceRounding" :options="roundingOptions" />
                <a-button v-else-if="column.key === 'actions'" danger @click="removeCategoryRule(index)">
                  <template #icon><DeleteOutlined /></template>
                </a-button>
              </template>
            </a-table>
          </div>
        </a-tab-pane>
        <a-tab-pane key="system" tab="系统级别">
          <div class="sync-config-grid">
            <a-form-item label="同步分类">
              <a-switch v-model:checked="syncOptions.syncCategories" />
            </a-form-item>
            <a-form-item label="更新商品名">
              <a-switch v-model:checked="syncOptions.updateName" />
            </a-form-item>
            <a-form-item label="更新商品说明">
              <a-switch v-model:checked="syncOptions.updateDescription" />
            </a-form-item>
            <a-form-item label="跳过已有商品">
              <a-switch v-model:checked="syncOptions.skipExisting" />
            </a-form-item>
          </div>
        </a-tab-pane>
      </a-tabs>
    </a-modal>

    <a-modal
      v-model:open="pullOpen"
      title="拉取上游商品"
      width="1120px"
      :confirm-loading="syncingSelected"
      @ok="syncSelectedProducts"
      @cancel="pullOpen = false"
    >
      <template #footer>
        <a-button @click="clearSelection">清空</a-button>
        <a-button @click="selectFiltered">全选</a-button>
        <a-button @click="pullProducts" :loading="pulling">
          <template #icon><CloudDownloadOutlined /></template>
          重新拉取
        </a-button>
        <a-button type="primary" :disabled="selectedProductCount === 0" :loading="syncingSelected" @click="syncSelectedProducts">
          确定上架 {{ selectedProductCount }}
        </a-button>
      </template>
      <div class="pull-toolbar">
        <a-select v-model:value="pullState.connectorId" :options="connectorOptions" class="source-select" />
        <a-select v-model:value="pullState.categoryId" allow-clear placeholder="分类" :options="previewCategoryOptions" class="source-select" />
        <a-input-search v-model:value="pullState.keyword" placeholder="搜索商品 ID 或名称" class="pull-search" />
      </div>
      <a-alert
        v-if="preview"
        type="info"
        show-icon
        class="pull-summary"
        :message="`已拉取 ${preview.totalCategories} 个分类、${preview.totalProducts} 个商品，当前选中 ${selectedProductCount} 个。`"
      />
      <a-empty v-if="!preview" description="请选择货源后拉取商品" />
      <a-table
        v-else
        row-key="key"
        size="small"
        :columns="productColumns"
        :data-source="filteredProducts"
        :pagination="{ pageSize: 12, showSizeChanger: true }"
        :scroll="{ x: 1050 }"
      >
        <template #bodyCell="{ column, record }">
          <a-checkbox
            v-if="column.key === 'select'"
            :checked="selectedProducts.has(record.key)"
            :disabled="syncOptions.skipExisting && record.existing"
            @change="toggleProduct(record.key, $event)"
          />
          <span v-else-if="column.key === 'category'">{{ record.categoryName }}</span>
          <a-tag v-else-if="column.key === 'upstreamId'">{{ record.upstreamId }}</a-tag>
          <a-tag v-else-if="column.key === 'existing'" :color="record.existing ? 'green' : 'blue'">
            {{ record.existing ? `已存在 #${record.existingClassId}` : '新商品' }}
          </a-tag>
          <span v-else-if="column.key === 'sourcePrice'">￥{{ formatMoney(record.sourcePrice) }}</span>
          <strong v-else-if="column.key === 'generatedPrice'">￥{{ formatMoney(record.generatedPrice) }}</strong>
        </template>
      </a-table>
    </a-modal>

    <a-modal
      v-model:open="priceSyncOpen"
      title="按上游更新本地价格/状态"
      width="720px"
      :confirm-loading="priceSyncing"
      @ok="runPriceSync"
      @cancel="priceSyncOpen = false"
    >
      <a-alert
        type="warning"
        show-icon
        class="price-sync-tip"
        message="该操作只更新已存在的本地课程价格、状态、说明和对接开关，不会新增商品。下架缺失商品仅在选择全部分类时生效。"
      />
      <a-form layout="vertical" :model="priceSyncForm">
        <div class="sync-config-grid">
          <a-form-item label="货源">
            <a-select v-model:value="priceSyncForm.connectorId" :options="wk29ConnectorOptions" @change="onPriceSyncConnectorChange" />
          </a-form-item>
          <a-form-item label="上游分类">
            <div class="category-select-row">
              <a-select v-model:value="priceSyncForm.upstreamCategoryId" :options="priceSyncCategoryOptions" />
              <a-button :loading="pullingPriceCategories" @click="pullPriceSyncCategories">拉取分类</a-button>
            </div>
          </a-form-item>
          <a-form-item label="价格模式">
            <a-select v-model:value="priceSyncForm.priceMode" :options="priceModeOptions" />
          </a-form-item>
          <a-form-item :label="priceSyncForm.priceMode === 'fixed_add' ? '固定加价' : '价格倍数'">
            <a-input-number v-model:value="priceSyncForm.priceValue" :min="0" :max="100000" :step="0.01" class="full-input" />
          </a-form-item>
          <a-form-item label="价格取整">
            <a-select v-model:value="priceSyncForm.priceRounding" :options="roundingOptions" />
          </a-form-item>
          <a-form-item label="下架缺失商品">
            <a-switch v-model:checked="priceSyncForm.offlineMissing" :disabled="priceSyncForm.upstreamCategoryId !== 'all'" />
          </a-form-item>
        </div>
      </a-form>
    </a-modal>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue';
import { message } from 'ant-design-vue';
import {
  CloudDownloadOutlined,
  DeleteOutlined,
  DollarOutlined,
  KeyOutlined,
  PauseCircleOutlined,
  PlusOutlined,
  SettingOutlined,
  SyncOutlined,
} from '@ant-design/icons-vue';
import DataToolbar from '@/components/DataToolbar.vue';
import {
  createConnector,
  fetchConnectorBalance,
  fetchConnectors,
  pullWK29Classes,
  removeConnector,
  syncWK29Classes,
  syncWK29Orders,
  syncWK29Prices,
  updateConnector,
  type CascadeDeleteResult,
  type ConnectorCategoryPriceRule,
  type ConnectorReplaceRule,
  type ConnectorRow,
  type WK29CategoryPreview,
  type WK29ProductPreview,
  type WK29PullResult,
} from '@/api/admin';

const MAX_TIMEOUT_MS = 60000;
const loading = ref(false);
const rows = ref<ConnectorRow[]>([]);
const total = ref(0);
const modalOpen = ref(false);
const secretOpen = ref(false);
const configOpen = ref(false);
const pullOpen = ref(false);
const priceSyncOpen = ref(false);
const pulling = ref(false);
const pullingPriceCategories = ref(false);
const configSaving = ref(false);
const syncingNow = ref(false);
const syncingSelected = ref(false);
const priceSyncing = ref(false);
const orderSyncingIds = ref<Set<number>>(new Set());
const editingId = ref<number | null>(null);
const activeConnector = ref<ConnectorRow | null>(null);
const preview = ref<WK29PullResult | null>(null);
const selectedProducts = ref<Set<string>>(new Set());
const configTab = ref('source');
const query = reactive({ q: '', status: undefined as string | undefined, page: 1, perPage: 20 });

interface ConnectorForm {
  name: string;
  baseUrl: string;
  appKey: string;
  appSecret: string;
  accessType: '29_common' | 'common' | 'custom';
  status: string;
  sortOrder: number;
  timeoutMs: number;
}

interface CategoryRuleForm extends ConnectorCategoryPriceRule {
  upstreamId: string;
}

interface ProductRow extends WK29ProductPreview {
  key: string;
  categoryId: string;
  categoryName: string;
}

const form = reactive<ConnectorForm>({
  name: '',
  baseUrl: '',
  appKey: '',
  appSecret: '',
  accessType: '29_common',
  status: 'active',
  sortOrder: 10,
  timeoutMs: 8000,
});

const secretForm = reactive({ appSecret: '' });

const configForm = reactive({
  orderSyncEnabled: true,
  sourceSyncEnabled: true,
  priceMode: 'multiplier',
  priceValue: 1,
  priceRounding: 'none',
  replaceRules: [] as ConnectorReplaceRule[],
  categoryRules: [] as CategoryRuleForm[],
});

const syncOptions = reactive({
  syncCategories: true,
  updateName: false,
  updateDescription: false,
  skipExisting: true,
});

const pullState = reactive({
  connectorId: undefined as number | undefined,
  keyword: '',
  categoryId: undefined as string | undefined,
});

const priceSyncForm = reactive({
  connectorId: undefined as number | undefined,
  upstreamCategoryId: 'all',
  priceMode: 'multiplier',
  priceValue: 1,
  priceRounding: 'none',
  offlineMissing: false,
});

const statusOptions = [
  { label: '启用', value: 'active' },
  { label: '停用', value: 'disabled' },
];

const accessOptions = [
  { label: '29通用', value: '29_common' },
  { label: '常见货源', value: 'common' },
  { label: '自定义接口', value: 'custom' },
];

const priceModeOptions = [
  { label: '价格倍率', value: 'multiplier' },
  { label: '固定加价', value: 'fixed_add' },
];

const roundingOptions = [
  { label: '不取整', value: 'none' },
  { label: '向下取整', value: 'floor' },
  { label: '向上取整', value: 'ceil' },
  { label: '四舍五入', value: 'round' },
];

const columns = [
  { title: 'ID', dataIndex: 'id', width: 70 },
  { title: '排序', dataIndex: 'sortOrder', width: 80 },
  { title: '接口名称', dataIndex: 'name', width: 180 },
  { title: '接入方式', key: 'kind', width: 110 },
  { title: '接口地址', dataIndex: 'baseUrl', width: 260 },
  { title: 'UID', dataIndex: 'appKey', width: 150 },
  { title: '同步', key: 'sync', width: 150 },
  { title: '价格策略', key: 'price', width: 160 },
  { title: '状态', key: 'status', width: 90 },
  { title: '操作', key: 'actions', width: 620 },
];

const categoryRuleColumns = [
  { title: '上游分类 ID', key: 'upstreamId', width: 170 },
  { title: '价格模式', key: 'priceMode', width: 150 },
  { title: '数值', key: 'priceValue', width: 140 },
  { title: '取整', key: 'priceRounding', width: 150 },
  { title: '操作', key: 'actions', width: 80 },
];

const productColumns = [
  { title: '选择', key: 'select', width: 70 },
  { title: '分类', key: 'category', width: 150 },
  { title: '商品 ID', key: 'upstreamId', width: 120 },
  { title: '商品名', dataIndex: 'name', width: 260 },
  { title: '成本价', key: 'sourcePrice', width: 110 },
  { title: '上架价', key: 'generatedPrice', width: 110 },
  { title: '本地状态', key: 'existing', width: 130 },
  { title: '说明', dataIndex: 'description', width: 260 },
];

const pagination = computed(() => ({
  current: query.page,
  pageSize: query.perPage,
  total: total.value,
  showSizeChanger: true,
}));

const connectorOptions = computed(() => rows.value.map((item) => ({ label: `${item.id} - ${item.name}`, value: item.id })));

const wk29ConnectorOptions = computed(() =>
  rows.value.filter((item) => item.kind === '29wk').map((item) => ({ label: `${item.id} - ${item.name}`, value: item.id })),
);

const previewCategoryOptions = computed(() =>
  (preview.value?.categories || []).map((category) => ({ label: `${category.upstreamName} (${category.productCount})`, value: category.upstreamId })),
);

const priceSyncCategoryOptions = computed(() => [{ label: '全部分类', value: 'all' }, ...previewCategoryOptions.value]);

const allProducts = computed<ProductRow[]>(() => {
  if (!preview.value) {
    return [];
  }
  return preview.value.categories.flatMap((category) =>
    category.products.map((product) => ({
      ...product,
      key: productKey(category.upstreamId, product.upstreamId),
      categoryId: category.upstreamId,
      categoryName: category.upstreamName,
    })),
  );
});

const filteredProducts = computed(() => {
  const keyword = pullState.keyword.trim().toLowerCase();
  return allProducts.value.filter((item) => {
    if (pullState.categoryId && item.categoryId !== pullState.categoryId) {
      return false;
    }
    if (!keyword) {
      return true;
    }
    return `${item.upstreamId} ${item.kcId} ${item.name}`.toLowerCase().includes(keyword);
  });
});

const selectedProductCount = computed(() => selectedProducts.value.size);

async function load() {
  loading.value = true;
  try {
    const data = await fetchConnectors(query);
    rows.value = data.items.map(withConnectorDefaults);
    total.value = data.total;
  } catch (error) {
    message.error(error instanceof Error ? error.message : '货源加载失败');
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

function resetForm() {
  Object.assign(form, {
    name: '',
    baseUrl: '',
    appKey: '',
    appSecret: '',
    accessType: '29_common',
    status: 'active',
    sortOrder: 10,
    timeoutMs: 8000,
  });
}

function openCreate() {
  editingId.value = null;
  resetForm();
  modalOpen.value = true;
}

function openEdit(row: ConnectorRow) {
  activeConnector.value = row;
  editingId.value = row.id;
  Object.assign(form, {
    name: row.name,
    baseUrl: row.baseUrl,
    appKey: row.appKey,
    appSecret: '',
    accessType: accessTypeFromKind(row.kind),
    status: row.status,
    sortOrder: row.sortOrder || 10,
    timeoutMs: row.timeoutMs || 8000,
  });
  modalOpen.value = true;
}

async function save() {
  const payload = {
    ...connectorPayloadFromForm(),
    ...(editingId.value && activeConnector.value ? configPayload(activeConnector.value) : defaultConfigPayload()),
  };
  if (!validateConnectorPayload(payload)) {
    return;
  }
  try {
    if (editingId.value) {
      await updateConnector(editingId.value, payload);
    } else {
      await createConnector(payload);
    }
    modalOpen.value = false;
    await load();
  } catch (error) {
    message.error(error instanceof Error ? error.message : '货源保存失败');
  }
}

function openSecret(row: ConnectorRow) {
  activeConnector.value = row;
  secretForm.appSecret = '';
  secretOpen.value = true;
}

async function saveSecret() {
  if (!activeConnector.value || !secretForm.appSecret.trim()) {
    message.error('新 Key 不能为空');
    return;
  }
  try {
    await updateConnector(activeConnector.value.id, {
      ...payloadFromConnector(activeConnector.value),
      appSecret: secretForm.appSecret.trim(),
    });
    secretOpen.value = false;
    message.success('密钥已更新');
  } catch (error) {
    message.error(error instanceof Error ? error.message : '密钥更新失败');
  }
}

async function toggleStatus(row: ConnectorRow) {
  try {
    await updateConnector(row.id, {
      ...payloadFromConnector(row),
      status: row.status === 'active' ? 'disabled' : 'active',
    });
    await load();
  } catch (error) {
    message.error(error instanceof Error ? error.message : '状态更新失败');
  }
}

async function deleteRow(id: number) {
  try {
    const result = await removeConnector(id);
    message.success(cascadeDeleteText('货源已删除', result));
    await load();
  } catch (error) {
    message.error(error instanceof Error ? error.message : '货源删除失败');
  }
}

function cascadeDeleteText(prefix: string, result: CascadeDeleteResult) {
  return `${prefix}：平台/课程 ${result.deletedClasses || 0}，收藏 ${result.deletedFavorites || 0}，密价 ${result.deletedSpecialPrices || 0}，空分类 ${result.deletedEmptyCategories || 0}`;
}

function openConfig(row: ConnectorRow) {
  activeConnector.value = row;
  loadConfigForm(row);
  if (preview.value?.connectorId === row.id) {
    syncCategoryRulesFromPreview(preview.value.categories);
  }
  configTab.value = 'source';
  configOpen.value = true;
}

async function saveConfigOnly() {
  if (!activeConnector.value) {
    return;
  }
  configSaving.value = true;
  try {
    await saveConfig(activeConnector.value);
    configOpen.value = false;
    await load();
  } catch (error) {
    message.error(error instanceof Error ? error.message : '配置保存失败');
  } finally {
    configSaving.value = false;
  }
}

async function saveAndSync() {
  if (!activeConnector.value) {
    return;
  }
  syncingNow.value = true;
  try {
    await saveConfig(activeConnector.value);
    const pulled = await pullWK29Classes({
      connectorId: activeConnector.value.id,
      skipExisting: syncOptions.skipExisting,
    });
    const result = await syncWK29Classes({
      connectorId: activeConnector.value.id,
      syncCategories: syncOptions.syncCategories,
      updateName: syncOptions.updateName,
      updateDescription: syncOptions.updateDescription,
      skipExisting: syncOptions.skipExisting,
      categories: pulled.categories.map((category) => ({
        upstreamId: category.upstreamId,
        enabled: true,
        strategy: category.localCategory ? 'bind_existing' : 'create_new',
        localCategory: category.localCategory,
        products: category.products.map((product) => ({ upstreamId: product.upstreamId, sync: !(syncOptions.skipExisting && product.existing) })),
      })),
    });
    message.success(`同步完成：新增 ${result.inserted}，更新 ${result.updated}，跳过 ${result.skipped}`);
    configOpen.value = false;
    await load();
  } catch (error) {
    message.error(error instanceof Error ? error.message : '保存并同步失败');
  } finally {
    syncingNow.value = false;
  }
}

function openPull(row: ConnectorRow) {
  activeConnector.value = row;
  pullState.connectorId = row.id;
  pullState.keyword = '';
  pullState.categoryId = undefined;
  preview.value = null;
  selectedProducts.value = new Set();
  pullOpen.value = true;
  void pullProducts();
}

async function pullProducts() {
  if (!pullState.connectorId) {
    message.error('请选择货源');
    return;
  }
  pulling.value = true;
  try {
    preview.value = await pullWK29Classes({
      connectorId: pullState.connectorId,
      skipExisting: syncOptions.skipExisting,
    });
    selectedProducts.value = new Set(
      preview.value.categories.flatMap((category) =>
        category.products
          .filter((product) => product.sync && !(syncOptions.skipExisting && product.existing))
          .map((product) => productKey(category.upstreamId, product.upstreamId)),
      ),
    );
    if (configOpen.value) {
      syncCategoryRulesFromPreview(preview.value.categories);
    }
  } catch (error) {
    message.error(error instanceof Error ? error.message : '商品拉取失败');
  } finally {
    pulling.value = false;
  }
}

async function syncSelectedProducts() {
  if (!preview.value || !pullState.connectorId || selectedProducts.value.size === 0) {
    return;
  }
  syncingSelected.value = true;
  try {
    const result = await syncWK29Classes({
      connectorId: pullState.connectorId,
      syncCategories: syncOptions.syncCategories,
      updateName: syncOptions.updateName,
      updateDescription: syncOptions.updateDescription,
      skipExisting: syncOptions.skipExisting,
      categories: preview.value.categories.map((category) => {
        const selectedInCategory = category.products.some((product) => selectedProducts.value.has(productKey(category.upstreamId, product.upstreamId)));
        return {
          upstreamId: category.upstreamId,
          enabled: selectedInCategory,
          strategy: category.localCategory ? category.strategy : 'create_new',
          localCategory: category.localCategory,
          products: category.products.map((product) => ({
            upstreamId: product.upstreamId,
            sync: selectedProducts.value.has(productKey(category.upstreamId, product.upstreamId)),
          })),
        };
      }),
    });
    message.success(`上架完成：新增 ${result.inserted}，更新 ${result.updated}，跳过 ${result.skipped}`);
    await pullProducts();
  } catch (error) {
    message.error(error instanceof Error ? error.message : '上架失败');
  } finally {
    syncingSelected.value = false;
  }
}

function selectFiltered() {
  const next = new Set(selectedProducts.value);
  for (const product of filteredProducts.value) {
    if (!(syncOptions.skipExisting && product.existing)) {
      next.add(product.key);
    }
  }
  selectedProducts.value = next;
}

function clearSelection() {
  selectedProducts.value = new Set();
}

function toggleProduct(key: string, event: Event) {
  const checked = (event.target as HTMLInputElement).checked;
  const next = new Set(selectedProducts.value);
  if (checked) {
    next.add(key);
  } else {
    next.delete(key);
  }
  selectedProducts.value = next;
}

async function queryBalance(row: ConnectorRow) {
  if (row.kind !== '29wk') {
    message.info('当前仅 29 通用货源支持余额查询');
    return;
  }
  try {
    const result = await fetchConnectorBalance(row.id);
    message.success(`${row.name} 余额：￥${formatMoney(result.balance)}`);
  } catch (error) {
    message.error(error instanceof Error ? error.message : '余额查询失败');
  }
}

function openPriceSync(row: ConnectorRow) {
  if (row.kind !== '29wk') {
    message.info('当前仅 29 通用货源支持按上游更新价格');
    return;
  }
  activeConnector.value = row;
  Object.assign(priceSyncForm, {
    connectorId: row.id,
    upstreamCategoryId: 'all',
    priceMode: normalizePriceMode(row.priceMode),
    priceValue: Number(row.priceValue || 1),
    priceRounding: normalizeRounding(row.priceRounding),
    offlineMissing: false,
  });
  if (preview.value?.connectorId !== row.id) {
    preview.value = null;
  }
  priceSyncOpen.value = true;
}

function onPriceSyncConnectorChange(value: number) {
  const row = rows.value.find((item) => item.id === value);
  if (!row) {
    return;
  }
  activeConnector.value = row;
  priceSyncForm.upstreamCategoryId = 'all';
  priceSyncForm.priceMode = normalizePriceMode(row.priceMode);
  priceSyncForm.priceValue = Number(row.priceValue || 1);
  priceSyncForm.priceRounding = normalizeRounding(row.priceRounding);
  priceSyncForm.offlineMissing = false;
  preview.value = null;
}

async function pullPriceSyncCategories() {
  if (!priceSyncForm.connectorId) {
    message.error('请选择货源');
    return;
  }
  pullingPriceCategories.value = true;
  try {
    preview.value = await pullWK29Classes({
      connectorId: priceSyncForm.connectorId,
      skipExisting: true,
    });
    message.success(`已拉取 ${preview.value.totalCategories} 个分类`);
  } catch (error) {
    message.error(error instanceof Error ? error.message : '分类拉取失败');
  } finally {
    pullingPriceCategories.value = false;
  }
}

async function runPriceSync() {
  if (!priceSyncForm.connectorId) {
    message.error('请选择货源');
    return;
  }
  priceSyncing.value = true;
  try {
    const result = await syncWK29Prices({
      connectorId: priceSyncForm.connectorId,
      upstreamCategoryId: priceSyncForm.upstreamCategoryId,
      priceMode: normalizePriceMode(priceSyncForm.priceMode),
      priceValue: Number(priceSyncForm.priceValue || 0),
      priceRounding: normalizeRounding(priceSyncForm.priceRounding),
      offlineMissing: priceSyncForm.upstreamCategoryId === 'all' && priceSyncForm.offlineMissing,
    });
    message.success(`价格同步完成：匹配 ${result.total}，更新 ${result.updated}，缺失 ${result.missing}，下架 ${result.offlined}`);
    priceSyncOpen.value = false;
    await load();
  } catch (error) {
    message.error(error instanceof Error ? error.message : '价格同步失败');
  } finally {
    priceSyncing.value = false;
  }
}

async function runOrderSync(row: ConnectorRow) {
  if (row.kind !== '29wk') {
    message.info('当前仅 29 通用货源支持订单同步');
    return;
  }
  setOrderSyncing(row.id, true);
  try {
    const result = await syncWK29Orders({ connectorId: row.id });
    message.success(
      `订单同步完成：拉取 ${result.fetched}，匹配 ${result.matched}，更新 ${result.updated}，跳过 ${result.skipped}，失败 ${result.failed}`,
    );
    await load();
  } catch (error) {
    message.error(error instanceof Error ? error.message : '订单同步失败');
  } finally {
    setOrderSyncing(row.id, false);
  }
}

function setOrderSyncing(id: number, active: boolean) {
  const next = new Set(orderSyncingIds.value);
  if (active) {
    next.add(id);
  } else {
    next.delete(id);
  }
  orderSyncingIds.value = next;
}

async function saveConfig(row: ConnectorRow) {
  await updateConnector(row.id, {
    ...payloadFromConnector(row),
    orderSyncEnabled: configForm.orderSyncEnabled,
    sourceSyncEnabled: configForm.sourceSyncEnabled,
    priceMode: configForm.priceMode,
    priceValue: configForm.priceValue || 0,
    priceRounding: configForm.priceRounding,
    replaceRulesJson: JSON.stringify(configForm.replaceRules.filter((rule) => rule.from.trim())),
    categoryPriceRulesJson: JSON.stringify(categoryRuleMap()),
  });
}

function loadConfigForm(row: ConnectorRow) {
  configForm.orderSyncEnabled = row.orderSyncEnabled !== false;
  configForm.sourceSyncEnabled = row.sourceSyncEnabled !== false;
  configForm.priceMode = normalizePriceMode(row.priceMode);
  configForm.priceValue = Number(row.priceValue || 1);
  configForm.priceRounding = normalizeRounding(row.priceRounding);
  configForm.replaceRules = parseReplaceRules(row.replaceRulesJson);
  configForm.categoryRules = Object.entries(parseCategoryRules(row.categoryPriceRulesJson)).map(([upstreamId, rule]) => ({
    upstreamId,
    priceMode: normalizePriceMode(rule.priceMode),
    priceValue: Number(rule.priceValue || 1),
    priceRounding: normalizeRounding(rule.priceRounding),
  }));
}

function addReplaceRule() {
  configForm.replaceRules.push({ from: '', to: '' });
}

function removeReplaceRule(index: number) {
  configForm.replaceRules.splice(index, 1);
}

function addCategoryRule() {
  configForm.categoryRules.push({ upstreamId: '', priceMode: 'multiplier', priceValue: 1, priceRounding: 'none' });
}

function removeCategoryRule(index: number) {
  configForm.categoryRules.splice(index, 1);
}

function syncCategoryRulesFromPreview(categories: WK29CategoryPreview[]) {
  const existing = new Set(configForm.categoryRules.map((item) => item.upstreamId));
  for (const category of categories) {
    if (!existing.has(category.upstreamId)) {
      configForm.categoryRules.push({
        upstreamId: category.upstreamId,
        priceMode: normalizePriceMode(configForm.priceMode),
        priceValue: Number(configForm.priceValue || 1),
        priceRounding: normalizeRounding(configForm.priceRounding),
      });
      existing.add(category.upstreamId);
    }
  }
}

function categoryRuleMap() {
  return configForm.categoryRules.reduce<Record<string, ConnectorCategoryPriceRule>>((result, rule) => {
    const key = rule.upstreamId.trim();
    if (!key) {
      return result;
    }
    result[key] = {
      priceMode: normalizePriceMode(rule.priceMode),
      priceValue: Number(rule.priceValue || 0),
      priceRounding: normalizeRounding(rule.priceRounding),
    };
    return result;
  }, {});
}

function connectorPayloadFromForm() {
  return {
    name: form.name.trim(),
    baseUrl: form.baseUrl.trim(),
    appKey: form.appKey.trim(),
    appSecret: form.appSecret.trim(),
    kind: kindFromAccessType(form.accessType),
    status: form.status,
    sortOrder: form.sortOrder || 0,
    timeoutMs: form.timeoutMs || 8000,
  };
}

function payloadFromConnector(row: ConnectorRow) {
  return {
    name: row.name,
    baseUrl: row.baseUrl,
    appKey: row.appKey,
    appSecret: '',
    kind: row.kind,
    status: row.status,
    timeoutMs: row.timeoutMs || 8000,
    sortOrder: row.sortOrder || 10,
    ...configPayload(row),
  };
}

function configPayload(row: ConnectorRow) {
  return {
    orderSyncEnabled: row.orderSyncEnabled !== false,
    sourceSyncEnabled: row.sourceSyncEnabled !== false,
    priceMode: normalizePriceMode(row.priceMode),
    priceValue: Number(row.priceValue || 1),
    priceRounding: normalizeRounding(row.priceRounding),
    replaceRulesJson: row.replaceRulesJson || '[]',
    categoryPriceRulesJson: row.categoryPriceRulesJson || '{}',
  };
}

function defaultConfigPayload() {
  return {
    orderSyncEnabled: true,
    sourceSyncEnabled: true,
    priceMode: 'multiplier',
    priceValue: 1,
    priceRounding: 'none',
    replaceRulesJson: '[]',
    categoryPriceRulesJson: '{}',
  };
}

function validateConnectorPayload(payload: ReturnType<typeof connectorPayloadFromForm>) {
  if (!payload.name) {
    message.error('接口名称不能为空');
    return false;
  }
  if (payload.status === 'active' && !payload.baseUrl) {
    message.error('启用状态下必须填写接口地址');
    return false;
  }
  if (payload.baseUrl && !isHttpUrl(payload.baseUrl)) {
    message.error('接口地址必须是有效的 http 或 https URL');
    return false;
  }
  if (payload.timeoutMs < 0 || payload.timeoutMs > MAX_TIMEOUT_MS) {
    message.error(`超时时间必须在 0 到 ${MAX_TIMEOUT_MS} 毫秒之间`);
    return false;
  }
  return true;
}

function withConnectorDefaults(row: ConnectorRow): ConnectorRow {
  return {
    ...row,
    sortOrder: row.sortOrder ?? 10,
    orderSyncEnabled: row.orderSyncEnabled ?? true,
    sourceSyncEnabled: row.sourceSyncEnabled ?? true,
    priceMode: row.priceMode || 'multiplier',
    priceValue: row.priceValue || 1,
    priceRounding: row.priceRounding || 'none',
    replaceRulesJson: row.replaceRulesJson || '[]',
    categoryPriceRulesJson: row.categoryPriceRulesJson || '{}',
  };
}

function parseReplaceRules(raw: string): ConnectorReplaceRule[] {
  try {
    const parsed = JSON.parse(raw || '[]') as ConnectorReplaceRule[];
    return Array.isArray(parsed) ? parsed.map((item) => ({ from: item.from || '', to: item.to || '' })) : [];
  } catch {
    return [];
  }
}

function parseCategoryRules(raw: string): Record<string, ConnectorCategoryPriceRule> {
  try {
    const parsed = JSON.parse(raw || '{}') as Record<string, ConnectorCategoryPriceRule>;
    return parsed && typeof parsed === 'object' && !Array.isArray(parsed) ? parsed : {};
  } catch {
    return {};
  }
}

function accessTypeFromKind(kind: string): ConnectorForm['accessType'] {
  return kind === '29wk' ? '29_common' : 'common';
}

function kindFromAccessType(value: ConnectorForm['accessType']) {
  return value === '29_common' ? '29wk' : 'generic';
}

function accessLabel(kind: string) {
  return kind === '29wk' ? '29通用' : '常见货源';
}

function priceRuleText(row: ConnectorRow) {
  const mode = normalizePriceMode(row.priceMode) === 'fixed_add' ? '加' : 'x';
  return `${mode}${formatMoney(Number(row.priceValue || 1))} / ${roundingLabel(row.priceRounding)}`;
}

function normalizePriceMode(value: string) {
  return value === 'fixed_add' ? 'fixed_add' : 'multiplier';
}

function normalizeRounding(value: string) {
  return ['floor', 'ceil', 'round'].includes(value) ? value : 'none';
}

function roundingLabel(value: string) {
  return roundingOptions.find((item) => item.value === normalizeRounding(value))?.label || '不取整';
}

function productKey(categoryId: string, productId: string) {
  return `${categoryId}:${productId}`;
}

function formatMoney(value: number) {
  return Number(value || 0).toFixed(2);
}

function isHttpUrl(value: string) {
  try {
    const parsed = new URL(value);
    return parsed.protocol === 'http:' || parsed.protocol === 'https:';
  } catch {
    return false;
  }
}

onMounted(load);
</script>

<style scoped>
.connectors-page {
  display: grid;
  gap: 16px;
}

.connector-actions {
  flex-wrap: wrap;
}

.connector-form-grid,
.sync-config-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 14px;
}

.connector-form-grid--small {
  grid-template-columns: 160px minmax(0, 1fr);
}

.config-section {
  margin-top: 18px;
}

.section-title,
.pull-toolbar {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 12px;
}

.section-title {
  justify-content: space-between;
}

.replace-row {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(0, 1fr) 42px;
  gap: 10px;
  margin-bottom: 10px;
}

.source-select {
  width: 240px;
}

.pull-search {
  max-width: 340px;
}

.pull-summary {
  margin-bottom: 12px;
}

.price-sync-tip {
  margin-bottom: 16px;
}

.category-select-row {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 96px;
  gap: 8px;
}

@media (max-width: 900px) {
  .connector-form-grid,
  .connector-form-grid--small,
  .sync-config-grid,
  .replace-row,
  .category-select-row {
    grid-template-columns: 1fr;
  }

  .pull-toolbar {
    align-items: stretch;
    flex-direction: column;
  }

  .source-select,
  .pull-search {
    width: 100%;
    max-width: none;
  }
}
</style>
