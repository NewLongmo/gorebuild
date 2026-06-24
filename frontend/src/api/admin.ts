import { deleteData, getData, patchData, postData, putData, setAuthToken, type PageResult } from './http';

export interface DashboardStats {
  users: number;
  classes: number;
  orders: number;
  pending: number;
  flashOrders: number;
  flashPending: number;
  queueOrders: number;
  queueRefreshes: number;
  queueSubmit: number;
  queueSubmitFlash: number;
  queueRefresh: number;
  queueRefreshFlash: number;
  activeUsers: number;
  onlineClasses: number;
}

export interface UserRow {
  id: number;
  parentId: number;
  account: string;
  name: string;
  balance: number;
  priceRate: number;
  inviteCode: string;
  invitePriceRate: number;
  role: string;
  status: string;
  createdAt: string;
  lastIp: string;
}

export interface InviteCodeRow {
  id: number;
  code: string;
  note: string;
  maxUses: number;
  usedCount: number;
  priceRate: number;
  status: string;
  createdBy: number;
  createdAt: string;
  updatedAt: string;
  expiresAt: string | null;
}

export interface AgentTreeNode extends UserRow {
  parentAccount: string;
  depth: number;
  directChildren: number;
  children?: AgentTreeNode[];
}

export interface AgentTreeResult {
  items: AgentTreeNode[];
  total: number;
  matched: number;
  truncated: boolean;
}

export interface ClassRow {
  id: number;
  name: string;
  price: number;
  dockingCode: string;
  queryParam: string;
  queryPlatform: string;
  dockingPlatform: string;
  priceOperator: string;
  description: string;
  category: string;
  status: string;
  sort: number;
  bridgeEnabled: boolean;
}

export interface CategoryRow {
  id: number;
  sort: number;
  name: string;
  status: string;
  pinned: boolean;
  description: string;
  createdAt: string;
  updatedAt: string;
}

export interface CascadeDeleteResult {
  id: number;
  deletedClasses: number;
  deletedFavorites: number;
  deletedSpecialPrices: number;
  deletedEmptyCategories: number;
}

export interface OrderRow {
  id: number;
  userId: number;
  classId: number;
  connectorId: number;
  executionMode: string;
  pluginCode: string;
  workerId: string;
  proxyId: number;
  remoteOrderId: string;
  platform: string;
  school: string;
  studentName: string;
  account: string;
  courseId: string;
  courseName: string;
  fee: number;
  dockingCode: string;
  flashMode: boolean;
  dockingStatus: string;
  status: string;
  progress: string;
  retryCount: number;
  remarks: string;
  score: string;
  durationMinutes: number;
  createdAt: string;
}

export interface OrderEventRow {
  id: number;
  orderId: number;
  userId: number;
  level: string;
  source: string;
  eventType: string;
  content: string;
  progress: string;
  visibleToUser: boolean;
  createdAt: string;
}

export interface OrderBatchResult {
  requested?: number;
  requeued?: number;
  refunded?: number;
  deleted?: number;
  skipped: number;
  failed: number;
  items: Array<{
    id: number;
    status: string;
    error?: string;
  }>;
}

export interface ConnectorRow {
  id: number;
  name: string;
  baseUrl: string;
  appKey: string;
  kind: string;
  status: string;
  timeoutMs: number;
  sortOrder: number;
  orderSyncEnabled: boolean;
  sourceSyncEnabled: boolean;
  priceMode: 'multiplier' | 'fixed_add' | string;
  priceValue: number;
  priceRounding: 'none' | 'floor' | 'ceil' | 'round' | string;
  replaceRulesJson: string;
  categoryPriceRulesJson: string;
  createdAt: string;
}

export interface AdminMenuRow {
  id: number;
  parentId: number;
  name: string;
  route: string;
  icon: string;
  type: 'dir' | 'menu' | string;
  sortOrder: number;
  visible: boolean;
  permission: string;
  createdAt: string;
  updatedAt: string;
  children?: AdminMenuRow[];
}

