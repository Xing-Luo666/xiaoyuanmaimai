import request from '@/utils/request'

// UAS用户登录（OAuth2流程使用）
export function uasLogin(data) {
  return request({ url: '/uas/login', method: 'post', data })
}

// 获取授权页应用信息
export function getAuthorizeInfo(params) {
  return request({ url: '/oauth/authorize', method: 'get', params })
}

// 用户确认授权
export function confirmAuthorize(data) {
  return request({ url: '/oauth/authorize', method: 'post', data })
}

// 应用后端用 code 换取 access_token
export function exchangeToken(data) {
  return request({
    url: '/oauth/token',
    method: 'post',
    data,
    headers: { 'Content-Type': 'application/x-www-form-urlencoded' }
  })
}
