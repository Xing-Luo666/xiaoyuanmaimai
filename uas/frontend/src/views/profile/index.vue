<template>
  <div class="profile-page">
    <el-row :gutter="16">
      <el-col :span="8">
        <el-card shadow="never">
          <div class="user-card">
            <el-avatar :size="80"><el-icon :size="40"><User /></el-icon></el-avatar>
            <h3>{{ user.nickname || user.username }}</h3>
            <p>{{ roleText }}</p>
            <el-divider />
            <ul class="info-list">
              <li><span>用户名</span><span>{{ user.username }}</span></li>
              <li><span>手机号</span><span>{{ user.phone || '-' }}</span></li>
              <li><span>邮箱</span><span>{{ user.email || '-' }}</span></li>
              <li><span>部门</span><span>{{ user.deptName || '-' }}</span></li>
              <li><span>注册时间</span><span>{{ user.createTime }}</span></li>
              <li><span>上次登录</span><span>{{ user.loginDate }}</span></li>
            </ul>
          </div>
        </el-card>
      </el-col>
      <el-col :span="16">
        <el-card shadow="never">
          <el-tabs v-model="activeTab">
            <el-tab-pane label="基本资料" name="info">
              <el-form ref="infoFormRef" :model="infoForm" :rules="infoRules" label-width="100px">
                <el-form-item label="昵称" prop="nickname"><el-input v-model="infoForm.nickname" /></el-form-item>
                <el-form-item label="手机号"><el-input v-model="infoForm.phone" /></el-form-item>
                <el-form-item label="邮箱"><el-input v-model="infoForm.email" /></el-form-item>
                <el-form-item label="性别"><el-radio-group v-model="infoForm.sex"><el-radio :value="1">男</el-radio><el-radio :value="2">女</el-radio><el-radio :value="0">未知</el-radio></el-radio-group></el-form-item>
                <el-form-item><el-button type="primary" @click="submitInfo">保存</el-button></el-form-item>
              </el-form>
            </el-tab-pane>
            <el-tab-pane label="修改密码" name="pwd">
              <el-form ref="pwdFormRef" :model="pwdForm" :rules="pwdRules" label-width="120px">
                <el-form-item label="原密码" prop="oldPassword"><el-input v-model="pwdForm.oldPassword" type="password" show-password /></el-form-item>
                <el-form-item label="新密码" prop="newPassword"><el-input v-model="pwdForm.newPassword" type="password" show-password /></el-form-item>
                <el-form-item label="确认新密码" prop="confirmPassword"><el-input v-model="pwdForm.confirmPassword" type="password" show-password /></el-form-item>
                <el-form-item><el-button type="primary" @click="submitPwd">保存</el-button></el-form-item>
              </el-form>
            </el-tab-pane>
          </el-tabs>
        </el-card>
      </el-col>
    </el-row>
  </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { User } from '@element-plus/icons-vue'
import { getProfile, updateProfile, changePwd } from '@/api/auth'

const activeTab = ref('info')
const user = ref({})
const infoFormRef = ref(null)
const pwdFormRef = ref(null)
const infoForm = reactive({ nickname: '', phone: '', email: '', sex: 0 })
const pwdForm = reactive({ oldPassword: '', newPassword: '', confirmPassword: '' })
const infoRules = { nickname: [{ required: true, message: '请输入昵称', trigger: 'blur' }] }
const pwdRules = {
  oldPassword: [{ required: true, message: '请输入原密码', trigger: 'blur' }],
  newPassword: [{ required: true, message: '请输入新密码', trigger: 'blur' }, { min: 6, message: '密码不能少于6位', trigger: 'blur' }],
  confirmPassword: [{ required: true, message: '请确认密码', trigger: 'blur' }, { validator: (rule, value, cb) => value === pwdForm.newPassword ? cb() : cb(new Error('两次密码不一致')), trigger: 'blur' }]
}
const roleText = computed(() => user.value.roleName || '管理员')

const loadProfile = async () => {
  try {
    const res = await getProfile()
    user.value = res.data || {}
    Object.assign(infoForm, { nickname: user.value.nickname, phone: user.value.phone, email: user.value.email, sex: user.value.sex })
  } catch (e) {}
}

const submitInfo = async () => {
  try { await infoFormRef.value.validate(); await updateProfile(infoForm); ElMessage.success('保存成功'); loadProfile() } catch (e) {}
}

const submitPwd = async () => {
  try { await pwdFormRef.value.validate(); await changePwd(pwdForm); ElMessage.success('修改成功，请重新登录'); pwdForm.oldPassword = ''; pwdForm.newPassword = ''; pwdForm.confirmPassword = '' } catch (e) {}
}

onMounted(() => { loadProfile() })
</script>

<style lang="scss" scoped>
.user-card { text-align: center; padding: 20px 0;
  h3 { margin: 12px 0 4px; }
  p { color: #909399; font-size: 13px; }
}
.info-list { text-align: left; list-style: none; padding: 0; margin: 0;
  li { display: flex; justify-content: space-between; padding: 8px 0; border-bottom: 1px solid #ebeef5; font-size: 13px;
    span:first-child { color: #909399; }
  }
}
</style>