export interface ConnectorReplaceRule {
  from: string;
  to: string;
}

export interface ConnectorCategoryPriceRule {
  priceMode: 'multiplier' | 'fixed_add' | string;
  priceValue: number;
  priceRounding: 'none' | 'floor' | 'ceil' | 'round' | string;
}

export interface WK29ProductPreview {
  upstreamId: string;
  kcId: string;
  name: string;
  description: string;
  status: string;
  sourcePrice: number;
  generatedPrice: number;
  existing: boolean;
  existingClassId: number;
  sync: boolean;
}

export interface WK29CategoryPreview {
  upstreamId: string;
  upstreamName: string;
  enabled: boolean;
  strategy: string;
  localCategory: string;
  productCount: number;
  products: WK29ProductPreview[];
}

export interface WK29PullResult {
  connectorId: number;
  priceRate: number;
  totalCategories: number;
  totalProducts: number;
  localCategories: string[];
  categories: WK29CategoryPreview[];
}

export interface WK29SyncResult {
  totalCategories: number;
  totalProducts: number;
  inserted: number;
  updated: number;
  skipped: number;
  categoriesInserted: number;
  categoriesUpdated: number;
}

export interface DashboardTrendPoint {
  date: string;
  orders: number;
  revenue: number;
  recharge: number;
  spend: number;
  profit: number;
}

export interface DashboardRankRow {
  id: number;
  name: string;
  count: number;
  amount: number;
}

export interface DashboardSourceStat {
  id: number;
  name: string;
  kind: string;
  status: string;
  orders: number;
  todayOrders: number;
}

export interface DashboardStatistics {
  summary: {
    totalUsers: number;
    todayNewUsers: number;
    totalOrders: number;
    todayOrders: number;
    pendingOrders: number;
    doneOrders: number;
    failedOrders: number;
    onlineClasses: number;
    activeConnectors: number;
    agentBalance: number;
    todayRecharge: number;
    totalRecharge: number;
    todaySpend: number;
    totalSpend: number;
    todayRevenue: number;
    totalRevenue: number;
    todayProfit: number;
    totalProfit: number;
  };
  trend7: DashboardTrendPoint[];
  trend30: DashboardTrendPoint[];
  userOrderRank: DashboardRankRow[];
  platformRank: DashboardRankRow[];
  rechargeRank: DashboardRankRow[];
  inviteRank: DashboardRankRow[];
  sourceStats: DashboardSourceStat[];
  generatedAt: string;
}

export interface ClassDeduplicateGroup {
  key: string;
  category: string;
  dockingPlatform: string;
  dockingCode: string;
  keepId: number;
  deleteIds: number[];
  names: string[];
  count: number;
}

export interface WK29PriceSyncResult {
  connectorId: number;
  total: number;
  updated: number;
  missing: number;
  offlined: number;
}

export interface WK29OrderSyncResult {
  connectors: number;
  fetched: number;
  matched: number;
  updated: number;
  skipped: number;
  failed: number;
  items: Array<{
    connectorId: number;
    name: string;
    fetched: number;
    matched: number;
    updated: number;
    skipped: number;
    failed: number;
    error?: string;
  }>;
}

export interface SpecialPriceRow {
  id: number;
  userId: number;
  classId: number;
  mode: number;
  price: number;
  userAccount: string;
  className: string;
  createdAt: string;
  updatedAt: string;
}

export interface LogRow {
  id: number;
  userId: number;
  type: string;
  text: string;
  amount: number;
  createdAt: string;
  sourceIp: string;
}

export interface WorkOrderRow {
  id: number;
  userId: number;
  userAccount: string;
  category: string;
  title: string;
  content: string;
  answer: string;
  status: string;
  progress: number;
  attachmentUrl: string;
  userVisible: boolean;
  createdAt: string;
  updatedAt: string;
}

