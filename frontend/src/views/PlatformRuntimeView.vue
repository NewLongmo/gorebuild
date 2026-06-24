<template>
  <section class="page runtime-page">
    <div class="page-heading">
      <div>
        <h1>直跑插件</h1>
        <p>管理平台直跑插件、Worker 节点、代理池和远程控制命令。</p>
      </div>
      <a-button :loading="refreshing" @click="loadActive">刷新</a-button>
    </div>

    <a-tabs v-model:activeKey="activeTab" @change="loadActive">
      <a-tab-pane key="plugins" tab="插件">
        <DataToolbar v-model="pluginQuery.q" placeholder="搜索插件代码、名称或说明" @search="reloadPlugins">
          <div class="data-toolbar__filters">
            <a-select
              v-model:value="pluginQuery.status"
              allow-clear
              class="status-select"
              placeholder="状态"
              :options="pluginStatusOptions"
              @change="reloadPlugins"
            />
            <a-button type="primary" @click="openPluginCreate">新增插件</a-button>
          </div>
        </DataToolbar>

        <a-table
          row-key="code"
          :columns="pluginColumns"
          :data-source="plugins"
          :loading="pluginsLoading"
          :pagination="pluginPagination"
          :scroll="{ x: 1180 }"
          @change="handlePluginTableChange"
        >
          <template #bodyCell="{ column, record }">
            <template v-if="column.key === 'code'">
              <a-tag>plugin:{{ record.code }}</a-tag>
            </template>
            <template v-else-if="column.key === 'status'">
              <a-tag :color="record.status === 'active' ? 'green' : 'default'">
                {{ record.status === 'active' ? '启用' : '停用' }}
              </a-tag>
            </template>
            <template v-else-if="column.key === 'supports'">
              <a-tag :color="record.supportsQuery ? 'blue' : 'default'">查课{{ record.supportsQuery ? '开' : '关' }}</a-tag>
              <a-tag :color="record.supportsSubmit ? 'green' : 'default'">下单{{ record.supportsSubmit ? '开' : '关' }}</a-tag>
              <a-tag :color="record.supportsRefresh ? 'purple' : 'default'">刷新{{ record.supportsRefresh ? '开' : '关' }}</a-tag>
            </template>
            <template v-else-if="column.key === 'concurrency'">
              <span>{{ record.maxConcurrency }} 并发</span>
              <div class="muted">{{ record.accountSerial ? '同账号串行' : '同账号可并行' }}</div>
            </template>
            <template v-else-if="column.key === 'config'">
              <pre class="compact-json">{{ prettyJSON(record.configJson) }}</pre>
            </template>
            <template v-else-if="column.key === 'actions'">
              <div class="table-actions">
                <a-button size="small" @click="openPluginEdit(record)">编辑</a-button>
                <a-popconfirm title="确定删除该插件配置？不会删除历史订单。" @confirm="deletePlugin(record.code)">
                  <a-button size="small" danger>删除</a-button>
                </a-popconfirm>
              </div>
            </template>
          </template>
        </a-table>
      </a-tab-pane>

      <a-tab-pane key="workers" tab="Worker">
        <div class="runtime-toolbar">
          <div class="muted">Worker 每 10 秒上报一次心跳；暂停只是不再接新任务，不会删除队列。</div>
          <div class="table-actions">
            <a-button :loading="commandSending === '*:pause_accept'" @click="sendGlobalCommand('pause_accept')">暂停全部</a-button>
            <a-button :loading="commandSending === '*:resume_accept'" @click="sendGlobalCommand('resume_accept')">恢复全部</a-button>
          </div>
        </div>
        <a-table row-key="workerId" :columns="workerColumns" :data-source="workers" :loading="workersLoading" :pagination="false" :scroll="{ x: 1180 }">
          <template #bodyCell="{ column, record }">
            <template v-if="column.key === 'status'">
              <a-tag :color="workerStatusColor(record.status)">{{ workerStatusText(record.status) }}</a-tag>
            </template>
            <template v-else-if="column.key === 'acceptNew'">
              <a-tag :color="record.acceptNew ? 'green' : 'orange'">{{ record.acceptNew ? '接单中' : '暂停接单' }}</a-tag>
            </template>
            <template v-else-if="column.key === 'current'">
              <span>{{ record.currentOrderId || '-' }}</span>
              <div v-if="record.currentPluginCode" class="muted">plugin:{{ record.currentPluginCode }}</div>
            </template>
            <template v-else-if="column.key === 'heartbeat'">
              <span>{{ formatTime(record.heartbeatAt) }}</span>
              <div class="muted">{{ heartbeatAge(record.heartbeatAt) }}</div>
            </template>
            <template v-else-if="column.key === 'actions'">
              <div class="table-actions">
                <a-button
                  size="small"
                  :disabled="!record.acceptNew"
                  :loading="commandSending === `${record.workerId}:pause_accept`"
                  @click="commandWorker(record.workerId, 'pause_accept')"
                >
                  暂停
                </a-button>
                <a-button
                  size="small"
                  :disabled="record.acceptNew"
                  :loading="commandSending === `${record.workerId}:resume_accept`"
                  @click="commandWorker(record.workerId, 'resume_accept')"
                >
                  恢复
                </a-button>
                <a-button size="small" danger :loading="commandSending === `${record.workerId}:stop`" @click="commandWorker(record.workerId, 'stop')">
                  停止接单
                </a-button>
              </div>
            </template>
          </template>
        </a-table>
      </a-tab-pane>

      <a-tab-pane key="proxies" tab="代理池">
        <DataToolbar v-model="proxyQuery.q" placeholder="搜索代理名称或地址" @search="reloadProxies">
          <div class="data-toolbar__filters">
            <a-select
              v-model:value="proxyQuery.status"
              allow-clear
              class="status-select"
              placeholder="状态"
              :options="proxyStatusOptions"
              @change="reloadProxies"
            />
            <a-button type="primary" @click="openProxyCreate">新增代理</a-button>
          </div>
        </DataToolbar>

        <a-table
          row-key="id"
          :columns="proxyColumns"
          :data-source="proxies"
          :loading="proxiesLoading"
          :pagination="proxyPagination"
          :scroll="{ x: 1180 }"
          @change="handleProxyTableChange"
        >
          <template #bodyCell="{ column, record }">
            <template v-if="column.key === 'url'">
              <code>{{ record.maskedUrl || '-' }}</code>
            </template>
            <template v-else-if="column.key === 'status'">
              <a-tag :color="record.status === 'active' ? 'green' : 'default'">
                {{ record.status === 'active' ? '启用' : '停用' }}
              </a-tag>
            </template>
            <template v-else-if="column.key === 'usage'">
              <span>{{ record.inUseCount }}/{{ record.maxConcurrency }}</span>
              <div class="muted">总 {{ record.useCount }} 成功 {{ record.successCount }} 失败 {{ record.failCount }}</div>
            </template>
            <template v-else-if="column.key === 'error'">
              <span :class="{ danger: record.lastError }">{{ record.lastError || '-' }}</span>
            </template>
            <template v-else-if="column.key === 'actions'">
              <div class="table-actions">
                <a-button size="small" @click="openProxyEdit(record)">编辑</a-button>
                <a-popconfirm title="确定删除这个代理？正在执行的任务不会被删除。" @confirm="deleteProxy(record.id)">
                  <a-button size="small" danger>删除</a-button>
                </a-popconfirm>
              </div>
            </template>
          </template>
        </a-table>
      </a-tab-pane>

      <a-tab-pane key="commands" tab="命令记录">
        <DataToolbar v-model="commandQuery.workerId" placeholder="按 Worker ID 过滤" search-text="过滤" @search="reloadCommands">
          <a-button @click="reloadCommands">刷新</a-button>
        </DataToolbar>
        <a-table
          row-key="id"
          :columns="commandColumns"
          :data-source="commands"
          :loading="commandsLoading"
          :pagination="commandPagination"
          :scroll="{ x: 980 }"
          @change="handleCommandTableChange"
        >
          <template #bodyCell="{ column, record }">
            <template v-if="column.key === 'command'">
              <a-tag>{{ commandText(record.command) }}</a-tag>
            </template>
            <template v-else-if="column.key === 'status'">
              <a-tag :color="record.status === 'done' ? 'green' : record.status === 'failed' ? 'red' : 'processing'">
                {{ commandStatusText(record.status) }}
              </a-tag>
            </template>
          </template>
        </a-table>
      </a-tab-pane>
    </a-tabs>

    <a-modal v-model:open="pluginModalOpen" :title="editingPluginCode ? '编辑插件' : '新增插件'" :confirm-loading="pluginSaving" @ok="savePlugin">
      <a-form layout="vertical">
        <a-form-item label="插件代码" required>
          <a-input v-model:value="pluginForm.code" :disabled="Boolean(editingPluginCode)" placeholder="如 manual，不需要写 plugin:" />
        </a-form-item>
        <a-form-item label="插件名称" required>
          <a-input v-model:value="pluginForm.name" maxlength="120" />
        </a-form-item>
        <a-form-item label="说明">
          <a-textarea v-model:value="pluginForm.description" :rows="3" maxlength="500" />
        </a-form-item>
        <div class="runtime-form-grid">
          <a-form-item label="状态">
            <a-select v-model:value="pluginForm.status" :options="pluginStatusOptions" />
          </a-form-item>
          <a-form-item label="排序">
            <a-input-number v-model:value="pluginForm.sortOrder" :min="0" :max="100000" class="full-input" />
          </a-form-item>
          <a-form-item label="最大并发">
            <a-input-number v-model:value="pluginForm.maxConcurrency" :min="1" :max="100" class="full-input" />
          </a-form-item>
          <a-form-item label="同账号串行">
            <a-switch v-model:checked="pluginForm.accountSerial" />
          </a-form-item>
        </div>
        <div class="runtime-switch-row">
          <a-checkbox v-model:checked="pluginForm.supportsQuery">支持查课</a-checkbox>
          <a-checkbox v-model:checked="pluginForm.supportsSubmit">支持下单</a-checkbox>
          <a-checkbox v-model:checked="pluginForm.supportsRefresh">支持刷新/补刷</a-checkbox>
        </div>
        <a-form-item label="配置 JSON">
          <a-textarea v-model:value="pluginForm.configJson" :rows="6" />
        </a-form-item>
      </a-form>
    </a-modal>

    <a-modal v-model:open="proxyModalOpen" :title="editingProxyId ? '编辑代理' : '新增代理'" :confirm-loading="proxySaving" @ok="saveProxy">
      <a-form layout="vertical">
        <a-form-item label="名称">
          <a-input v-model:value="proxyForm.name" maxlength="120" />
        </a-form-item>
        <a-form-item :label="editingProxyId ? '代理地址（留空不覆盖）' : '代理地址'" :required="!editingProxyId">
          <a-input-password v-model:value="proxyForm.proxyUrl" placeholder="http://user:pass@host:port 或 socks5://host:port" />
        </a-form-item>
        <div class="runtime-form-grid">
          <a-form-item label="类型">
            <a-select v-model:value="proxyForm.kind" :options="proxyKindOptions" />
          </a-form-item>
          <a-form-item label="状态">
            <a-select v-model:value="proxyForm.status" :options="proxyStatusOptions" />
          </a-form-item>
          <a-form-item label="最大并发">
            <a-input-number v-model:value="proxyForm.maxConcurrency" :min="1" :max="100" class="full-input" />
          </a-form-item>
        </div>
      </a-form>
    </a-modal>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue';
