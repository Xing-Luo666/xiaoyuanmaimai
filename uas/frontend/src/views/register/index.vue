<template>
  <div class="register-container">
    <div class="register-bg">
      <div class="bg-shape shape1"></div>
      <div class="bg-shape shape2"></div>
      <div class="bg-shape shape3"></div>
    </div>
    <div class="register-box">
      <div class="register-left">
        <div class="welcome">
          <h1>统一身份认证管理系统</h1>
          <p>Unified Authentication Service</p>
          <p class="desc">注册UAS账号，一次注册即可登录所有接入应用<br/>校园二手交易平台等第三方应用可通过OAuth2直接登录</p>
          <ul class="features">
            <li>📱 手机号一键注册</li>
            <li>🔐 BCrypt加密存储密码</li>
            <li>⚡ 注册即登录，无需重复操作</li>
            <li>🛡️ 多应用统一身份认证</li>
          </ul>
        </div>
      </div>
      <div class="register-right">
        <el-form ref="registerFormRef" :model="registerForm" :rules="rules" class="register-form" @keyup.enter="handleRegister">
          <h2 class="title">用户注册</h2>

          <el-form-item prop="phone">
            <el-input v-model="registerForm.phone" placeholder="请输入手机号" size="large" :prefix-icon="Iphone" maxlength="11" @blur="handlePhoneBlur">
              <template #append>
                <el-button :loading="phoneChecking" @click="handlePhoneBlur">检查</el-button>
              </template>
            </el-input>
          </el-form-item>

          <el-form-item prop="password">
            <el-input v-model="registerForm.password" type="password" placeholder="请设置密码（至少6位）" size="large" :prefix-icon="Lock" show-password />
          </el-form-item>

          <el-form-item prop="confirmPassword">
            <el-input v-model="registerForm.confirmPassword" type="password" placeholder="请再次输入密码" size="large" :prefix-icon="Lock" show-password />
          </el-form-item>

          <el-form-item prop="nickname">
            <el-input v-model="registerForm.nickname" placeholder="昵称（选填，默认使用手机号后4位）" size="large" :prefix-icon="User" maxlength="20" />
          </el-form-item>

          <el-form-item prop="realName">
            <el-input v-model="registerForm.realName" placeholder="真实姓名（选填，便于实名认证）" size="large" :prefix-icon="Postcard" maxlength="20" />
          </el-form-item>

          <el-form-item prop="email">
            <el-input v-model="registerForm.email" placeholder="邮箱（选填）" size="large" :prefix-icon="Message" maxlength="50" />
          </el-form-item>

          <el-form-item>
            <el-button type="primary" size="large" class="register-btn" :loading="loading" @click="handleRegister">注 册</el-button>
          </el-form-item>

          <div class="tips">
            <span>已有账号？</span>
            <el-link type="primary" :underline="false" @click="$router.push('/login')">前往登录</el-link>
          </div>
        </el-form>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { User, Lock, Iphone, Postcard, Message } from '@element-plus/icons-vue'
import { register, checkPhone } from '@/api/auth'

const router = useRouter()
const registerFormRef = ref(null)
const loading = ref(false)
const phoneChecking = ref(false)
const phoneAvailable = ref(false)

const registerForm = reactive({
  phone: '',
  password: '',
  confirmPassword: '',
  nickname: '',
  realName: '',
  email: ''
})

// 手机号校验
const validatePhone = (rule, value, callback) => {
  if (!value) {
    callback(new Error('请输入手机号'))
  } else if (!/^1[3-9]\d{9}$/.test(value)) {
    callback(new Error('手机号格式不正确'))
  } else {
    callback()
  }
}

// 确认密码校验
const validateConfirmPassword = (rule, value, callback) => {
  if (!value) {
    callback(new Error('请再次输入密码'))
  } else if (value !== registerForm.password) {
    callback(new Error('两次输入的密码不一致'))
  } else {
    callback()
  }
}

// 邮箱校验
const validateEmail = (rule, value, callback) => {
  if (!value) {
    callback()
  } else if (!/^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$/.test(value)) {
    callback(new Error('邮箱格式不正确'))
  } else {
    callback()
  }
}