export interface RechargeCardRow {
  id: number;
  code: string;
  amount: number;
  status: string;
  userId: number;
  userAccount: string;
  createdAt: string;
  usedAt: string | null;
}

export interface RecommendationRow {
  id: number;
  classId: number;
  title: string;
  note: string;
  sortOrder: number;
  visible: boolean;
  createdAt: string;
  updatedAt: string;
  className: string;
  classCategory: string;
  classPrice: number;
  classStatus: string;
  classDocking: string;
  classQuery: string;
  classDescription: string;
  classBridgeEnabled: boolean;
}

export interface SystemJobRow {
  name: string;
  status: string;
  enabled: boolean;
  lastStartedAt: string | null;
  lastFinishedAt: string | null;
  lastDurationMs: number;
  lastError: string;
  lastSummaryJson: string;
  heartbeatAt: string | null;
}

export interface PlatformPluginRow {
  code: string;
  name: string;
  description: string;
  status: string;
  sortOrder: number;
  supportsQuery: boolean;
  supportsSubmit: boolean;
  supportsRefresh: boolean;
  maxConcurrency: number;
  accountSerial: boolean;
  configJson: string;
  createdAt: string;
  updatedAt: string;
}

export interface WorkerNodeRow {
  id: number;
  workerId: string;
  hostname: string;
  status: string;
  acceptNew: boolean;
  maxConcurrency: number;
  runningCount: number;
  currentOrderId: number;
  currentPluginCode: string;
  message: string;
  startedAt: string | null;
  heartbeatAt: string | null;
  updatedAt: string;
}

export interface WorkerCommandRow {
  id: number;
  workerId: string;
  command: string;
  status: string;
  result: string;
  createdAt: string;
  executedAt: string | null;
}

export interface WorkerProxyRow {
  id: number;
  name: string;
  maskedUrl: string;
  kind: string;
  status: string;
  maxConcurrency: number;
  inUseCount: number;
  useCount: number;
  successCount: number;
  failCount: number;
  lastUsedAt: string | null;
  lastError: string;
  createdAt: string;
  updatedAt: string;
}

export interface LoginResult {
  token: string;
  expiresAt: number;
  user: {
    uid: number;
    account: string;
    name: string;
    role: string;
    status: string;
  };
}

export function login(account: string, password: string) {
  return postData<LoginResult>('/api/v1/auth/login', { account, password }).then((result) => {
    setAuthToken(result.token);
    return result;
  });
}

export function register(data: { account: string; password: string; name?: string; inviteCode: string }) {
  return postData<LoginResult>('/api/v1/auth/register', data).then((result) => {
    setAuthToken(result.token);
    return result;
  });
}

export function logout() {
  return postData<{ ok: boolean }>('/api/v1/auth/logout').finally(() => setAuthToken(''));
}

export function fetchMe() {
  return getData<{ uid: number; account: string; role: string; exp: number }>('/api/v1/auth/me');
}

export function fetchDashboard() {
  return getData<DashboardStats>('/api/v1/dashboard');
}

export function fetchDashboardStatistics() {
  return getData<DashboardStatistics>('/api/v1/dashboard/statistics');
}

export function fetchUsers(params: Record<string, unknown>) {
  return getData<PageResult<UserRow>>('/api/v1/users', params);
}

export function fetchAgentTree(params: Record<string, unknown>) {
  return getData<AgentTreeResult>('/api/v1/agents/tree', params);
}

export function fetchInviteCodes(params: Record<string, unknown>) {
  return getData<PageResult<InviteCodeRow>>('/api/v1/invite-codes', params);
}

export function createInviteCode(data: Record<string, unknown>) {
  return postData<InviteCodeRow>('/api/v1/invite-codes', data);
}

export function updateInviteCode(id: number, data: Record<string, unknown>) {
  return patchData<{ id: number }>(`/api/v1/invite-codes/${id}`, data);
}