import { message } from 'ant-design-vue';
import DataToolbar from '@/components/DataToolbar.vue';
import {
  createPlatformPlugin,
  createWorkerProxy,
  fetchPlatformPlugins,
  fetchWorkerCommands,
  fetchWorkerNodes,
  fetchWorkerProxies,
  removePlatformPlugin,
  removeWorkerProxy,
  sendWorkerCommand,
  updatePlatformPlugin,
  updateWorkerProxy,
  type PlatformPluginRow,
  type WorkerCommandRow,
  type WorkerNodeRow,
  type WorkerProxyRow,
} from '@/api/admin';

const activeTab = ref('plugins');
const refreshing = ref(false);

const plugins = ref<PlatformPluginRow[]>([]);
const workers = ref<WorkerNodeRow[]>([]);
const proxies = ref<WorkerProxyRow[]>([]);
const commands = ref<WorkerCommandRow[]>([]);

const pluginsLoading = ref(false);
const workersLoading = ref(false);
const proxiesLoading = ref(false);
const commandsLoading = ref(false);
const pluginSaving = ref(false);
const proxySaving = ref(false);
const pluginModalOpen = ref(false);
const proxyModalOpen = ref(false);
const editingPluginCode = ref('');
const editingProxyId = ref<number | null>(null);
const commandSending = ref('');

