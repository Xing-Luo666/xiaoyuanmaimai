<template>
  <div class="authorize-page">
    <div class="authorize-box">
      <div class="header">
        <h1>统一身份认证中心</h1>
        <p>Unified Authentication Service</p>
      </div>
      <div class="content" v-loading="loading">
        <template v-if="appInfo">
          <div class="app-info">
            <div class="app-icon">
              <el-icon :size="40"><Connection /></el-icon>
            </div>
            <div class="app-detail">
              <h2>{{ appInfo.appName }}</h2>
              <p>{{ appInfo.description || '该应用请求获取您的身份信息' }}</p>
              <p class="app-id">AppID: {{ appInfo.appId }}</p>
            </div>
          </div>
          <el-divider />
          <div class="scope-list">
            <p class="title">该应用将获得以下权限：</p>
            <ul>
              <li><el-icon><CircleCheck /></el-icon> 获取您的基本信息（手机号、姓名、昵称）</li>
              <li><el-icon><CircleCheck /></el-icon> 获取您的认证等级</li>
              <li><el-icon><CircleCheck /></el-icon> 获取您的头像和邮箱</li>
            </ul>
          </div>
          <el-divider />

          <!-- 已登录：显示授权确认 -->
          <div class="confirm-section" v-if="uasUser">
            <p class="title">您正在以以下身份授权</p>
            <div class="user-info">
              <el-avatar :size="40"><el-icon><User /></el-icon></el-avatar>
              <div>
                <p class="username">{{ uasUser.nickname || uasUser.realName || '已登录用户' }}</p>
                <p class="user-type">{{ uasUser.userType === 'personal' ? '个体用户' : '企业用户' }}</p>
              </div>
            </div>
            <el-button type="primary" size="large" class="btn-block" :loading="confirmLoading" @click="handleConfirm">同意授权</el-button>
            <el-button size="large" class="btn-block mt10" @click="handleCancel">拒绝</el-button>
          </div>

          <!-- 未登录：登录 / 注册 切换 -->
          <div v-else>
            <el-tabs v-model="activeTab" class="auth-tabs">
              <el-tab-pane label="登录" name="login">
                <el-form :model="loginForm" label-width="0" @keyup.enter="handleLogin">
                  <el-form-item>
                    <el-input v-model="loginForm.username" placeholder="手机号 / 企业用户名" :prefix-icon="User" size="large" />
                  </el-form-item>
                  <el-form-item>
                    <el-input v-model="loginForm.password" type="password" placeholder="密码" :prefix-icon="Lock" size="large" show-password />
                  </el-form-item>
                  <el-form-item>
                    <el-radio-group v-model="loginForm.userType">
                      <el-radio value="personal">个体用户</el-radio>
                      <el-radio value="corp">企业用户</el-radio>
                    </el-radio-group>
                  </el-form-item>
                  <el-form-item>
                    <div class="captcha-row">
                      <el-input v-model="loginForm.code" placeholder="验证码" :prefix-icon="Picture" size="large" maxlength="6" />
                      <img v-if="captcha.img" :src="captcha.img" class="captcha-img" @click="getCaptchaCode" alt="验证码" title="点击刷新" />
                      <div v-else class="captcha-img captcha-placeholder" @click="getCaptchaCode">点击加载</div>
                    </div>
                  </el-form-item>
                  <el-button type="primary" size="large" class="btn-block" :loading="loginLoading" @click="handleLogin">登 录</el-button>
                </el-form>
              </el-tab-pane>

              <el-tab-pane label="注册" name="register">
                <el-form ref="registerFormRef" :model="registerForm" :rules="registerRules" label-width="0" @keyup.enter="handleRegister">
                  <el-form-item prop="phone">
                    <el-input v-model="registerForm.phone" placeholder="手机号" :prefix-icon="Iphone" size="large" maxlength="11" />
                  </el-form-item>
                  <el-form-item prop="password">
                    <el-input v-model="registerForm.password" type="password" placeholder="设置密码（至少6位）" :prefix-icon="Lock" size="large" show-password />
                  </el-form-item>
                  <el-form-item prop="confirmPassword">
                    <el-input v-model="registerForm.confirmPassword" type="password" placeholder="再次输入密码" :prefix-icon="Lock" size="large" show-password />
                  </el-form-item>
                  <el-form-item prop="nickname">
                    <el-input v-model="registerForm.nickname" placeholder="昵称（选填）" :prefix-icon="User" size="large" maxlength="20" />
                  </el-form-item>
                  <el-form-item>
                    <div class="captcha-row">
                      <el-input v-model="registerForm.code" placeholder="验证码" :prefix-icon="Picture" size="large" maxlength="6" />
                      <img v-if="captcha.img" :src="captcha.img" class="captcha-img" @click="getCaptchaCode" alt="验证码" title="点击刷新" />
                      <div v-else class="captcha-img captcha-placeholder" @click="getCaptchaCode">点击加载</div>
                    </div>
                  </el-form-item>
                  <el-button type="primary" size="large" class="btn-block" :loading="registerLoading" @click="handleRegister">注 册</el-button>
                </el-form>
              </el-tab-pane>
            </el-tabs>
          </div>
        </template>
      </div>
      <div class="footer">
        <p>授权后该应用可访问您的UAS账户信息，请确认您信任该应用</p>
        <p class="copyright">© 2026 统一身份认证管理系统</p>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { User, Lock, Connection, CircleCheck, Picture, Iphone } from '@element-plus/icons-vue'
