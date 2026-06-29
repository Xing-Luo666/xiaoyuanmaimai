import request from '@/utils/request'

export function login(data) {
  return request({ url: '/auth/login', method: 'post', data })
}

export function logout() {
  return request({ url: '/auth/logout', method: 'post' })
}

export function getUserInfo() {
  return request({ url: '/auth/userinfo', method: 'get' })
}

export function getRouters() {
  return request({ url: '/auth/routers', method: 'get' })
}

export function getCaptcha() {
  return request({ url: '/auth/captcha', method: 'get' })
}

export function updateProfile(data) {
  return request({ url: '/auth/profile', method: 'put', data })
}

export function getProfile() {
  return request({ url: '/auth/profile', method: 'get' })
}

export function changePwd(data) {
  return request({ url: '/auth/password', method: 'put', data })
}

export function changePassword(data) {
  return request({ url: '/auth/password', method: 'put', data })
}

export function uploadAvatar(formData) {
  return request({
    url: '/auth/avatar',
    method: 'post',
    data: formData,
    headers: { 'Content-Type': 'multipart/form-data' }
  })
}

// 用户注册（自然人用户自助注册UAS账号）
export function register(data) {
  return request({ url: '/auth/register', method: 'post', data })
}

// 检查手机号是否已注册
export function checkPhone(phone) {
  return request({ url: '/auth/check-phone', method: 'get', params: { phone } })
}