const pluginTotal = ref(0);
const proxyTotal = ref(0);
const commandTotal = ref(0);

const pluginQuery = reactive({ q: '', status: undefined as string | undefined, page: 1, perPage: 20 });
const proxyQuery = reactive({ q: '', status: undefined as string | undefined, page: 1, perPage: 20 });
const commandQuery = reactive({ workerId: '', page: 1, perPage: 20 });

const pluginForm = reactive({
  code: '',
  name: '',
  description: '',
  status: 'disabled',
  sortOrder: 10,
  supportsQuery: false,
  supportsSubmit: true,
  supportsRefresh: true,
  maxConcurrency: 1,
  accountSerial: true,
  configJson: '{}',
});

const proxyForm = reactive({
  name: '',
  proxyUrl: '',
  kind: 'http',
  status: 'active',
  maxConcurrency: 1,
});

const pluginStatusOptions = [
  { label: '启用', value: 'active' },
  { label: '停用', value: 'disabled' },
];

const proxyStatusOptions = [
  { label: '启用', value: 'active' },
  { label: '停用', value: 'disabled' },
];

const proxyKindOptions = [
  { label: 'HTTP', value: 'http' },
  { label: 'SOCKS5', value: 'socks5' },
];

const pluginColumns = [
  { title: '代码', key: 'code', width: 150 },
  { title: '名称', dataIndex: 'name', width: 180 },
  { title: '说明', dataIndex: 'description', width: 260 },
  { title: '能力', key: 'supports', width: 220 },
  { title: '并发', key: 'concurrency', width: 140 },
  { title: '配置', key: 'config', width: 260 },
  { title: '状态', key: 'status', width: 90 },
  { title: '更新时间', dataIndex: 'updatedAt', width: 180 },
  { title: '操作', key: 'actions', width: 150 },
];

