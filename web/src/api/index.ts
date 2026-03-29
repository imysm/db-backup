import axios from 'axios'

const api = axios.create({
  baseURL: '/api/v1',
  timeout: 30000
})

// 请求拦截器：发送 X-API-Key header
api.interceptors.request.use(
  config => {
    const apiKey = localStorage.getItem('api_key')
    if (apiKey) {
      config.headers['X-API-Key'] = apiKey
    }
    return config
  },
  error => {
    return Promise.reject(error)
  }
)

// 响应拦截器
api.interceptors.response.use(
  response => {
    return response.data
  },
  error => {
    if (error.response?.status === 401) {
      localStorage.removeItem('api_key')
      window.location.href = '/login'
    }
    return Promise.reject(error)
  }
)

export default api

// 登录验证（通过发起一个需要认证的请求来验证 API Key）
export const loginAPI = {
  verify: (apiKey: string) => {
    const tempApi = axios.create({
      baseURL: '/api/v1',
      timeout: 10000,
      headers: { 'X-API-Key': apiKey }
    })
    return tempApi.get('/stats')
  }
}

// 任务 API
export const jobAPI = {
  list: (params?: any) => api.get('/jobs', { params }),
  get: (id: number) => api.get(`/jobs/${id}`),
  create: (data: any) => api.post('/jobs', data),
  update: (id: number, data: any) => api.put(`/jobs/${id}`, data),
  delete: (id: number) => api.delete(`/jobs/${id}`),
  run: (id: number) => api.post(`/jobs/${id}/run`),
  testConnection: (id: number) => api.post(`/jobs/${id}/test-connection`),
  nextRuns: (id: number, cron?: string) => api.get(`/jobs/${id}/next-runs`, { params: { cron } }),
  batchUpdate: (ids: number[], data: any) => api.put('/jobs/batch', { ids, ...data }),
  batchDelete: (ids: number[]) => api.post('/jobs/batch-delete', { ids }),
}

// 记录 API
export const recordAPI = {
  list: (params?: any) => api.get('/records', { params }),
  get: (id: number) => api.get(`/records/${id}`),
  delete: (id: number) => api.delete(`/records/${id}`),
  download: (id: number) => api.get(`/records/${id}/download`, { responseType: 'blob' })
}

// 验证 API
export const verifyAPI = {
  verify: (id: number) => api.post(`/verify/${id}`),
  testRestore: (id: number) => api.post(`/verify/${id}/restore`),
  batch: (ids: number[]) => api.post('/verify/batch', { ids })
}

// 恢复 API
export const restoreAPI = {
  list: (params?: any) => api.get('/restore/list', { params }),
  restore: (data: any) => api.post('/restore', data),
  validate: (id: number) => api.get(`/restore/validate/${id}`),
  validatePOST: (id: number, data?: any) => api.post(`/restore/validate/${id}`, data || {}),
  detail: (id: number) => api.get(`/restore/${id}`)
}

// 统计 API
export const statsAPI = {
  get: () => api.get('/stats')
}
