<template>
  <section class="page">
    <div class="page-heading">
      <div>
        <h1>账号安全</h1>
        <p>修改登录密码并查看账号流水与操作记录。</p>
      </div>
    </div>

    <a-row :gutter="[16, 16]">
      <a-col :xs="24" :lg="8">
        <a-card :bordered="false" title="修改密码">
          <a-form layout="vertical" :model="passwordForm">
            <a-form-item label="当前密码" required>
              <a-input-password v-model:value="passwordForm.currentPassword" autocomplete="current-password" />
            </a-form-item>
            <a-form-item label="新密码" required>
              <a-input-password v-model:value="passwordForm.newPassword" autocomplete="new-password" />
            </a-form-item>
            <a-form-item label="确认新密码" required>
              <a-input-password v-model:value="passwordForm.confirmPassword" autocomplete="new-password" />
            </a-form-item>
            <a-button type="primary" block :loading="savingPassword" @click="savePassword">更新密码</a-button>
          </a-form>
        </a-card>

        <a-card :bordered="false" title="API 接口" class="account-api-card">
          <a-alert
            type="info"
            show-icon
            message="旧版接口兼容"
            description="外部系统可继续用 uid + key 调用 api.php?act=getmoney/getclass/add/uporder。"
          />
          <div class="api-key-box">
            <span>UID</span>
            <strong>{{ profile?.id || '-' }}</strong>
          </div>
          <div class="api-key-box">
            <span>Key</span>
            <strong>{{ profile?.apiKey || '未开通' }}</strong>
          </div>
          <div class="heading-actions">
            <a-button type="primary" :loading="savingApiKey" @click="regenerateApiKey">
              {{ profile?.apiKey ? '更换 Key' : '开通接口' }}
            </a-button>
            <a-popconfirm title="确定关闭 API 接口？关闭后旧 Key 立即失效。" @confirm="disableApiKey">
              <a-button danger :disabled="!profile?.apiKey" :loading="savingApiKey">关闭接口</a-button>
            </a-popconfirm>
          </div>
        </a-card>

        <a-card :bordered="false" title="代理邀请" class="account-api-card">
          <a-form layout="vertical" :model="inviteForm">
            <a-form-item label="我的邀请码">
              <a-input v-model:value="inviteForm.code" placeholder="留空自动生成" />
            </a-form-item>
            <a-form-item label="下级默认倍率" required>
              <a-input-number v-model:value="inviteForm.priceRate" :min="profile?.priceRate || 0.0001" :step="0.01" class="full-input" />
            </a-form-item>
            <a-button type="primary" block :loading="savingInvite" @click="saveInvite">保存邀请设置</a-button>
          </a-form>
        </a-card>

        <a-card :bordered="false" title="我的代理公告" class="account-api-card">
          <a-textarea v-model:value="notice" :rows="5" maxlength="5000" />
          <a-button class="notice-save-button" type="primary" block :loading="savingNotice" @click="saveNotice">保存公告</a-button>
        </a-card>
      </a-col>

      <a-col :xs="24" :lg="16">
        <a-card :bordered="false" title="账号记录">
          <DataToolbar v-model="search" placeholder="搜索日志" @search="loadLogs">
            <div class="data-toolbar__filters">
              <a-button @click="loadLogs">刷新</a-button>
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
              <template v-if="column.key === 'amount'">
                <span :class="Number(record.amount) < 0 ? 'amount-negative' : 'amount-positive'">
                  {{ Number(record.amount).toFixed(2) }}
                </span>
              </template>
            </template>
          </a-table>
        </a-card>
      </a-col>
    </a-row>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue';
import { message } from 'ant-design-vue';
import DataToolbar from '@/components/DataToolbar.vue';
import {
  changeAgentPassword,
  disableAgentApiKey,
  fetchAgentLogs,
  fetchAgentProfile,
  regenerateAgentApiKey,
  updateAgentInvite,
  updateAgentNotice,
  type AgentProfile,
} from '@/api/user';
import type { LogRow } from '@/api/admin';

const savingPassword = ref(false);
const savingApiKey = ref(false);
const savingNotice = ref(false);
const savingInvite = ref(false);
const loading = ref(false);
const rows = ref<LogRow[]>([]);
const profile = ref<AgentProfile>();
const notice = ref('');
const search = ref('');
const page = ref(1);
const perPage = ref(10);
const total = ref(0);
const passwordForm = reactive({
  currentPassword: '',
  newPassword: '',
  confirmPassword: '',
});
const inviteForm = reactive({
  code: '',
  priceRate: 1,
});

