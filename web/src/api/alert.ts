import api from './index'

// 通知渠道 API
export const channelAPI = {
  list: (params?: { type?: string; enabled?: boolean; page?: number; page_size?: number }) =>
    api.get('/alert/channels', { params }),
  get: (id: number) => api.get(`/alert/channels/${id}`),
  create: (data: any) => api.post('/alert/channels', data),
  update: (id: number, data: any) => api.put(`/alert/channels/${id}`, data),
  delete: (id: number) => api.delete(`/alert/channels/${id}`),
  test: (id: number) => api.post(`/alert/channels/${id}/test`)
}

// 告警规则 API
export const alertRuleAPI = {
  list: (params?: { level?: string; enabled?: boolean; page?: number; page_size?: number }) =>
    api.get('/alert/rules', { params }),
  get: (id: number) => api.get(`/alert/rules/${id}`),
  create: (data: any) => api.post('/alert/rules', data),
  update: (id: number, data: any) => api.put(`/alert/rules/${id}`, data),
  delete: (id: number) => api.delete(`/alert/rules/${id}`),
  copy: (id: number) => api.post(`/alert/rules/${id}/copy`)
}

// 告警记录 API
export const alertAPI = {
  list: (params?: {
    level?: string;
    status?: string;
    rule_id?: number;
    start_time?: string;
    end_time?: string;
    page?: number;
    page_size?: number;
  }) => api.get('/alerts', { params }),
  get: (id: number) => api.get(`/alerts/${id}`),
  acknowledge: (id: number, note?: string) => api.post(`/alerts/${id}/acknowledge`, { note }),
  resolve: (id: number, note?: string) => api.post(`/alerts/${id}/resolve`, { note }),
  addNote: (id: number, content: string) => api.post(`/alerts/${id}/notes`, { content })
}

// Dashboard 告警统计 API
export const alertDashboardAPI = {
  overview: () => api.get('/dashboard/alerts/overview'),
  stats: (params?: any) => api.get('/dashboard/alerts/stats', { params })
}

// 告警类型枚举
export const AlertLevelOptions = [
  { label: 'P0 - 紧急', value: 'P0', color: '#f56c6c' },
  { label: 'P1 - 重要', value: 'P1', color: '#e6a23c' },
  { label: 'P2 - 一般', value: 'P2', color: '#409eff' },
  { label: 'P3 - 提示', value: 'P3', color: '#909399' }
]

export const AlertStatusOptions = [
  { label: '活跃', value: 'active', color: '#f56c6c' },
  { label: '已确认', value: 'acknowledged', color: '#e6a23c' },
  { label: '已解决', value: 'resolved', color: '#67c23a' },
  { label: '已升级', value: 'escalated', color: '#f56c6c' }
]

export const ChannelTypeOptions = [
  { label: '飞书', value: 'feishu' },
  { label: '企业微信', value: 'wecom' },
  { label: '钉钉', value: 'dingtalk' },
  { label: '邮件', value: 'email' }
]

// 条件操作符
export const ConditionOperatorOptions = [
  { label: '等于', value: 'eq' },
  { label: '不等于', value: 'ne' },
  { label: '大于', value: 'gt' },
  { label: '大于等于', value: 'gte' },
  { label: '小于', value: 'lt' },
  { label: '小于等于', value: 'lte' },
  { label: '包含', value: 'contains' },
  { label: '正则匹配', value: 'regex' },
  { label: '在列表中', value: 'in' },
  { label: '不在列表中', value: 'not_in' }
]

// 事件类型
export const EventTypeOptions = [
  { label: '备份失败', value: 'backup_failed' },
  { label: '备份超时', value: 'backup_timeout' },
  { label: '备份过慢', value: 'backup_slow' },
  { label: '恢复失败', value: 'restore_failed' },
  { label: '存储满', value: 'storage_full' },
  { label: '加密失败', value: 'encryption_failed' },
  { label: '压缩失败', value: 'compression_failed' },
  { label: '上传失败', value: 'upload_failed' },
  { label: '验证失败', value: 'verification_failed' }
]
