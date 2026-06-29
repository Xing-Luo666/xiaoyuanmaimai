import request from '@/utils/request'

// 系统管理员
export function listSysUser(query) {
  return request({ url: '/system/user/list', method: 'get', params: query })
}
export function getSysUser(id) {
  return request({ url: '/system/user/' + id, method: 'get' })
}
export function addSysUser(data) {
  return request({ url: '/system/user', method: 'post', data })
}
export function updateSysUser(data) {
  return request({ url: '/system/user', method: 'put', data })
}
export function deleteSysUser(id) {
  return request({ url: '/system/user/' + id, method: 'delete' })
}
export function resetSysUserPwd(id, password) {
  return request({ url: '/system/user/' + id + '/resetPwd', method: 'put', data: { password } })
}
export function changeSysUserStatus(id, status) {
  return request({ url: '/system/user/' + id + '/status', method: 'put', data: { status } })
}

// 角色
export function listRole(query) {
  return request({ url: '/system/role/list', method: 'get', params: query })
}
export function getRole(id) {
  return request({ url: '/system/role/' + id, method: 'get' })
}
export function addRole(data) {
  return request({ url: '/system/role', method: 'post', data })
}
export function updateRole(data) {
  return request({ url: '/system/role', method: 'put', data })
}
export function deleteRole(id) {
  return request({ url: '/system/role/' + id, method: 'delete' })
}

// 菜单
export function listMenu(query) {
  return request({ url: '/system/menu/list', method: 'get', params: query })
}
export function getMenu(id) {
  return request({ url: '/system/menu/' + id, method: 'get' })
}
export function addMenu(data) {
  return request({ url: '/system/menu', method: 'post', data })
}
export function updateMenu(data) {
  return request({ url: '/system/menu', method: 'put', data })
}
export function deleteMenu(id) {
  return request({ url: '/system/menu/' + id, method: 'delete' })
}
export function menuTreeSelect() {
  return request({ url: '/system/menu/treeselect', method: 'get' })
}

// 操作日志
export function listOperLog(query) {
  return request({ url: '/log/operlog/list', method: 'get', params: query })
}
export function cleanOperLog() {
  return request({ url: '/log/operlog/clean', method: 'delete' })
}

// 登录日志
export function listLoginLog(query) {
  return request({ url: '/log/loginlog/list', method: 'get', params: query })
}
export function cleanLoginLog() {
  return request({ url: '/log/loginlog/clean', method: 'delete' })
}
