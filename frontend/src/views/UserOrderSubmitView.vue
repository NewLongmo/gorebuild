<template>
  <section class="page order-submit-page">
    <div class="page-heading">
      <div>
        <h1>订单提交</h1>
        <p>选择平台、查询课程并提交订单。</p>
      </div>
      <a-button :loading="bootstrapLoading" @click="reloadAll">刷新</a-button>
    </div>

    <div class="order-submit-grid">
      <a-card :bordered="false" class="work-panel">
        <div class="submit-toolbar">
          <a-space>
            <span>批量模式</span>
            <a-switch v-model:checked="batchMode" />
            <span>清理标签</span>
            <a-switch v-model:checked="cleanLabels" />
          </a-space>
          <strong>余额：{{ Number(profile.balance).toFixed(2) }}</strong>
        </div>

        <div class="category-box">
          <a-button :type="selectedCategory === '' ? 'primary' : 'default'" @click="selectCategory('')">所有</a-button>
          <a-button
            v-for="item in visibleCategories"
            :key="item.id"
            :type="selectedCategory === item.name ? 'primary' : 'default'"
            @click="selectCategory(item.name)"
          >
            {{ item.name }}
          </a-button>
          <a-button v-if="categories.length > categoryLimit" @click="categoryExpanded = !categoryExpanded">
            {{ categoryExpanded ? '收起' : '更多' }}
          </a-button>
        </div>

        <a-input-search v-model:value="platformSearch" placeholder="输入关键词搜索平台" allow-clear @search="reloadClasses" />

        <div class="platform-line">
          <a-select
            v-model:value="selectedClassId"
            show-search
            allow-clear
            :filter-option="false"
            :loading="classesLoading"
            placeholder="请选择学习平台"
            :options="classOptions"
            @search="handlePlatformSearch"
            @change="handleClassChange"
          />
          <a-tooltip :title="isFavoriteSelected ? '取消收藏' : '收藏平台'">
            <a-button :disabled="!selectedClass" @click="toggleFavorite">
              <StarFilled v-if="isFavoriteSelected" />
              <StarOutlined v-else />
            </a-button>
          </a-tooltip>
        </div>

        <a-alert v-if="selectedClass" class="class-tip" type="info" show-icon>
          <template #message>
            {{ selectedClass.name }} · {{ Number(selectedClass.userPrice).toFixed(2) }}
          </template>
          <template #description>
            {{ selectedClass.description || selectedClass.category || '平台已上架' }}
          </template>
        </a-alert>

        <a-alert v-if="orderTips" class="class-tip" type="warning" show-icon :message="orderTips" />

        <div v-if="recommendations.length" class="recommendation-strip">
          <span>推荐</span>
          <a-tag v-for="item in recommendations" :key="item.id" color="purple" @click="pickRecommendation(item)">
            {{ item.title || item.class.name }}
          </a-tag>
        </div>

        <a-textarea
          v-model:value="userinfo"
          :rows="batchMode ? 8 : 4"
          :placeholder="batchMode ? '每行一组：学校 账号 密码' : '格式：学校 账号 密码'"
          @blur="autoCorrectOnBlur"
        />

        <div v-if="inputWarnings.length" class="input-warnings">
          <a-tag v-for="warning in inputWarnings" :key="warning" color="orange">{{ warning }}</a-tag>
        </div>

        <div class="submit-actions">
          <a-button @click="correctInput">整理</a-button>
          <a-button @click="clearInput">清空</a-button>
          <a-button type="primary" :loading="querying" :disabled="!canQuery" @click="queryCourses">
            <SearchOutlined /> 查询课程
          </a-button>
          <a-button type="primary" ghost :loading="submitting" :disabled="!canSubmit" @click="submitOrders">
            <SendOutlined /> 立即提交 ({{ submitCount }})
          </a-button>
        </div>

        <a-divider />

        <div class="favorite-strip">
          <span>收藏</span>
          <a-tag v-for="item in favorites" :key="item.id" color="blue" @click="pickFavorite(item)">{{ item.name }}</a-tag>
          <span v-if="!favorites.length" class="muted">暂无收藏</span>
        </div>
      </a-card>

      <a-card :bordered="false" class="result-panel">
        <template #title>
          <div class="result-title">
            <span>查询结果</span>
            <a-space>
              <span>显示ID</span>
              <a-switch v-model:checked="showIds" size="small" />
              <a-button size="small" :disabled="!queryResults.length" @click="copyResults">复制结果</a-button>
            </a-space>
          </div>
        </template>

        <div v-if="!queryResults.length" class="empty-result">
          <a-empty description="查课后显示课程数据" />
        </div>
        <div v-else>
          <div class="result-tools">
            <a-button size="small" @click="selectAllResults">全选</a-button>
            <a-button size="small" @click="clearResultsSelection">清空</a-button>
            <span>已选 {{ selectedCourseKeys.length }} / {{ queryResults.length }}</span>
          </div>
          <a-checkbox-group v-model:value="selectedCourseKeys" class="course-result-list">
            <label v-for="(item, index) in queryResults" :key="candidateKey(item, index)" class="course-result-row">
              <a-checkbox :value="candidateKey(item, index)" />
              <span class="course-result-name">{{ item.name || '未命名课程' }}</span>
              <code v-if="showIds">{{ item.id || '-' }}</code>
            </label>
          </a-checkbox-group>
        </div>

        <a-divider />
        <div class="rank-grid">
          <div>
            <h3>热销平台</h3>
            <ol>
              <li v-for="item in platformRank" :key="item.name">
                <span>{{ item.name }}</span>
                <strong>{{ item.count }}</strong>
              </li>
            </ol>
          </div>
          <div>
            <h3>热门分类</h3>
            <ol>
              <li v-for="item in categoryRank" :key="item.name">
                <span>{{ item.name }}</span>
                <strong>{{ item.count }}</strong>
              </li>
            </ol>
          </div>
        </div>
      </a-card>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue';
