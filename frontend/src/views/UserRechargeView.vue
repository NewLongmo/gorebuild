<template>
  <section class="page">
    <div class="page-heading">
      <div>
        <h1>充值卡</h1>
      </div>
    </div>

    <div class="form-panel">
      <a-form layout="vertical">
        <a-form-item label="卡密">
          <a-input v-model:value="code" allow-clear />
        </a-form-item>
        <a-space>
          <a-button :loading="querying" @click="queryCard">查询</a-button>
          <a-button type="primary" :loading="redeeming" @click="redeemCard">立即充值</a-button>
        </a-space>
      </a-form>
    </div>

    <a-descriptions v-if="card" class="detail-block" :column="1" bordered size="small">
      <a-descriptions-item label="卡密">{{ card.code }}</a-descriptions-item>
      <a-descriptions-item label="金额">{{ Number(card.amount).toFixed(2) }}</a-descriptions-item>
      <a-descriptions-item label="状态">
        <a-tag :color="card.status === 'used' ? 'green' : 'blue'">{{ card.status === 'used' ? '已使用' : '未使用' }}</a-tag>
      </a-descriptions-item>
      <a-descriptions-item label="使用时间">{{ card.usedAt || '-' }}</a-descriptions-item>
    </a-descriptions>
  </section>
</template>

<script setup lang="ts">
import { ref } from 'vue';
import { message } from 'ant-design-vue';
import { queryRechargeCard, redeemRechargeCard } from '@/api/user';
import type { RechargeCardRow } from '@/api/admin';

const code = ref('');
const querying = ref(false);
const redeeming = ref(false);
const card = ref<RechargeCardRow>();

async function queryCard() {
  if (!code.value.trim()) {
    message.error('请输入卡密');
    return;
  }
  querying.value = true;
  try {
    card.value = await queryRechargeCard(code.value.trim());
    message.success('查询成功');
  } catch (error) {
    card.value = undefined;
    message.error(error instanceof Error ? error.message : '查询失败');
  } finally {
    querying.value = false;
  }
}

async function redeemCard() {
  if (!code.value.trim()) {
    message.error('请输入卡密');
    return;
  }
  redeeming.value = true;
  try {
    card.value = await redeemRechargeCard(code.value.trim());
    message.success(`充值成功，到账 ${Number(card.value.amount).toFixed(2)}`);
  } catch (error) {
    message.error(error instanceof Error ? error.message : '充值失败');
  } finally {
    redeeming.value = false;
  }
}
</script>
