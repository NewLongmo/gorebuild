<template>
  <main class="login-page">
    <section class="login-panel">
      <div class="login-brand">
        <span class="brand__mark">G</span>
        <div>
          <h1>DW0RDWK 管理台</h1>
          <p>登录后继续管理订单与代理。</p>
        </div>
      </div>

      <a-form layout="vertical" :model="form" @finish="submit">
        <a-form-item label="账号" name="account" :rules="[{ required: true, message: '请输入账号' }]">
          <a-input v-model:value="form.account" autocomplete="username" />
        </a-form-item>
        <a-form-item label="密码" name="password" :rules="[{ required: true, message: '请输入密码' }]">
          <a-input-password v-model:value="form.password" autocomplete="current-password" />
        </a-form-item>
        <a-button type="primary" html-type="submit" block :loading="loading">登录</a-button>
      </a-form>

      <div class="auth-switch">
        拿到了邀请码？
        <RouterLink to="/register">注册代理账号</RouterLink>
      </div>
      <div class="auth-switch">
        已经下单？
        <RouterLink to="/support">自助查单补单</RouterLink>
      </div>
    </section>
  </main>
</template>

<script setup lang="ts">
import { reactive, ref } from 'vue';
import { useRouter } from 'vue-router';
import { message } from 'ant-design-vue';
import { login } from '@/api/admin';
import { useAppStore } from '@/stores/app';

const router = useRouter();
const app = useAppStore();
const loading = ref(false);
const form = reactive({
  account: 'admin',
  password: '',
});

async function submit() {
  loading.value = true;
  try {
    const result = await login(form.account, form.password);
    app.setAccount(result.user.account);
    app.setRole(result.user.role);
    router.replace(result.user.role === 'admin' ? '/dashboard' : '/user/dashboard');
  } catch (error) {
    message.error(error instanceof Error ? error.message : '登录失败');
  } finally {
    loading.value = false;
  }
}
</script>