import { message } from 'ant-design-vue';
import { SearchOutlined, SendOutlined, StarFilled, StarOutlined } from '@ant-design/icons-vue';
import {
  addAgentClassFavorite,
  createAgentOrdersBatch,
  fetchAgentClasses,
  fetchOrderSubmitBootstrap,
  fetchOrderSubmitRecommendations,
  queryAgentCourses,
  removeAgentClassFavorite,
  type AgentClassRow,
  type AgentOrderPayload,
  type AgentRecommendationRow,
  type CourseQueryCandidate,
  type OrderLeaderboardRow,
} from '@/api/user';
import type { CategoryRow } from '@/api/admin';

const emit = defineEmits<{ changed: [] }>();

interface ParsedInput {
  school: string;
  account: string;
  password: string;
}

const categoryLimit = 12;
const bootstrapLoading = ref(false);
const classesLoading = ref(false);
const querying = ref(false);
const submitting = ref(false);
const batchMode = ref(false);
const cleanLabels = ref(true);
const showIds = ref(false);
const categoryExpanded = ref(false);
const selectedCategory = ref('');
const platformSearch = ref('');
const selectedClassId = ref<number>();
const userinfo = ref('');
const categories = ref<CategoryRow[]>([]);
const classes = ref<AgentClassRow[]>([]);
const favorites = ref<AgentClassRow[]>([]);
const favoriteIds = ref<Set<number>>(new Set());
const recommendations = ref<AgentRecommendationRow[]>([]);
const platformRank = ref<OrderLeaderboardRow[]>([]);
const categoryRank = ref<OrderLeaderboardRow[]>([]);
const orderTips = ref('');
const queryResults = ref<CourseQueryCandidate[]>([]);
const selectedCourseKeys = ref<string[]>([]);
const profile = reactive({ balance: 0, priceRate: 1, account: '' });

