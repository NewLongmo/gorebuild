export const roleLabels: Record<string, string> = {
  admin: '管理员',
  agent: '代理',
};

export const userStatusLabels: Record<string, string> = {
  active: '启用',
  disabled: '停用',
};

export const classStatusLabels: Record<string, string> = {
  online: '上架',
  offline: '下架',
};

export const orderStatusLabels: Record<string, string> = {
  pending: '待处理',
  queued: '排队中',
  processing: '处理中',
  done: '已完成',
  failed: '失败',
  cancelled: '已取消',
  refunded: '已退款',
};

export const dockingStatusLabels: Record<string, string> = {
  pending: '待推送',
  sent: '已推送',
  refresh_requested: '待刷新',
  failed: '对接失败',
  queue_failed: '入队失败',
  cancelled: '已取消',
  refunded: '已退款',
};

export function labelOf(labels: Record<string, string>, value?: string) {
  if (!value) {
    return '-';
  }
  return labels[value] || value;
}