const workerColumns = [
  { title: 'Worker ID', dataIndex: 'workerId', width: 240 },
  { title: '主机', dataIndex: 'hostname', width: 160 },
  { title: '状态', key: 'status', width: 100 },
  { title: '接单', key: 'acceptNew', width: 110 },
  { title: '并发', dataIndex: 'maxConcurrency', width: 80 },
  { title: '运行中', dataIndex: 'runningCount', width: 90 },
  { title: '当前订单', key: 'current', width: 130 },
  { title: '消息', dataIndex: 'message', width: 200 },
  { title: '心跳', key: 'heartbeat', width: 220 },
  { title: '操作', key: 'actions', width: 230 },
];

const proxyColumns = [
  { title: 'ID', dataIndex: 'id', width: 70 },
  { title: '名称', dataIndex: 'name', width: 160 },
  { title: '代理地址', key: 'url', width: 260 },
  { title: '类型', dataIndex: 'kind', width: 90 },
  { title: '状态', key: 'status', width: 90 },
  { title: '使用情况', key: 'usage', width: 180 },
  { title: '最近使用', dataIndex: 'lastUsedAt', width: 180 },
  { title: '最近错误', key: 'error', width: 260 },
  { title: '操作', key: 'actions', width: 140 },
];

const commandColumns = [
  { title: 'ID', dataIndex: 'id', width: 80 },
  { title: 'Worker ID', dataIndex: 'workerId', width: 240 },
  { title: '命令', key: 'command', width: 130 },
  { title: '状态', key: 'status', width: 110 },
  { title: '结果', dataIndex: 'result' },
  { title: '创建时间', dataIndex: 'createdAt', width: 180 },
  { title: '执行时间', dataIndex: 'executedAt', width: 180 },
];