const visibleCategories = computed(() => (categoryExpanded.value ? categories.value : categories.value.slice(0, categoryLimit)));
const selectedClass = computed(() => classes.value.find((item) => item.id === selectedClassId.value) || favorites.value.find((item) => item.id === selectedClassId.value));
const classOptions = computed(() => classes.value.map((item) => ({ label: `${item.name} · ${Number(item.userPrice).toFixed(2)}`, value: item.id })));
const isFavoriteSelected = computed(() => !!selectedClass.value && favoriteIds.value.has(selectedClass.value.id));
const parsedInputs = computed(() => parseInputBlock(userinfo.value, batchMode.value, cleanLabels.value));
const inputWarnings = computed(() => inputWarningsFor(userinfo.value));
const canQuery = computed(() => !!selectedClass.value && parsedInputs.value.length > 0 && !!parsedInputs.value[0].account);
const canSubmit = computed(() => !!selectedClass.value && parsedInputs.value.length > 0);
const selectedCandidates = computed(() => queryResults.value.filter((item, index) => selectedCourseKeys.value.includes(candidateKey(item, index))));
const submitCount = computed(() => Math.max(1, selectedCandidates.value.length || 1) * Math.max(1, parsedInputs.value.length || 0));

async function reloadAll() {
  await loadBootstrap();
  await loadRecommendations();
  await reloadClasses();
}

async function loadBootstrap() {
  bootstrapLoading.value = true;
  try {
    const data = await fetchOrderSubmitBootstrap();
    categories.value = data.categories;
    favorites.value = data.favorites;
    favoriteIds.value = new Set(data.favoriteIds);
    profile.balance = data.profile.balance;
    profile.priceRate = data.profile.priceRate;
    profile.account = data.profile.account;
  } catch (error) {
    message.error(error instanceof Error ? error.message : '订单提交初始化失败');
  } finally {
    bootstrapLoading.value = false;
  }
}

async function loadRecommendations() {
  try {
    const data = await fetchOrderSubmitRecommendations();
    recommendations.value = data.items;
    platformRank.value = data.platformRank;
    categoryRank.value = data.categoryRank;
    orderTips.value = data.orderTips || '';
  } catch {
    recommendations.value = [];
    platformRank.value = [];
    categoryRank.value = [];
    orderTips.value = '';
  }
}

async function reloadClasses() {
  classesLoading.value = true;
  try {
    const data = await fetchAgentClasses({
      q: platformSearch.value,
      category: selectedCategory.value || undefined,
      page: 1,
      perPage: 100,
    });
    classes.value = data.items;
    if (!selectedClassId.value && data.items.length) {
      selectedClassId.value = data.items[0].id;
    }
  } catch (error) {
    message.error(error instanceof Error ? error.message : '平台加载失败');
  } finally {
    classesLoading.value = false;
  }
}

function selectCategory(name: string) {
  selectedCategory.value = name;
  selectedClassId.value = undefined;
  queryResults.value = [];
  selectedCourseKeys.value = [];
  void reloadClasses();
}

function handlePlatformSearch(value: string) {
  platformSearch.value = value;
  void reloadClasses();
}

function handleClassChange() {
  queryResults.value = [];
  selectedCourseKeys.value = [];
}

function autoCorrectOnBlur() {
  if (userinfo.value.trim()) {
    correctInput();
  }
}

function correctInput() {
  const corrected = parseInputBlock(userinfo.value, batchMode.value, true)
    .map((item) => [item.school, item.account, item.password].filter(Boolean).join(' '))
    .join('\n');
  if (corrected && corrected !== userinfo.value.trim()) {
    userinfo.value = corrected;
    message.success('已整理账号信息');
  }
}

function clearInput() {
  userinfo.value = '';
  queryResults.value = [];
  selectedCourseKeys.value = [];
}

async function toggleFavorite() {
  if (!selectedClass.value) return;
  const id = selectedClass.value.id;
  try {
    if (favoriteIds.value.has(id)) {
      await removeAgentClassFavorite(id);
      favoriteIds.value.delete(id);
      favorites.value = favorites.value.filter((item) => item.id !== id);
    } else {
      await addAgentClassFavorite(id);
      favoriteIds.value.add(id);
      if (!favorites.value.some((item) => item.id === id)) {
        favorites.value.unshift(selectedClass.value);
      }
    }
  } catch (error) {
    message.error(error instanceof Error ? error.message : '收藏更新失败');
  }
}

