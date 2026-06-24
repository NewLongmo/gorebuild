import { getData, patchData, postData } from './http';

export interface PublicOrderRow {
  id: number;
  platform: string;
  school: string;
  studentName: string;
  account: string;
  courseName: string;
  status: string;
  dockingStatus: string;
  progress: string;
  remarks: string;
  score: string;
  durationMinutes: number;
  createdAt: string;
}

export interface PublicOrderSearchResult {
  account: string;
  items: PublicOrderRow[];
  total: number;
  notice: string;
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

export function searchPublicOrders(account: string) {
  return postData<PublicOrderSearchResult>('/api/v1/public/orders/search', { account });
}

export function refreshPublicOrder(id: number, account: string) {
  return postData<{ id: number }>(`/api/v1/public/orders/${id}/refresh`, { account });
}

export function fetchPublicOrderEvents(id: number, account: string) {
  return getData<{ items: OrderEventRow[] }>(`/api/v1/public/orders/${id}/events`, { account });
}

export function updatePublicOrderPassword(id: number, account: string, password: string) {
  return patchData<{ id: number }>(`/api/v1/public/orders/${id}/password`, { account, password });
}

export function resubmitPublicOrder(id: number, account: string) {
  return postData<{ id: number }>(`/api/v1/public/orders/${id}/resubmit`, { account });
}
