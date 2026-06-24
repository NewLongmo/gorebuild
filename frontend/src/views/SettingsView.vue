<template>
  <section class="page">
    <div class="page-heading">
      <div>
        <h1>系统设置</h1>
        <p>维护站点名称、默认通道和运行时刷新策略。</p>
      </div>
      <a-button type="primary" :loading="saving" @click="save">保存</a-button>
    </div>

    <a-card :bordered="false">
      <a-form layout="vertical" :model="form">
        <a-form-item label="站点名称">
          <a-input v-model:value="form.site_name" />
        </a-form-item>
        <a-form-item label="全站公告">
          <a-textarea v-model:value="form.site_notice" :rows="5" maxlength="5000" />
        </a-form-item>
        <a-form-item label="弹窗公告">
          <a-textarea v-model:value="form.popup_notice" :rows="4" maxlength="5000" />
        </a-form-item>
        <a-form-item label="通知订阅地址">
          <a-input v-model:value="form.notice_url" placeholder="https://..." />
        </a-form-item>
        <a-form-item label="下单提示">
          <a-textarea v-model:value="form.order_tips" :rows="4" maxlength="5000" />
        </a-form-item>
        <a-form-item label="公开查单提示">
          <a-textarea v-model:value="form.open_query_notice" :rows="3" maxlength="2000" />
        </a-form-item>
        <a-form-item label="充值赠送规则 JSON">
          <a-textarea v-model:value="form.recharge_bonus_rules" :rows="4" placeholder='[{"min":300,"rate":1.1}]' />
        </a-form-item>
        <a-form-item label="控制台缓存秒数">
          <a-input-number v-model:value="dashboardCacheSeconds" :min="5" :max="3600" class="full-input" />
        </a-form-item>
        <a-form-item label="默认对接通道">
          <a-select
            v-model:value="form.default_connector_id"
            allow-clear
            placeholder="未选择时使用第一个启用通道"
            :options="connectorOptions"
            class="full-input"
          />
        </a-form-item>
        <a-form-item label="订单页自动刷新">
          <a-switch v-model:checked="orderAutoRefresh" />
        </a-form-item>
      </a-form>
    </a-card>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue';
import { message } from 'ant-design-vue';
import { fetchConnectors, fetchSettings, saveSettings } from '@/api/admin';

const saving = ref(false);
const connectorOptions = ref<{ label: string; value: string }[]>([]);
const form = reactive<Record<string, string>>({
  site_name: 'DW0RDWK',
  site_notice: '',
  popup_notice: '',
  notice_url: '',
  order_tips: '',
  open_query_notice: '',
  recharge_bonus_rules: '[]',
  dashboard_cache_seconds: '30',
  default_connector_id: '',
  order_auto_refresh: 'false',
});

const dashboardCacheSeconds = computed({
  get: () => Number(form.dashboard_cache_seconds || 30),
  set: (value: number) => {
    form.dashboard_cache_seconds = String(value);
  },
});

const orderAutoRefresh = computed({
  get: () => form.order_auto_refresh === 'true',
  set: (value: boolean) => {
    form.order_auto_refresh = String(value);
  },
});

async function load() {
  try {
    const [settings, connectors] = await Promise.all([
      fetchSettings(),
      fetchConnectors({ status: 'active', perPage: 100 }),
    ]);
    Object.assign(form, settings);
    connectorOptions.value = connectors.items
      .filter((item) => item.baseUrl)
      .map((item) => ({ label: `${item.name} (#${item.id})`, value: String(item.id) }));
  } catch (error) {
    message.error(error instanceof Error ? error.message : '设置加载失败');
  }
}

async function save() {
  form.site_name = form.site_name.trim();
  const cacheSeconds = Number(form.dashboard_cache_seconds);
  if (!form.site_name) {
    message.error('站点名称不能为空');
    return;
  }
  form.site_notice = (form.site_notice || '').trim();
  form.popup_notice = (form.popup_notice || '').trim();
  form.notice_url = (form.notice_url || '').trim();
  form.order_tips = (form.order_tips || '').trim();
  form.open_query_notice = (form.open_query_notice || '').trim();
  form.recharge_bonus_rules = (form.recharge_bonus_rules || '[]').trim();
  if (form.site_notice.length > 5000 || form.popup_notice.length > 5000) {
    message.error('公告内容不能超过 5000 字');
    return;
  }
  if (form.order_tips.length > 5000 || form.open_query_notice.length > 2000) {
    message.error('运营提示内容过长');
    return;
  }
  try {
    const parsed = JSON.parse(form.recharge_bonus_rules || '[]');
    if (!Array.isArray(parsed)) {
      throw new Error('not array');
    }
  } catch {
    message.error('充值赠送规则必须是 JSON 数组');
    return;
  }
  if (form.notice_url && !/^https?:\/\//i.test(form.notice_url)) {
    message.error('通知订阅地址必须以 http:// 或 https:// 开头');
    return;
  }
  if (!Number.isInteger(cacheSeconds) || cacheSeconds < 5 || cacheSeconds > 3600) {
    message.error('控制台缓存秒数必须在 5 到 3600 之间');
    return;
  }
  form.dashboard_cache_seconds = String(cacheSeconds);
  if (!form.default_connector_id) {
    form.default_connector_id = '';
  }
  saving.value = true;
  try {
    await saveSettings(form);
    message.success('设置已保存');
  } catch (error) {
    message.error(error instanceof Error ? error.message : '设置保存失败');
  } finally {
    saving.value = false;
  }
}

onMounted(load);
</script>