import { getAuthorizeInfo, confirmAuthorize, uasLogin } from '@/api/oauth'
import { getCaptcha, register } from '@/api/auth'

const route = useRoute()
const router = useRouter()

const loading = ref(true)
const loginLoading = ref(false)
const registerLoading = ref(false)
const confirmLoading = ref(false)
const appInfo = ref(null)
const uasUser = ref(null)
const activeTab = ref('login')
const registerFormRef = ref(null)

const loginForm = reactive({
  username: '',
  password: '',
  code: '',
  uuid: '',
  userType: 'personal'
})

const registerForm = reactive({
  phone: '',
  password: '',
  confirmPassword: '',
  nickname: '',
  code: '',
  uuid: ''
})

const captcha = reactive({ img: '', uuid: '' })

const validatePhone = (rule, value, callback) => {
  if (!value) callback(new Error('请输入手机号'))
  else if (!/^1[3-9]\d{9}$/.test(value)) callback(new Error('手机号格式不正确'))
  else callback()
}
const validateConfirmPassword = (rule, value, callback) => {
  if (!value) callback(new Error('请再次输入密码'))
  else if (value !== registerForm.password) callback(new Error('两次输入的密码不一致'))
  else callback()
}
const registerRules = {
  phone: [{ required: true, validator: validatePhone, trigger: 'blur' }],
  password: [
    { required: true, message: '请设置密码', trigger: 'blur' },
    { min: 6, message: '密码至少6位', trigger: 'blur' }
  ],
  confirmPassword: [{ required: true, validator: validateConfirmPassword, trigger: 'blur' }]
}

const getCaptchaCode = async () => {
  try {
    const res = await getCaptcha()
    captcha.img = res.data.img
    captcha.uuid = res.data.uuid
  } catch (e) {}
}

const loadAppInfo = async () => {
  try {
    const res = await getAuthorizeInfo({
      client_id: route.query.client_id,
      redirect_uri: route.query.redirect_uri,
      response_type: route.query.response_type,
      state: route.query.state,
      scope: route.query.scope
    })
    appInfo.value = res.data
  } catch (e) {
    ElMessage.error('应用信息获取失败')
  } finally {
    loading.value = false
  }
}

const handleLogin = async () => {
  if (!loginForm.username || !loginForm.password) {
    ElMessage.warning('请输入账号密码')
    return
  }
  if (!loginForm.code) {
    ElMessage.warning('请输入验证码')
    return
  }
  loginLoading.value = true
  try {
    const res = await uasLogin({
      username: loginForm.username,
      password: loginForm.password,
      code: loginForm.code,
      uuid: captcha.uuid,
      userType: loginForm.userType
    })
    uasUser.value = {
      userId: res.data.userId,
      userType: res.data.userType,
      nickname: res.data.nickname,
      realName: res.data.realName
    }
    ElMessage.success('登录成功')
  } catch (e) {
    getCaptchaCode()
    loginForm.code = ''
  } finally {
    loginLoading.value = false
  }
}