export function removeInviteCode(id: number) {
  return deleteData<{ id: number }>(`/api/v1/invite-codes/${id}`);
}

export function fetchClasses(params: Record<string, unknown>) {
  return getData<PageResult<ClassRow>>('/api/v1/classes', params);
}

export function fetchCategories(params: Record<string, unknown>) {
  return getData<PageResult<CategoryRow>>('/api/v1/categories', params);
}

export function createCategory(data: Record<string, unknown>) {
  return postData<CategoryRow>('/api/v1/categories', data);
}

export function updateCategory(id: number, data: Record<string, unknown>) {
  return patchData<{ id: number }>(`/api/v1/categories/${id}`, data);
}

export function removeCategory(id: number) {
  return deleteData<CascadeDeleteResult>(`/api/v1/categories/${id}`);
}

export function fetchOrders(params: Record<string, unknown>) {
  return getData<PageResult<OrderRow>>('/api/v1/orders', params);
}

export function fetchSpecialPrices(params: Record<string, unknown>) {
  return getData<PageResult<SpecialPriceRow>>('/api/v1/special-prices', params);
}

export function saveSpecialPrice(data: Record<string, unknown>) {
  return postData<SpecialPriceRow>('/api/v1/special-prices', data);
}

export function updateSpecialPrice(id: number, data: Record<string, unknown>) {
  return patchData<{ id: number }>(`/api/v1/special-prices/${id}`, data);
}

export function removeSpecialPrice(id: number) {
  return deleteData<{ id: number }>(`/api/v1/special-prices/${id}`);
}

export function createUser(data: Record<string, unknown>) {
  return postData<UserRow>('/api/v1/users', data);
}

export function updateUser(id: number, data: Record<string, unknown>) {
  return patchData<{ id: number }>(`/api/v1/users/${id}`, data);
}

export function resetUserPassword(id: number) {
  return postData<{ id: number; password: string }>(`/api/v1/users/${id}/password/reset`);
}

export function adjustUserBalance(id: number, amount: number, reason: string) {
  return postData<{ id: number }>(`/api/v1/users/${id}/balance`, { amount, reason });
}

export function removeUser(id: number) {
  return deleteData<{ id: number }>(`/api/v1/users/${id}`);
}

export function createClass(data: Record<string, unknown>) {
  return postData<ClassRow>('/api/v1/classes', data);
}

export function updateClass(id: number, data: Record<string, unknown>) {
  return patchData<{ id: number }>(`/api/v1/classes/${id}`, data);
}

export function removeClass(id: number) {
  return deleteData<{ id: number }>(`/api/v1/classes/${id}`);
}

export function batchUpdateClassStatus(ids: number[], status: string) {
  return postData<{ affected: number }>('/api/v1/classes/batch/status', { ids, status });
}

export function batchMoveClasses(ids: number[], category: string) {
  return postData<{ affected: number }>('/api/v1/classes/batch/move', { ids, category });
}

export function batchDeleteClasses(ids: number[]) {
  return postData<{ affected: number }>('/api/v1/classes/batch/delete', { ids });
}

export function batchPatchClasses(updates: Array<Record<string, unknown>>) {
  return patchData<{ updated: number }>('/api/v1/classes/batch', { updates });
}

export function replaceClassKeywords(data: Record<string, unknown>) {
  return postData<{ affected: number }>('/api/v1/classes/keywords/replace', data);
}

export function addClassPrefix(data: Record<string, unknown>) {
  return postData<{ affected: number }>('/api/v1/classes/prefix', data);
}

export function previewClassDeduplicate(data: Record<string, unknown>) {
  return postData<{ groups: ClassDeduplicateGroup[]; deleteCount: number }>('/api/v1/classes/deduplicate/preview', data);
}

export function applyClassDeduplicate(data: Record<string, unknown>) {
  return postData<{ affected: number }>('/api/v1/classes/deduplicate/apply', data);
}

export function createOrder(data: Record<string, unknown>) {
  return postData<OrderRow>('/api/v1/orders', data);
}

