import axios from 'axios'
import { ElMessage } from 'element-plus'

export const baseURL = (import.meta.env.MODE === 'production')
  ? '/api/v1/' // 生产环境使用相对路径，由nginx代理
  : 'http://localhost:8080' // 开发环境使用本地开发服务器地址


const instance = axios.create({
  baseURL: baseURL,
  timeout: 60000,
  headers: {
    'Content-Type': 'application/json',
  },
})

instance.interceptors.request.use(
  (config) => {
    return config
  },
  (error) => {
    return Promise.reject(error)
  }
)

instance.interceptors.response.use(
  (response) => {
    if (response.data.code !== 0) {
      ElMessage.error(response.data.msg || '请求失败')
      return Promise.reject(response.data)
    }
    return response.data
  },
  (error) => {
    const message = error.response?.data?.msg || error.message || '请求失败'
    ElMessage.error(message)
    return Promise.reject(error)
  }
)

export default instance
