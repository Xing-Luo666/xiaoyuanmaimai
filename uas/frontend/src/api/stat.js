import request from '@/utils/request'

// 统计分析
export function statAccount() {
  return request({ url: '/stat/account', method: 'get' })
}
export function statLogin() {
  return request({ url: '/stat/login', method: 'get' })
}
export function statApi() {
  return request({ url: '/stat/api', method: 'get' })
}
export function statSms() {
  return request({ url: '/stat/sms', method: 'get' })
}

// 统计概览（Dashboard & 统计页面）
export function getStatOverview() {
  return request({ url: '/stat/overview', method: 'get' })
}
export function getStatTrend() {
  return request({ url: '/stat/trend', method: 'get' })
}
export function getStatAppType() {
  return request({ url: '/stat/appType', method: 'get' })
}
export function getStatTopApps() {
  return request({ url: '/stat/topApps', method: 'get' })
}
export function getStatActiveUsers() {
  return request({ url: '/stat/activeUsers', method: 'get' })
}

// 日志
export function listLoginLog(query) {
  return request({ url: '/log/loginLog/list', method: 'get', params: query })
}
export function cleanLoginLog() {
  return request({ url: '/log/loginLog/clean', method: 'delete' })
}
export function listAuditLog(query) {
  return request({ url: '/log/auditLog/list', method: 'get', params: query })
}
export function cleanAuditLog() {
  return request({ url: '/log/auditLog/clean', method: 'delete' })
}
export function listSmsLog(query) {
  return request({ url: '/log/smsLog/list', method: 'get', params: query })
}
export function cleanSmsLog() {
  return request({ url: '/log/smsLog/clean', method: 'delete' })
}