export function updateOrder(id: number, data: Record<string, unknown>) {
  return patchData<{ id: number }>(`/api/v1/orders/${id}`, data);
}

export function removeOrder(id: number) {
  return deleteData<{ id: number }>(`/api/v1/orders/${id}`);
}

export function refreshOrder(id: number) {
  return postData<{ id: number }>(`/api/v1/orders/${id}/refresh`);
}

export function fetchOrderEvents(id: number) {
  return getData<{ items: OrderEventRow[] }>(`/api/v1/orders/${id}/events`);
}

export function refundOrder(id: number) {
  return postData<{ id: number }>(`/api/v1/orders/${id}/refund`);
}

export function batchRefreshOrders(ids: number[]) {
  return postData<OrderBatchResult>('/api/v1/orders/batch/refresh', { ids });
}

export function batchResubmitOrders(ids: number[]) {
  return postData<OrderBatchResult>('/api/v1/orders/batch/resubmit', { ids });
}

export function batchRefundOrders(ids: number[]) {
  return postData<OrderBatchResult>('/api/v1/orders/batch/refund', { ids });
}

export function batchDeleteOrders(ids: number[]) {
  return postData<OrderBatchResult>('/api/v1/orders/batch/delete', { ids });
}

export function recoverOrderQueues(batchSize = 500) {
  return postData<{ recovered: number }>(`/api/v1/orders/queues/recover?batchSize=${batchSize}`);
}

export function fetchWorkOrders(params: Record<string, unknown>) {
  return getData<PageResult<WorkOrderRow>>('/api/v1/work-orders', params);
}

export function updateWorkOrder(id: number, data: Record<string, unknown>) {
  return patchData<{ id: number }>(`/api/v1/work-orders/${id}`, data);
}

export function removeWorkOrder(id: number) {
  return deleteData<{ id: number }>(`/api/v1/work-orders/${id}`);
}

export function fetchRechargeCards(params: Record<string, unknown>) {
  return getData<PageResult<RechargeCardRow>>('/api/v1/recharge-cards', params);
}

export function createRechargeCards(data: { count?: number; amount: number; codes?: string[] }) {
  return postData<{ items: RechargeCardRow[] }>('/api/v1/recharge-cards', data);
}

export function removeRechargeCard(id: number) {
  return deleteData<{ id: number }>(`/api/v1/recharge-cards/${id}`);
}

export function fetchRecommendations(params: Record<string, unknown>) {
  return getData<PageResult<RecommendationRow>>('/api/v1/recommendations', params);
}

export function createRecommendation(data: Record<string, unknown>) {
  return postData<RecommendationRow>('/api/v1/recommendations', data);
}

export function updateRecommendation(id: number, data: Record<string, unknown>) {
  return patchData<{ id: number }>(`/api/v1/recommendations/${id}`, data);
}

export function removeRecommendation(id: number) {
  return deleteData<{ id: number }>(`/api/v1/recommendations/${id}`);
}

export function fetchConnectors(params: Record<string, unknown>) {
  return getData<PageResult<ConnectorRow>>('/api/v1/connectors', params);
}

export function createConnector(data: Record<string, unknown>) {
  return postData<ConnectorRow>('/api/v1/connectors', data);
}

export function updateConnector(id: number, data: Record<string, unknown>) {
  return patchData<{ id: number }>(`/api/v1/connectors/${id}`, data);
}

export function removeConnector(id: number) {
  return deleteData<CascadeDeleteResult>(`/api/v1/connectors/${id}`);
}

export function pullWK29Classes(data: Record<string, unknown>) {
  return postData<WK29PullResult>('/api/v1/connectors/29wk/pull', data);
}

export function syncWK29Classes(data: Record<string, unknown>) {
  return postData<WK29SyncResult>('/api/v1/connectors/29wk/sync', data);
}