const pluginPagination = computed(() => ({
  current: pluginQuery.page,
  pageSize: pluginQuery.perPage,
  total: pluginTotal.value,
  showSizeChanger: true,
}));

const proxyPagination = computed(() => ({
  current: proxyQuery.page,
  pageSize: proxyQuery.perPage,
  total: proxyTotal.value,
  showSizeChanger: true,
}));

const commandPagination = computed(() => ({
  current: commandQuery.page,
  pageSize: commandQuery.perPage,
  total: commandTotal.value,
  showSizeChanger: true,
}));

async function loadActive() {
  refreshing.value = true;
  try {
    if (activeTab.value === 'plugins') {
      await loadPlugins();
    } else if (activeTab.value === 'workers') {
      await loadWorkers();
    } else if (activeTab.value === 'proxies') {
      await loadProxies();
    } else {
      await loadCommands();
    }
  } finally {
    refreshing.value = false;
  }
}

async function loadPlugins() {
  pluginsLoading.value = true;
  try {
    const data = await fetchPlatformPlugins({ ...pluginQuery });
    plugins.value = data.items;
    pluginTotal.value = data.total;
  } catch (error) {
    message.error(error instanceof Error ? error.message : '插件加载失败');
  } finally {
    pluginsLoading.value = false;
  }
}

async function loadWorkers() {
  workersLoading.value = true;
  try {
    const data = await fetchWorkerNodes();
    workers.value = data.items;
  } catch (error) {
    message.error(error instanceof Error ? error.message : 'Worker 加载失败');
  } finally {
    workersLoading.value = false;
  }
}

async function loadProxies() {
  proxiesLoading.value = true;
  try {
    const data = await fetchWorkerProxies({ ...proxyQuery });
    proxies.value = data.items;
    proxyTotal.value = data.total;
  } catch (error) {
    message.error(error instanceof Error ? error.message : '代理池加载失败');
  } finally {
    proxiesLoading.value = false;
  }
}

async function loadCommands() {
  commandsLoading.value = true;
  try {
    const data = await fetchWorkerCommands({ ...commandQuery });
    commands.value = data.items;
    commandTotal.value = data.total;
  } catch (error) {
    message.error(error instanceof Error ? error.message : '命令记录加载失败');
  } finally {
    commandsLoading.value = false;
  }
}

function reloadPlugins() {
  pluginQuery.page = 1;
  void loadPlugins();
}

function reloadProxies() {
  proxyQuery.page = 1;
  void loadProxies();
}

function reloadCommands() {
  commandQuery.page = 1;
  void loadCommands();
}

function handlePluginTableChange(next: { current?: number; pageSize?: number }) {
  pluginQuery.page = next.current || 1;
  pluginQuery.perPage = next.pageSize || 20;
  void loadPlugins();
}

function handleProxyTableChange(next: { current?: number; pageSize?: number }) {
  proxyQuery.page = next.current || 1;
  proxyQuery.perPage = next.pageSize || 20;
  void loadProxies();
}

function handleCommandTableChange(next: { current?: number; pageSize?: number }) {
  commandQuery.page = next.current || 1;
  commandQuery.perPage = next.pageSize || 20;
  void loadCommands();
}

