<template>
  <main class="login-page auth-page">
    <section class="login-panel auth-panel">
      <div class="login-brand">
        <span class="brand__mark">G</span>
        <div>
          <h1>邀请码注册</h1>
          <p>输入管理员发放的邀请码，创建代理账号。</p>
        </div>
      </div>

      <a-form layout="vertical" :model="form" @finish="submit">
        <a-form-item label="邀请码" name="inviteCode" :rules="[{ required: true, message: '请输入邀请码' }]">
          <a-input v-model:value="form.inviteCode" autocomplete="one-time-code" />
        </a-form-item>
        <a-form-item label="账号" name="account" :rules="[{ required: true, message: '请输入账号' }]">
          <a-input v-model:value="form.account" autocomplete="username" />
        </a-form-item>
        <a-form-item label="名称">
          <a-input v-model:value="form.name" autocomplete="name" />
        </a-form-item>
        <a-form-item label="密码" name="password" :rules="[{ required: true, message: '请输入密码' }]">
          <a-input-password v-model:value="form.password" autocomplete="new-password" />
        </a-form-item>
        <a-form-item label="确认密码" name="confirmPassword" :rules="[{ required: true, message: '请再次输入密码' }]">
          <a-input-password v-model:value="form.confirmPassword" autocomplete="new-password" />
        </a-form-item>
        <a-button type="primary" html-type="submit" block :loading="loading">注册并进入</a-button>
      </a-form>

      <div class="auth-switch">
        已有账号？
        <RouterLink to="/login">返回登录</RouterLink>
      </div>
    </section>
  </main>
</template>

<script setup lang="ts">
import { reactive, ref } from 'vue';
import { useRouter } from 'vue-router';
import { message } from 'ant-design-vue';
import { register } from '@/api/admin';
import { useAppStore } from '@/stores/app';

const router = useRouter();
const app = useAppStore();
const loading = ref(false);
const form = reactive({
  inviteCode: '',
  account: '',
  name: '',
  password: '',
  confirmPassword: '',
});

async function submit() {
  if (form.password.length < 8) {
    message.error('密码至少需要 8 位');
    return;
  }
  if (form.password !== form.confirmPassword) {
    message.error('两次输入的密码不一致');
    return;
  }
  loading.value = true;
  try {
    const result = await register({
      inviteCode: form.inviteCode.trim(),
      account: form.account.trim(),
      name: form.name.trim(),
      password: form.password,
    });
    app.setAccount(result.user.account);
    app.setRole(result.user.role);
    message.success('注册成功');
    router.replace(result.user.role === 'admin' ? '/dashboard' : '/user/dashboard');
  } catch (error) {
    message.error(error instanceof Error ? error.message : '注册失败');
  } finally {
    loading.value = false;
  }
}
</script>
