import { deleteData, getData, postData, putData, type PageResult } from './http';
import { patchData } from './http';
import type { CategoryRow, ClassRow, LogRow, OrderEventRow, OrderRow, RechargeCardRow, UserRow, WorkOrderRow } from './admin';

export interface AgentProfile {
  id: number;
  account: string;
  name: string;
  balance: number;
  priceRate: number;
  role: string;
  apiKey: string;
  inviteCode: string;
  invitePriceRate: number;
  notice: string;
}

export interface AgentDashboard {
  balance: number;
  priceRate: number;
  orders: number;
  pending: number;
  done: number;
  failed: number;
  todayOrders: number;
  unfinished: number;
  refreshing: number;
  subAgents: number;
  totalSpend: number;
  siteNotice: string;
  popupNotice: string;
  noticeUrl: string;
  parentNotice: string;
  ownNotice: string;
  trend7: Array<{ date: string; orders: number }>;
}

export interface AgentClassRow extends ClassRow {
  userPrice: number;
}

export interface AgentOrderPayload {
  classId: number;
  school?: string;
  studentName?: string;
  account: string;
  accountPassword?: string;
  courseId?: string;
  courseName: string;
  flashMode?: boolean;
  durationMinutes?: number;
}

export interface OrderSubmitBootstrap {
  profile: AgentProfile;
  categories: CategoryRow[];
  favoriteIds: number[];
  favorites: AgentClassRow[];
}

export interface OrderLeaderboardRow {
  name: string;
  count: number;
  amount: number;
  lastOrderAt: string;
}

export interface AgentRecommendationRow {
  id: number;
  title: string;
  note: string;
  sortOrder: number;
  class: AgentClassRow;
}

export interface OrderSubmitRecommendations {
  items: AgentRecommendationRow[];
  platformRank: OrderLeaderboardRow[];
  categoryRank: OrderLeaderboardRow[];
  orderTips: string;
}

export interface CourseQueryCandidate {
  id: string;
  name: string;
  raw: Record<string, unknown>;
}

export interface CourseQueryResult {
  classId: number;
  raw: Record<string, unknown>;
  candidates: CourseQueryCandidate[];
  userinfo: string;
}

export interface AgentBatchOrderResult {
  requested: number;
  succeeded: number;
  failed: number;
  items: Array<{
    index: number;
    orderId: number;
    status: string;
    error?: string;
  }>;
}

export interface ChildAgentPayload {
  account?: string;
  password?: string;
  name?: string;
  balance?: number;
  priceRate?: number;
  status?: string;
}

export function fetchAgentProfile() {
  return getData<AgentProfile>('/api/v1/me/profile');
}

export function fetchAgentDashboard() {
  return getData<AgentDashboard>('/api/v1/me/dashboard');
}

export function fetchAgentClasses(params: Record<string, unknown>) {
  return getData<PageResult<AgentClassRow>>('/api/v1/me/classes', params);
}

export function fetchAgentCategories() {
  return getData<{ items: CategoryRow[] }>('/api/v1/me/categories');
}

export function fetchOrderSubmitBootstrap() {
  return getData<OrderSubmitBootstrap>('/api/v1/me/order-submit/bootstrap');
}

export function fetchOrderSubmitRecommendations() {
  return getData<OrderSubmitRecommendations>('/api/v1/me/order-submit/recommendations');
}

export function queryAgentCourses(data: Record<string, unknown>) {
  return postData<CourseQueryResult>('/api/v1/me/course-query', data);
}

export function fetchAgentClassFavorites() {
  return getData<{ classIds: number[] }>('/api/v1/me/class-favorites');
}

export function addAgentClassFavorite(classId: number) {
  return putData<{ classId: number }>(`/api/v1/me/class-favorites/${classId}`, {});
}

export function removeAgentClassFavorite(classId: number) {
  return deleteData<{ classId: number }>(`/api/v1/me/class-favorites/${classId}`);
}

export function fetchChildAgents(params: Record<string, unknown>) {
  return getData<PageResult<UserRow>>('/api/v1/me/agents', params);
}

export function fetchAgentLogs(params: Record<string, unknown>) {
  return getData<PageResult<LogRow>>('/api/v1/me/logs', params);
}

export function changeAgentPassword(currentPassword: string, newPassword: string) {
  return postData<{ ok: boolean }>('/api/v1/me/password', { currentPassword, newPassword });
}

export function updateAgentNotice(notice: string) {
  return putData<{ notice: string }>('/api/v1/me/notice', { notice });
}

export function updateAgentInvite(code: string, priceRate: number) {
  return putData<{ inviteCode: string; invitePriceRate: number }>('/api/v1/me/invite', { code, priceRate });
}

export function regenerateAgentApiKey() {
  return postData<{ apiKey: string }>('/api/v1/me/api-key');
}

export function disableAgentApiKey() {
  return deleteData<{ ok: boolean }>('/api/v1/me/api-key');
}

export function createChildAgent(data: ChildAgentPayload) {
  return postData<UserRow>('/api/v1/me/agents', data as Record<string, unknown>);
}

export function updateChildAgent(id: number, data: ChildAgentPayload) {
  return patchData<{ id: number }>(`/api/v1/me/agents/${id}`, data as Record<string, unknown>);
}

export function resetChildAgentPassword(id: number) {
  return postData<{ id: number; password: string }>(`/api/v1/me/agents/${id}/password/reset`);
}

export function transferChildBalance(id: number, amount: number) {
  return postData<{ parentId: number; childId: number; chargedAmount: number; creditedAmount: number }>(`/api/v1/me/agents/${id}/balance`, { amount });
}

export function fetchAgentOrders(params: Record<string, unknown>) {
  return getData<PageResult<OrderRow>>('/api/v1/me/orders', params);
}

export function fetchAgentOrderEvents(id: number) {
  return getData<{ items: OrderEventRow[] }>(`/api/v1/me/orders/${id}/events`);
}

export function createAgentOrder(data: AgentOrderPayload) {
  return postData<OrderRow>('/api/v1/me/orders', data as unknown as Record<string, unknown>);
}

export function createAgentOrdersBatch(classId: number, entries: AgentOrderPayload[]) {
  return postData<AgentBatchOrderResult>('/api/v1/me/orders/batch', { classId, entries });
}

export function refreshAgentOrder(id: number) {
  return postData<{ id: number }>(`/api/v1/me/orders/${id}/refresh`);
}

export function cancelAgentOrder(id: number) {
  return postData<{ id: number }>(`/api/v1/me/orders/${id}/cancel`);
}

export function updateAgentOrderPassword(id: number, password: string) {
  return patchData<{ id: number }>(`/api/v1/me/orders/${id}/password`, { password });
}

export function fetchAgentWorkOrders(params: Record<string, unknown>) {
  return getData<PageResult<WorkOrderRow>>('/api/v1/me/work-orders', params);
}

export function createAgentWorkOrder(data: { category: string; title: string; content: string; attachmentUrl?: string }) {
  return postData<WorkOrderRow>('/api/v1/me/work-orders', data);
}

export function replyAgentWorkOrder(id: number, content: string) {
  return patchData<{ id: number }>(`/api/v1/me/work-orders/${id}/reply`, { content });
}

export function removeAgentWorkOrder(id: number) {
  return deleteData<{ id: number }>(`/api/v1/me/work-orders/${id}`);
}

export function queryRechargeCard(code: string) {
  return postData<RechargeCardRow>('/api/v1/me/recharge-cards/query', { code });
}

export function redeemRechargeCard(code: string) {
  return postData<RechargeCardRow>('/api/v1/me/recharge-cards/redeem', { code });
}