const handleRegister = async () => {
  try {
    await registerFormRef.value.validate()
    if (!registerForm.code) {
      ElMessage.warning('请输入验证码')
      return
    }
    registerLoading.value = true
    const res = await register({
      phone: registerForm.phone,
      password: registerForm.password,
      nickname: registerForm.nickname,
      code: registerForm.code,
      uuid: captcha.uuid
    })
    ElMessage.success(res.msg || '注册成功')
    // 注册成功后自动登录到授权确认状态
    if (res.data && res.data.userId) {
      uasUser.value = {
        userId: res.data.userId,
        userType: 'personal',
        nickname: res.data.nickname || registerForm.nickname,
        realName: ''
      }
    }
  } catch (e) {
    if (e && e.message) ElMessage.error(e.message)
    getCaptchaCode()
    registerForm.code = ''
  } finally {
    registerLoading.value = false
  }
}

const handleConfirm = async () => {
  confirmLoading.value = true
  try {
    const res = await confirmAuthorize({
      client_id: route.query.client_id,
      redirect_uri: route.query.redirect_uri,
      state: route.query.state,
      scope: route.query.scope,
      user_id: uasUser.value.userId,
      user_type: uasUser.value.userType
    })
    // 跳转到回调地址
    window.location.href = res.data.redirect_url
  } catch (e) {} finally {
    confirmLoading.value = false
  }
}

const handleCancel = () => {
  const redirect = route.query.redirect_uri
  if (redirect) {
    const sep = redirect.includes('?') ? '&' : '?'
    window.location.href = redirect + sep + 'error=access_denied'
  } else {
    router.push('/login')
  }
}

onMounted(() => {
  if (!route.query.client_id || !route.query.redirect_uri) {
    ElMessage.error('参数缺失')
    return
  }
  loadAppInfo()
  getCaptchaCode()
})
</script>

<style lang="scss" scoped>
.authorize-page {
  min-height: 100%;
  background: linear-gradient(135deg, #1e3c72 0%, #2a5298 100%);
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 20px;
}

.authorize-box {
  width: 480px;
  background: #fff;
  border-radius: 12px;
  overflow: hidden;
  box-shadow: 0 20px 60px rgba(0,0,0,0.3);

  .header {
    background: linear-gradient(135deg, #1e3c72, #2a5298);
    color: #fff;
    padding: 24px;
    text-align: center;
    h1 { font-size: 22px; margin-bottom: 6px; }
    p { font-size: 12px; opacity: 0.8; }
  }

  .content { padding: 24px; min-height: 280px; }

  .app-info {
    display: flex;
    gap: 16px;
    align-items: flex-start;
    .app-icon {
      width: 60px;
      height: 60px;
      border-radius: 8px;
      background: #ecf5ff;
      color: #409eff;
      display: flex;
      align-items: center;
      justify-content: center;
    }
    .app-detail {
      flex: 1;
      h2 { font-size: 18px; margin-bottom: 6px; }
      p { color: #606266; font-size: 13px; margin-bottom: 4px; }
      .app-id { color: #909399; font-size: 12px; }
    }
  }

  .scope-list {
    .title { font-weight: 600; margin-bottom: 10px; }
    ul { padding-left: 0; }
    li {
      display: flex;
      align-items: center;
      gap: 8px;
      padding: 6px 0;
      color: #606266;
      font-size: 14px;
      .el-icon { color: #67c23a; }
    }
  }

  .auth-tabs {
    :deep(.el-tabs__header) { margin-bottom: 16px; }
    :deep(.el-tabs__nav) { width: 100%; }
    :deep(.el-tabs__item) { width: 50%; text-align: center; }
  }

  .confirm-section {
    .title { font-weight: 600; margin-bottom: 16px; text-align: center; }
  }

  .user-info {
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 12px;
    background: #f5f7fa;
    border-radius: 8px;
    margin-bottom: 16px;
    .username { font-weight: 600; }
    .user-type { font-size: 12px; color: #909399; }
  }

  .captcha-row {
    display: flex;
    gap: 10px;
    width: 100%;
    .el-input { flex: 1; }
    .captcha-img {
      width: 120px;
      height: 40px;
      cursor: pointer;
      border-radius: 4px;
      border: 1px solid #dcdfe6;
      object-fit: cover;
    }
    .captcha-placeholder {
      display: flex;
      align-items: center;
      justify-content: center;
      color: #909399;
      font-size: 12px;
      background: #f5f7fa;
    }
  }

  .btn-block { width: 100%; }
  .mt10 { margin-top: 10px; }

  .footer {
    padding: 16px 24px;
    background: #f5f7fa;
    text-align: center;
    p { font-size: 12px; color: #909399; margin-bottom: 4px; }
    .copyright { color: #c0c4cc; }
  }
}
</style>