export function fetchConnectorBalance(id: number) {
  return postData<{ connectorId: number; balance: number; raw: Record<string, unknown> }>(`/api/v1/connectors/${id}/balance`);
}

export function syncWK29Prices(data: Record<string, unknown>) {
  return postData<WK29PriceSyncResult>('/api/v1/connectors/29wk/prices/sync', data);
}

export function syncWK29Orders(data: Record<string, unknown>) {
  return postData<WK29OrderSyncResult>('/api/v1/connectors/29wk/orders/sync', data);
}

export function fetchMenus(params?: Record<string, unknown>) {
  return getData<AdminMenuRow[]>('/api/v1/menus', params);
}

export function createMenu(data: Record<string, unknown>) {
  return postData<AdminMenuRow>('/api/v1/menus', data);
}

export function updateMenu(id: number, data: Record<string, unknown>) {
  return patchData<{ id: number }>(`/api/v1/menus/${id}`, data);
}

export function removeMenu(id: number) {
  return deleteData<{ id: number }>(`/api/v1/menus/${id}`);
}

export function sortMenus(items: Array<{ id: number; parentId: number; sortOrder: number }>) {
  return patchData<{ ok: boolean }>('/api/v1/menus/sort', { items });
}

export function fetchSettings() {
  return getData<Record<string, string>>('/api/v1/settings');
}

export function saveSettings(data: Record<string, string>) {
  return putData<{ ok: boolean }>('/api/v1/settings', data);
}

export function fetchSystemJobs() {
  return getData<{ items: SystemJobRow[] }>('/api/v1/system/jobs');
}

export function updateSystemJob(name: string, enabled: boolean) {
  return patchData<{ name: string; enabled: boolean }>(`/api/v1/system/jobs/${encodeURIComponent(name)}`, { enabled });
}

export function runSystemJob(name: string, params?: Record<string, unknown>) {
  return postData<Record<string, unknown>>(`/api/v1/system/jobs/${encodeURIComponent(name)}/run`, params || {});
}

export function fetchPlatformPlugins(params: Record<string, unknown>) {
  return getData<PageResult<PlatformPluginRow>>('/api/v1/platform-plugins', params);
}

export function createPlatformPlugin(data: Record<string, unknown>) {
  return postData<PlatformPluginRow>('/api/v1/platform-plugins', data);
}

export function updatePlatformPlugin(code: string, data: Record<string, unknown>) {
  return patchData<{ code: string }>(`/api/v1/platform-plugins/${encodeURIComponent(code)}`, data);
}

export function removePlatformPlugin(code: string) {
  return deleteData<{ code: string }>(`/api/v1/platform-plugins/${encodeURIComponent(code)}`);
}

export function fetchWorkerNodes() {
  return getData<{ items: WorkerNodeRow[] }>('/api/v1/worker-nodes');
}

export function sendWorkerCommand(workerId: string, command: string) {
  return postData<WorkerCommandRow>(`/api/v1/worker-nodes/${encodeURIComponent(workerId)}/commands`, { command });
}

export function fetchWorkerCommands(params: Record<string, unknown>) {
  return getData<PageResult<WorkerCommandRow>>('/api/v1/worker-commands', params);
}

export function fetchWorkerProxies(params: Record<string, unknown>) {
  return getData<PageResult<WorkerProxyRow>>('/api/v1/worker-proxies', params);
}

export function createWorkerProxy(data: Record<string, unknown>) {
  return postData<WorkerProxyRow>('/api/v1/worker-proxies', data);
}

export function updateWorkerProxy(id: number, data: Record<string, unknown>) {
  return patchData<{ id: number }>(`/api/v1/worker-proxies/${id}`, data);
}

export function removeWorkerProxy(id: number) {
  return deleteData<{ id: number }>(`/api/v1/worker-proxies/${id}`);
}

export function fetchLogs(params: Record<string, unknown>) {
  return getData<PageResult<LogRow>>('/api/v1/logs', params);
}
