import request from '@/utils/request'

// 自然人用户
export function listPersonalUser(query) {
  return request({ url: '/uas/user/list', method: 'get', params: query })
}
export function getPersonalUser(id) {
  return request({ url: '/uas/user/' + id, method: 'get' })
}
export function addPersonalUser(data) {
  return request({ url: '/uas/user', method: 'post', data })
}
export function updatePersonalUser(data) {
  return request({ url: '/uas/user', method: 'put', data })
}
export function deletePersonalUser(id) {
  return request({ url: '/uas/user/' + id, method: 'delete' })
}
export function changePersonalUserStatus(id, status) {
  return request({ url: '/uas/user/' + id + '/status', method: 'put', data: { status } })
}

// 法人用户
export function listCorpUser(query) {
  return request({ url: '/uas/corp/list', method: 'get', params: query })
}
export function getCorpUser(id) {
  return request({ url: '/uas/corp/' + id, method: 'get' })
}
export function addCorpUser(data) {
  return request({ url: '/uas/corp', method: 'post', data })
}
export function updateCorpUser(data) {
  return request({ url: '/uas/corp', method: 'put', data })
}
export function deleteCorpUser(id) {
  return request({ url: '/uas/corp/' + id, method: 'delete' })
}
export function changeCorpUserStatus(id, status) {
  return request({ url: '/uas/corp/' + id + '/status', method: 'put', data: { status } })
}

// 审核管理
export function listAudit(query) {
  return request({ url: '/uas/audit/list', method: 'get', params: query })
}
export function auditUser(id, data) {
  return request({ url: '/uas/audit/user/' + id, method: 'put', data })
}
export function auditCorp(id, data) {
  return request({ url: '/uas/audit/corp/' + id, method: 'put', data })
}

// 应用管理
export function listApp(query) {
  return request({ url: '/uas/app/list', method: 'get', params: query })
}
export function getApp(id) {
  return request({ url: '/uas/app/' + id, method: 'get' })
}
export function addApp(data) {
  return request({ url: '/uas/app', method: 'post', data })
}
export function updateApp(data) {
  return request({ url: '/uas/app', method: 'put', data })
}
export function deleteApp(id) {
  return request({ url: '/uas/app/' + id, method: 'delete' })
}
export function resetAppSecret(id) {
  return request({ url: '/uas/app/' + id + '/resetSecret', method: 'put' })
}

// 授权管理
export function listGrant(query) {
  return request({ url: '/uas/grant/list', method: 'get', params: query })
}
export function deleteGrant(id) {
  return request({ url: '/uas/grant/' + id, method: 'delete' })
}
