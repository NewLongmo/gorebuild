import { createApp } from 'vue';
import { createPinia } from 'pinia';
import {
  Alert,
  Badge,
  Button,
  Card,
  Col,
  ConfigProvider,
  Form,
  Input,
  InputNumber,
  Layout,
  Menu,
  Modal,
  Popconfirm,
  Row,
  Segmented,
  Select,
  Statistic,
  Switch,
  Table,
  Tag,
} from 'ant-design-vue';
import 'ant-design-vue/dist/reset.css';

import App from './App.vue';
import { setUnauthorizedHandler } from './api/http';
import { router } from './router';
import { useAppStore } from './stores/app';
import './styles/app.css';

const app = createApp(App);
const pinia = createPinia();

[
  Alert,
  Badge,
  Button,
  Card,
  Col,
  ConfigProvider,
  Form,
  Input,
  InputNumber,
  Layout,
  Menu,
  Modal,
  Popconfirm,
  Row,
  Segmented,
  Select,
  Statistic,
  Switch,
  Table,
  Tag,
].forEach((component) => app.use(component));

app.use(pinia).use(router);

setUnauthorizedHandler(() => {
  const store = useAppStore(pinia);
  store.setAccount('');
  store.setRole('');
  if (router.currentRoute.value.path !== '/login') {
    router.replace('/login');
  }
});

app.mount('#app');