function pickFavorite(row: AgentClassRow) {
  if (!classes.value.some((item) => item.id === row.id)) {
    classes.value.unshift(row);
  }
  selectedClassId.value = row.id;
  queryResults.value = [];
  selectedCourseKeys.value = [];
}

function pickRecommendation(row: AgentRecommendationRow) {
  if (!classes.value.some((item) => item.id === row.class.id)) {
    classes.value.unshift(row.class);
  }
  selectedClassId.value = row.class.id;
  queryResults.value = [];
  selectedCourseKeys.value = [];
}

async function queryCourses() {
  if (!selectedClass.value || !parsedInputs.value.length) return;
  const input = parsedInputs.value[0];
  querying.value = true;
  try {
    const result = await queryAgentCourses({
      classId: selectedClass.value.id,
      school: input.school,
      account: input.account,
      accountPassword: input.password,
    });
    queryResults.value = result.candidates;
    selectedCourseKeys.value = result.candidates.map((item, index) => candidateKey(item, index));
    if (!result.candidates.length) {
      message.warning('上游未返回课程');
    }
  } catch (error) {
    message.error(error instanceof Error ? error.message : '查课失败');
  } finally {
    querying.value = false;
  }
}

async function submitOrders() {
  if (!selectedClass.value) return;
  const entries = buildOrderEntries();
  if (!entries.length) {
    message.error('账号信息不能为空');
    return;
  }
  submitting.value = true;
  try {
    const result = await createAgentOrdersBatch(selectedClass.value.id, entries);
    profile.balance = Math.max(0, profile.balance - entries.length * Number(selectedClass.value.userPrice || 0));
    emit('changed');
    if (result.failed) {
      message.warning(`成功 ${result.succeeded} 条，失败 ${result.failed} 条`);
    } else {
      message.success(`已提交 ${result.succeeded} 条订单`);
    }
  } catch (error) {
    message.error(error instanceof Error ? error.message : '订单提交失败');
  } finally {
    submitting.value = false;
  }
}

function buildOrderEntries(): AgentOrderPayload[] {
  if (!selectedClass.value) return [];
  const courses = selectedCandidates.value.length ? selectedCandidates.value : [{ id: '', name: selectedClass.value.name, raw: {} }];
  const entries: AgentOrderPayload[] = [];
  for (const input of parsedInputs.value) {
    if (!input.account) continue;
    for (const course of courses) {
      entries.push({
        classId: selectedClass.value.id,
        school: input.school,
        account: input.account,
        accountPassword: input.password,
        courseId: String(course.id || ''),
        courseName: String(course.name || selectedClass.value.name),
      });
    }
  }
  return entries;
}

function selectAllResults() {
  selectedCourseKeys.value = queryResults.value.map((item, index) => candidateKey(item, index));
}

function clearResultsSelection() {
  selectedCourseKeys.value = [];
}

async function copyResults() {
  const text = queryResults.value.map((item) => `${item.id || '-'} ${item.name || ''}`.trim()).join('\n');
  try {
    await navigator.clipboard.writeText(text);
    message.success('已复制');
  } catch {
    message.warning('复制失败');
  }
}

function candidateKey(item: CourseQueryCandidate, index: number) {
  return `${item.id || 'course'}:${item.name || ''}:${index}`;
}

function parseInputBlock(text: string, batch: boolean, clean: boolean): ParsedInput[] {
  const lines = (batch ? text.split(/\r?\n/) : [text]).map((line) => line.trim()).filter(Boolean);
  return lines.map((line) => parseInputLine(clean ? cleanInputLabels(line) : line)).filter((item) => item.account || item.password || item.school);
}

function parseInputLine(line: string): ParsedInput {
  const normalized = normalizeSymbols(line).replace(/\s+/g, ' ').trim();
  const keyed = extractKeyedInput(normalized);
  if (keyed.account || keyed.password) {
    return keyed;
  }
  const parts = normalized.split(' ').filter(Boolean);
  if (parts.length >= 3) {
    return { school: parts[0], account: parts[1], password: parts.slice(2).join(' ') };
  }
  if (parts.length === 2) {
    return { school: '', account: parts[0], password: parts[1] };
  }
  return { school: '', account: parts[0] || '', password: '' };
}

