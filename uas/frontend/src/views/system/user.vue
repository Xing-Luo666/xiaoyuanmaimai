<template>
  <div>
    <el-card class="search-card" shadow="never">
      <el-form :inline="true" :model="query" class="search-form">
        <el-form-item label="用户名">
          <el-input v-model="query.username" placeholder="请输入用户名" clearable @keyup.enter="loadList" />
        </el-form-item>
        <el-form-item label="手机号">
          <el-input v-model="query.phone" placeholder="请输入手机号" clearable @keyup.enter="loadList" />
        </el-form-item>
        <el-form-item label="状态">
          <el-select v-model="query.status" placeholder="全部" clearable style="width:120px">
            <el-option label="启用" value="1" />
            <el-option label="禁用" value="0" />
          </el-select>
        </el-form-item>
        <el-form-item>
          <el-button type="primary" :icon="Search" @click="loadList">查询</el-button>
          <el-button :icon="Refresh" @click="resetQuery">重置</el-button>
        </el-form-item>
      </el-form>
    </el-card>

    <el-card shadow="never">
      <div class="mb10">
        <el-button type="primary" :icon="Plus" @click="handleAdd">新增</el-button>
      </div>
      <el-table v-loading="loading" :data="list" border>
        <el-table-column type="index" label="序号" width="60" />
        <el-table-column prop="username" label="用户名" width="140" />
        <el-table-column prop="nickname" label="昵称" width="140" />
        <el-table-column prop="email" label="邮箱" width="200" show-overflow-tooltip />
        <el-table-column prop="phone" label="手机号" width="130" />
        <el-table-column label="性别" width="80">
          <template #default="{row}">{{ ['','男','女'][row.sex] || '未知' }}</template>
        </el-table-column>
        <el-table-column label="状态" width="100">
          <template #default="{row}">
            <el-switch v-model="row.status" :active-value="1" :inactive-value="0" @change="handleStatusChange(row)" />
          </template>
        </el-table-column>
        <el-table-column prop="createTime" label="创建时间" width="170" :formatter="formatTimeCol" />
        <el-table-column label="操作" width="220" fixed="right">
          <template #default="{row}">
            <el-button link type="primary" @click="handleEdit(row)">编辑</el-button>
            <el-button link type="warning" @click="handleResetPwd(row)">重置密码</el-button>
            <el-button link type="danger" :disabled="row.username === 'admin'" @click="handleDelete(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
      <div class="pagination-wrap">
        <el-pagination v-model:current-page="query.pageNum" v-model:page-size="query.pageSize" :total="total" layout="total, prev, pager, next, jumper" @current-change="loadList" />
      </div>
    </el-card>

    <el-dialog v-model="dialog.visible" :title="dialog.title" width="600px">
      <el-form ref="formRef" :model="form" :rules="rules" label-width="100px">
        <el-form-item label="用户名" prop="username">
          <el-input v-model="form.username" :disabled="dialog.type === 'edit'" />
        </el-form-item>
        <el-form-item label="密码" prop="password" v-if="dialog.type === 'add'">
          <el-input v-model="form.password" type="password" show-password />
        </el-form-item>
        <el-form-item label="昵称" prop="nickname">
          <el-input v-model="form.nickname" />
        </el-form-item>
        <el-row>
          <el-col :span="12">
            <el-form-item label="手机号"><el-input v-model="form.phone" /></el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="邮箱"><el-input v-model="form.email" /></el-form-item>
          </el-col>
        </el-row>
        <el-form-item label="性别">
          <el-radio-group v-model="form.sex">
            <el-radio :value="1">男</el-radio>
            <el-radio :value="2">女</el-radio>
            <el-radio :value="0">未知</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="状态">
          <el-radio-group v-model="form.status">
            <el-radio :value="1">启用</el-radio>
            <el-radio :value="0">禁用</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="备注"><el-input v-model="form.remark" type="textarea" :rows="2" /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialog.visible = false">取消</el-button>
        <el-button type="primary" @click="submitForm">确定</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="pwdDialog.visible" title="重置密码" width="400px">
      <el-form label-width="100px">
        <el-form-item label="新密码">
          <el-input v-model="pwdDialog.password" type="password" show-password />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="pwdDialog.visible = false">取消</el-button>
        <el-button type="primary" @click="submitResetPwd">确定</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Search, Refresh, Plus } from '@element-plus/icons-vue'
import { listSysUser, addSysUser, updateSysUser, deleteSysUser, resetSysUserPwd, changeSysUserStatus } from '@/api/system'
import { formatTimeCol } from '@/utils/format'

const loading = ref(false)
const list = ref([])
const total = ref(0)
const query = reactive({ pageNum: 1, pageSize: 10, username: '', phone: '', status: '' })
const formRef = ref(null)
const dialog = reactive({ visible: false, title: '', type: 'add' })
const pwdDialog = reactive({ visible: false, id: null, password: '' })
const form = reactive({ id: null, username: '', password: '', nickname: '', email: '', phone: '', sex: 0, status: 1, remark: '' })
const rules = {
  username: [{ required: true, message: '请输入用户名', trigger: 'blur' }],
  password: [{ required: true, message: '请输入密码', trigger: 'blur' }],
  nickname: [{ required: true, message: '请输入昵称', trigger: 'blur' }]
}

const loadList = async () => {
  loading.value = true
  try {
    const res = await listSysUser(query)
    list.value = res.rows || []
    total.value = res.total || 0
  } finally { loading.value = false }
}

const resetQuery = () => { query.username = ''; query.phone = ''; query.status = ''; query.pageNum = 1; loadList() }

const resetForm = () => { Object.assign(form, { id: null, username: '', password: '', nickname: '', email: '', phone: '', sex: 0, status: 1, remark: '' }) }

const handleAdd = () => { resetForm(); dialog.title = '新增管理员'; dialog.type = 'add'; dialog.visible = true }
const handleEdit = (row) => { resetForm(); Object.assign(form, row); dialog.title = '编辑管理员'; dialog.type = 'edit'; dialog.visible = true }

const handleDelete = async (row) => {
  try {
    await ElMessageBox.confirm(`确认删除「${row.username}」？`, '提示', { type: 'warning' })
    await deleteSysUser(row.id)
    ElMessage.success('删除成功')
    loadList()
  } catch (e) {}
}

const handleStatusChange = async (row) => {
  try { await changeSysUserStatus(row.id, row.status); ElMessage.success('状态修改成功') } catch (e) { row.status = row.status === 1 ? 0 : 1 }
}

const handleResetPwd = (row) => { pwdDialog.id = row.id; pwdDialog.password = ''; pwdDialog.visible = true }

const submitResetPwd = async () => {
  if (!pwdDialog.password) { ElMessage.warning('请输入新密码'); return }
  try {
    await resetSysUserPwd(pwdDialog.id, pwdDialog.password)
    ElMessage.success('重置成功')
    pwdDialog.visible = false
  } catch (e) {}
}

const submitForm = async () => {
  try {
    await formRef.value.validate()
    if (dialog.type === 'add') { await addSysUser(form); ElMessage.success('新增成功') }
    else { await updateSysUser(form); ElMessage.success('修改成功') }
    dialog.visible = false
    loadList()
  } catch (e) {}
}

loadList()
</script>