function openPluginCreate() {
  editingPluginCode.value = '';
  Object.assign(pluginForm, {
    code: '',
    name: '',
    description: '',
    status: 'disabled',
    sortOrder: 10,
    supportsQuery: false,
    supportsSubmit: true,
    supportsRefresh: true,
    maxConcurrency: 1,
    accountSerial: true,
    configJson: '{}',
  });
  pluginModalOpen.value = true;
}

function openPluginEdit(row: PlatformPluginRow) {
  editingPluginCode.value = row.code;
  Object.assign(pluginForm, {
    code: row.code,
    name: row.name,
    description: row.description,
    status: row.status,
    sortOrder: row.sortOrder || 10,
    supportsQuery: row.supportsQuery,
    supportsSubmit: row.supportsSubmit,
    supportsRefresh: row.supportsRefresh,
    maxConcurrency: row.maxConcurrency || 1,
    accountSerial: row.accountSerial,
    configJson: row.configJson || '{}',
  });
  pluginModalOpen.value = true;
}

async function savePlugin() {
  const payload = pluginPayload();
  if (!payload) {
    return;
  }
  pluginSaving.value = true;
  try {
    if (editingPluginCode.value) {
      await updatePlatformPlugin(editingPluginCode.value, payload);
    } else {
      await createPlatformPlugin(payload);
    }
    pluginModalOpen.value = false;
    message.success('插件已保存');
    await loadPlugins();
  } catch (error) {
    message.error(error instanceof Error ? error.message : '插件保存失败');
  } finally {
    pluginSaving.value = false;
  }
}

function pluginPayload() {
  const code = normalizePluginCode(pluginForm.code);
  if (!code) {
    message.error('插件代码不能为空');
    return null;
  }
  if (!pluginForm.name.trim()) {
    message.error('插件名称不能为空');
    return null;
  }
  if (!isJSON(pluginForm.configJson)) {
    message.error('配置 JSON 格式不正确');
    return null;
  }
  return {
    code,
    name: pluginForm.name.trim(),
    description: pluginForm.description.trim(),
    status: pluginForm.status,
    sortOrder: pluginForm.sortOrder || 0,
    supportsQuery: pluginForm.supportsQuery,
    supportsSubmit: pluginForm.supportsSubmit,
    supportsRefresh: pluginForm.supportsRefresh,
    maxConcurrency: pluginForm.maxConcurrency || 1,
    accountSerial: pluginForm.accountSerial,
    configJson: pluginForm.configJson.trim() || '{}',
  };
}

async function deletePlugin(code: string) {
  try {
    await removePlatformPlugin(code);
    message.success('插件配置已删除');
    await loadPlugins();
  } catch (error) {
    message.error(error instanceof Error ? error.message : '插件删除失败');
  }
}

function openProxyCreate() {
  editingProxyId.value = null;
  Object.assign(proxyForm, { name: '', proxyUrl: '', kind: 'http', status: 'active', maxConcurrency: 1 });
  proxyModalOpen.value = true;
}

function openProxyEdit(row: WorkerProxyRow) {
  editingProxyId.value = row.id;
  Object.assign(proxyForm, {
    name: row.name,
    proxyUrl: '',
    kind: row.kind || 'http',
    status: row.status || 'active',
    maxConcurrency: row.maxConcurrency || 1,
  });
  proxyModalOpen.value = true;
}

async function saveProxy() {
  const payload = proxyPayload();
  if (!payload) {
    return;
  }
  proxySaving.value = true;
  try {
    if (editingProxyId.value) {
      await updateWorkerProxy(editingProxyId.value, payload);
    } else {
      await createWorkerProxy(payload);
    }
    proxyModalOpen.value = false;
    message.success('代理已保存');
    await loadProxies();
  } catch (error) {
    message.error(error instanceof Error ? error.message : '代理保存失败');
  } finally {
    proxySaving.value = false;
  }
}