function extractKeyedInput(line: string): ParsedInput {
  return {
    school: extractValue(line, ['学校', '院校', 'school']),
    account: extractValue(line, ['账号', '帐号', '用户名', 'account', 'username', 'user']),
    password: extractValue(line, ['密码', 'password', 'pwd', 'pass']),
  };
}

function extractValue(line: string, keys: string[]) {
  for (const key of keys) {
    const escaped = key.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
    const match = line.match(new RegExp(`${escaped}\\s*[:：=]?\\s*([^\\s,，;；]+)`, 'i'));
    if (match?.[1]) return match[1].trim();
  }
  return '';
}

function cleanInputLabels(line: string) {
  return line
    .replace(/(账号|帐号|用户名|account|username|user)\s*[:：=]/gi, ' ')
    .replace(/(密码|password|pwd|pass)\s*[:：=]/gi, ' ')
    .replace(/(学校|院校|school)\s*[:：=]/gi, ' ');
}

function normalizeSymbols(value: string) {
  const map: Record<string, string> = { '，': ',', '。': '.', '：': ':', '；': ';', '！': '!', '？': '?', '（': '(', '）': ')', '　': ' ' };
  return value.replace(/[，。：；！？（）　]/g, (char) => map[char] || char);
}

function inputWarningsFor(value: string) {
  const warnings: string[] = [];
  if (/[\u3000-\u303f\uff00-\uffef]/.test(value)) warnings.push('包含中文符号');
  if (/[\u4e00-\u9fa5]/.test(value) && !/(学校|院校|账号|帐号|密码|用户名)/.test(value)) warnings.push('包含中文字符');
  return warnings;
}

onMounted(() => {
  void reloadAll();
});
</script>

<style scoped>
.order-submit-grid {
  display: grid;
  grid-template-columns: minmax(360px, 0.95fr) minmax(360px, 1fr);
  gap: 16px;
  align-items: start;
}

.work-panel,
.result-panel {
  min-height: 440px;
  border-radius: 8px;
  box-shadow: 0 1px 2px rgb(15 23 42 / 6%);
}

.submit-toolbar,
.result-title,
.result-tools {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.category-box {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin: 14px 0;
}

.platform-line {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 40px;
  gap: 8px;
  margin-top: 12px;
}

.class-tip,
.submit-actions,
.favorite-strip,
.recommendation-strip,
.input-warnings {
  margin-top: 12px;
}

.submit-actions,
.favorite-strip,
.recommendation-strip {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  align-items: center;
}

.favorite-strip .ant-tag,
.recommendation-strip .ant-tag {
  cursor: pointer;
}

.muted {
  color: #667085;
}

.empty-result {
  display: grid;
  min-height: 300px;
  place-items: center;
}

.course-result-list {
  display: grid;
  gap: 8px;
  margin-top: 12px;
}

.course-result-row {
  display: grid;
  grid-template-columns: 24px minmax(0, 1fr) auto;
  gap: 8px;
  align-items: center;
  padding: 10px 12px;
  border: 1px solid #edf0f3;
  border-radius: 8px;
  cursor: pointer;
}

.course-result-row:hover {
  border-color: #91caff;
  background: #f5f9ff;
}

.course-result-name {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

code {
  color: #667085;
  font-size: 12px;
}

.rank-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 16px;
}

.rank-grid h3 {
  margin: 0 0 8px;
  font-size: 14px;
}

.rank-grid ol {
  display: grid;
  gap: 8px;
  margin: 0;
  padding: 0;
  list-style: none;
}

.rank-grid li {
  display: flex;
  justify-content: space-between;
  gap: 12px;
  padding: 8px 0;
  border-bottom: 1px solid #edf0f3;
}

.rank-grid span {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

@media (max-width: 980px) {
  .order-submit-grid {
    grid-template-columns: 1fr;
  }

  .rank-grid {
    grid-template-columns: 1fr;
  }
}
</style>