const rules = {
  phone: [{ required: true, validator: validatePhone, trigger: 'blur' }],
  password: [
    { required: true, message: '请设置密码', trigger: 'blur' },
    { min: 6, message: '密码至少6位', trigger: 'blur' }
  ],
  confirmPassword: [{ required: true, validator: validateConfirmPassword, trigger: 'blur' }],
  email: [{ validator: validateEmail, trigger: 'blur' }]
}

// 手机号失焦时检查是否已注册
const handlePhoneBlur = async () => {
  if (!/^1[3-9]\d{9}$/.test(registerForm.phone)) {
    phoneAvailable.value = false
    return
  }
  phoneChecking.value = true
  try {
    const res = await checkPhone(registerForm.phone)
    if (res.data && res.data.available) {
      ElMessage.success('手机号可注册')
      phoneAvailable.value = true
    } else {
      ElMessage.warning('该手机号已注册')
      phoneAvailable.value = false
    }
  } catch (e) {
    phoneAvailable.value = false
  } finally {
    phoneChecking.value = false
  }
}

const handleRegister = async () => {
  try {
    await registerFormRef.value.validate()
    loading.value = true
    const res = await register({
      phone: registerForm.phone,
      password: registerForm.password,
      nickname: registerForm.nickname,
      realName: registerForm.realName,
      email: registerForm.email
    })
    ElMessage.success(res.msg || '注册成功')
    // 注册成功后自动登录：保存token后跳转首页
    if (res.data && res.data.token) {
      localStorage.setItem('uas_token', res.data.token)
      localStorage.setItem('uas_user', JSON.stringify({
        userId: res.data.userId,
        username: res.data.phone,
        nickname: res.data.nickname,
        role: 'uas_user',
        authLevel: res.data.authLevel || 'L1'
      }))
      setTimeout(() => router.push('/'), 800)
    } else {
      // 没有自动登录则跳转登录页
      setTimeout(() => router.push('/login'), 800)
    }
  } catch (e) {
    if (e && e.message) ElMessage.error(e.message)
  } finally {
    loading.value = false
  }
}
</script>

<style lang="scss" scoped>
.register-container {
  position: relative;
  height: 100%;
  width: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, #1e3c72 0%, #2a5298 50%, #409eff 100%);
  overflow: hidden;
}

.register-bg {
  position: absolute;
  inset: 0;
  pointer-events: none;
  .bg-shape {
    position: absolute;
    border-radius: 50%;
    opacity: 0.08;
    background: #fff;
  }
  .shape1 { width: 400px; height: 400px; top: -150px; left: -150px; }
  .shape2 { width: 300px; height: 300px; bottom: -100px; right: -100px; }
  .shape3 { width: 200px; height: 200px; top: 40%; left: 30%; }
}

.register-box {
  position: relative;
  z-index: 10;
  display: flex;
  width: 920px;
  max-height: 92vh;
  background: #fff;
  border-radius: 12px;
  overflow: hidden;
  box-shadow: 0 20px 60px rgba(0,0,0,0.3);
}

.register-left {
  flex: 1;
  background: linear-gradient(135deg, #1e3c72 0%, #2a5298 100%);
  color: #fff;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 40px;
  .welcome {
    text-align: left;
    h1 { font-size: 24px; margin-bottom: 12px; }
    p { font-size: 14px; opacity: 0.85; margin-bottom: 8px; }
    .desc { font-size: 13px; line-height: 1.8; opacity: 0.7; margin-top: 16px; }
    .features {
      list-style: none;
      padding: 0;
      margin: 24px 0 0 0;
      li {
        font-size: 13px;
        line-height: 2;
        opacity: 0.85;
      }
    }
  }
}

.register-right {
  width: 440px;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 30px 40px;
  overflow-y: auto;
}

.register-form {
  width: 100%;
  .title {
    text-align: center;
    margin-bottom: 24px;
    color: #303133;
    font-size: 22px;
  }
  .register-btn {
    width: 100%;
    letter-spacing: 4px;
  }
  .tips {
    text-align: center;
    color: #909399;
    font-size: 13px;
    margin-top: 10px;
    .el-link { margin-left: 4px; }
  }
}

@media (max-width: 768px) {
  .register-box { width: 90%; height: auto; flex-direction: column; }
  .register-left { display: none; }
  .register-right { width: 100%; padding: 30px; }
}
</style>