function proxyPayload() {
  const proxyUrl = proxyForm.proxyUrl.trim();
  if (!editingProxyId.value && !proxyUrl) {
    message.error('代理地址不能为空');
    return null;
  }
  if (proxyUrl && !isProxyURL(proxyUrl)) {
    message.error('代理地址必须以 http://、https:// 或 socks5:// 开头');
    return null;
  }
  return {
    name: proxyForm.name.trim(),
    proxyUrl,
    kind: proxyForm.kind,
    status: proxyForm.status,
    maxConcurrency: proxyForm.maxConcurrency || 1,
  };
}

async function deleteProxy(id: number) {
  try {
    await removeWorkerProxy(id);
    message.success('代理已删除');
    await loadProxies();
  } catch (error) {
    message.error(error instanceof Error ? error.message : '代理删除失败');
  }
}

function sendGlobalCommand(command: string) {
  void commandWorker('*', command);
}

async function commandWorker(workerId: string, command: string) {
  commandSending.value = `${workerId}:${command}`;
  try {
    await sendWorkerCommand(workerId, command);
    message.success('命令已下发');
    await Promise.all([loadWorkers(), loadCommands()]);
  } catch (error) {
    message.error(error instanceof Error ? error.message : '命令下发失败');
  } finally {
    commandSending.value = '';
  }
}

function normalizePluginCode(value: string) {
  return value.trim().toLowerCase().replace(/^plugin:/, '');
}

function isJSON(value: string) {
  try {
    JSON.parse(value || '{}');
    return true;
  } catch {
    return false;
  }
}

function prettyJSON(value: string) {
  if (!value) return '{}';
  try {
    return JSON.stringify(JSON.parse(value), null, 2);
  } catch {
    return value;
  }
}

function isProxyURL(value: string) {
  return /^(https?|socks5):\/\//i.test(value.trim());
}

function workerStatusColor(status: string) {
  if (status === 'running') return 'green';
  if (status === 'stopped') return 'default';
  if (status === 'timeout' || status === 'failed') return 'red';
  return 'processing';
}

function workerStatusText(status: string) {
  const labels: Record<string, string> = {
    running: '运行中',
    stopped: '已停止',
    starting: '启动中',
    timeout: '心跳超时',
    failed: '失败',
  };
  return labels[status] || status || '未知';
}

function commandText(command: string) {
  const labels: Record<string, string> = {
    pause_accept: '暂停接单',
    resume_accept: '恢复接单',
    stop: '停止接单',
  };
  return labels[command] || command;
}

function commandStatusText(status: string) {
  const labels: Record<string, string> = {
    pending: '待执行',
    done: '已执行',
    failed: '失败',
  };
  return labels[status] || status || '待执行';
}

function formatTime(value: string | null) {
  return value || '-';
}

function heartbeatAge(value: string | null) {
  if (!value) {
    return '无心跳';
  }
  const timestamp = new Date(value).getTime();
  if (Number.isNaN(timestamp)) {
    return '时间未知';
  }
  const seconds = Math.max(0, Math.round((Date.now() - timestamp) / 1000));
  if (seconds < 60) {
    return `${seconds} 秒前`;
  }
  return `${Math.round(seconds / 60)} 分钟前`;
}

onMounted(() => {
  void Promise.all([loadPlugins(), loadWorkers(), loadProxies(), loadCommands()]);
});
</script>

<style scoped>
.runtime-page {
  display: grid;
  gap: 16px;
}

.runtime-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 16px;
}

.runtime-form-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12px;
}

.runtime-switch-row {
  display: flex;
  flex-wrap: wrap;
  gap: 14px;
  margin: 8px 0 18px;
}

.compact-json {
  max-width: 260px;
  max-height: 120px;
  margin: 0;
  overflow: auto;
  white-space: pre-wrap;
}

.muted {
  color: #667085;
  font-size: 12px;
}

.danger {
  color: #cf1322;
}

code {
  overflow-wrap: anywhere;
}

@media (max-width: 900px) {
  .runtime-toolbar,
  .runtime-form-grid {
    align-items: stretch;
    grid-template-columns: 1fr;
  }

  .runtime-toolbar {
    flex-direction: column;
  }
}
</style>