const columns = [
  { title: '时间', dataIndex: 'createdAt', width: 190 },
  { title: '类型', dataIndex: 'type', width: 180 },
  { title: '内容', dataIndex: 'text' },
  { title: '金额', key: 'amount', width: 120 },
];

const pagination = computed(() => ({
  current: page.value,
  pageSize: perPage.value,
  total: total.value,
  showSizeChanger: true,
}));

async function savePassword() {
  if (!passwordForm.currentPassword || !passwordForm.newPassword) {
    message.error('当前密码和新密码不能为空');
    return;
  }
  if (passwordForm.newPassword.length < 8) {
    message.error('新密码至少需要 8 位');
    return;
  }
  if (passwordForm.newPassword !== passwordForm.confirmPassword) {
    message.error('两次输入的新密码不一致');
    return;
  }
  savingPassword.value = true;
  try {
    await changeAgentPassword(passwordForm.currentPassword, passwordForm.newPassword);
    Object.assign(passwordForm, { currentPassword: '', newPassword: '', confirmPassword: '' });
    message.success('密码已更新');
    await loadLogs();
  } catch (error) {
    message.error(error instanceof Error ? error.message : '密码更新失败');
  } finally {
    savingPassword.value = false;
  }
}

async function loadProfile() {
  try {
    profile.value = await fetchAgentProfile();
    notice.value = profile.value.notice || '';
    inviteForm.code = profile.value.inviteCode || '';
    inviteForm.priceRate = profile.value.invitePriceRate || profile.value.priceRate || 1;
  } catch (error) {
    message.error(error instanceof Error ? error.message : '账号信息加载失败');
  }
}

async function saveNotice() {
  if (notice.value.length > 5000) {
    message.error('公告内容不能超过 5000 字');
    return;
  }
  savingNotice.value = true;
  try {
    const result = await updateAgentNotice(notice.value);
    notice.value = result.notice;
    await loadProfile();
    await loadLogs();
    message.success('公告已保存');
  } catch (error) {
    message.error(error instanceof Error ? error.message : '公告保存失败');
  } finally {
    savingNotice.value = false;
  }
}

async function saveInvite() {
  const code = inviteForm.code.trim();
  if (code && !/^[A-Za-z0-9_-]{4,64}$/.test(code)) {
    message.error('邀请码必须为 4 到 64 位字母、数字、横线或下划线');
    return;
  }
  if (inviteForm.priceRate < (profile.value?.priceRate || 0)) {
    message.error('下级默认倍率不能低于自己的价格倍率');
    return;
  }
  savingInvite.value = true;
  try {
    const result = await updateAgentInvite(code, inviteForm.priceRate);
    inviteForm.code = result.inviteCode;
    inviteForm.priceRate = result.invitePriceRate;
    await loadProfile();
    await loadLogs();
    message.success('邀请设置已保存');
  } catch (error) {
    message.error(error instanceof Error ? error.message : '邀请设置保存失败');
  } finally {
    savingInvite.value = false;
  }
}

async function regenerateApiKey() {
  savingApiKey.value = true;
  try {
    const result = await regenerateAgentApiKey();
    await loadProfile();
    await loadLogs();
    message.success(`接口 Key 已更新：${result.apiKey}`);
  } catch (error) {
    message.error(error instanceof Error ? error.message : '接口 Key 更新失败');
  } finally {
    savingApiKey.value = false;
  }
}

async function disableApiKey() {
  savingApiKey.value = true;
  try {
    await disableAgentApiKey();
    await loadProfile();
    await loadLogs();
    message.success('API 接口已关闭');
  } catch (error) {
    message.error(error instanceof Error ? error.message : 'API 接口关闭失败');
  } finally {
    savingApiKey.value = false;
  }
}

async function loadLogs() {
  loading.value = true;
  try {
    const data = await fetchAgentLogs({ q: search.value, page: page.value, perPage: perPage.value });
    rows.value = data.items;
    total.value = data.total;
  } catch (error) {
    message.error(error instanceof Error ? error.message : '日志加载失败');
  } finally {
    loading.value = false;
  }
}

function handleTableChange(next: { current?: number; pageSize?: number }) {
  page.value = next.current || 1;
  perPage.value = next.pageSize || 10;
  void loadLogs();
}

onMounted(() => {
  void loadProfile();
  void loadLogs();
});
</script>
