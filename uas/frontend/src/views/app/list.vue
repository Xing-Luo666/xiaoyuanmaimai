<template>
  <div>
    <el-card class="search-card" shadow="never">
      <el-form :inline="true" :model="query" class="search-form">
        <el-form-item label="应用名称">
          <el-input v-model="query.appName" placeholder="请输入应用名称" clearable @keyup.enter="loadList" />
        </el-form-item>
        <el-form-item label="类型">
          <el-select v-model="query.appType" placeholder="全部" clearable style="width:120px">
            <el-option label="Web应用" value="web" />
            <el-option label="移动应用" value="mobile" />
            <el-option label="桌面应用" value="desktop" />
          </el-select>
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
        <el-button type="primary" :icon="Plus" @click="handleAdd">新增应用</el-button>
      </div>
      <el-table v-loading="loading" :data="list" border>
        <el-table-column type="index" label="序号" width="60" />
        <el-table-column prop="appId" label="AppID" width="200" />
        <el-table-column prop="appName" label="应用名称" width="180" show-overflow-tooltip />
        <el-table-column prop="appType" label="类型" width="100">
          <template #default="{row}">
            <el-tag>{{ appTypeText(row.appType) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="sm4Secret" label="SM4秘钥" width="200" show-overflow-tooltip />
        <el-table-column prop="appSecret" label="AppSecret" width="200" show-overflow-tooltip />
        <el-table-column prop="redirectUri" label="回调地址" min-width="220" show-overflow-tooltip />
        <el-table-column label="状态" width="100">
          <template #default="{row}">
            <el-tag :type="row.status === 1 ? 'success' : 'danger'">
              {{ row.status === 1 ? '启用' : '禁用' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="createTime" label="创建时间" width="170" :formatter="formatTimeCol" />
        <el-table-column label="操作" width="220" fixed="right">
          <template #default="{row}">
            <el-button link type="primary" @click="handleView(row)">查看</el-button>
            <el-button link type="primary" @click="handleEdit(row)">编辑</el-button>
            <el-button link type="warning" @click="handleReset(row)">重置密钥</el-button>
            <el-button link type="danger" @click="handleDelete(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
      <div class="pagination-wrap">
        <el-pagination
          v-model:current-page="query.pageNum"
          v-model:page-size="query.pageSize"
          :total="total"
          layout="total, prev, pager, next, jumper"
          @current-change="loadList"
        />
      </div>
    </el-card>

    <!-- 表单 -->
    <el-dialog v-model="dialog.visible" :title="dialog.title" width="640px">
      <el-form ref="formRef" :model="form" :rules="rules" label-width="120px">
        <el-form-item label="应用名称" prop="appName">
          <el-input v-model="form.appName" :disabled="dialog.type === 'view'" />
        </el-form-item>
        <el-form-item label="应用类型">
          <el-select v-model="form.appType" :disabled="dialog.type === 'view'">
            <el-option label="Web应用" value="web" />
            <el-option label="移动应用" value="mobile" />
            <el-option label="桌面应用" value="desktop" />
          </el-select>
        </el-form-item>
        <el-form-item label="回调地址" prop="redirectUri">
          <el-input v-model="form.redirectUri" placeholder="如：http://localhost:8080/oauth/callback" :disabled="dialog.type === 'view'" />
        </el-form-item>
        <el-form-item label="状态">
          <el-radio-group v-model="form.status" :disabled="dialog.type === 'view'">
            <el-radio :value="1">启用</el-radio>
            <el-radio :value="0">禁用</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="应用描述">
          <el-input v-model="form.description" type="textarea" :rows="3" :disabled="dialog.type === 'view'" />
        </el-form-item>
        <template v-if="dialog.type === 'view'">
          <el-form-item label="AppID">
            <el-input :model-value="form.appId" readonly />
          </el-form-item>
          <el-form-item label="SM4秘钥">
            <el-input :model-value="form.sm4Secret" readonly />
          </el-form-item>
          <el-form-item label="AppSecret">
            <el-input :model-value="form.appSecret" readonly />
          </el-form-item>
        </template>
      </el-form>
      <template #footer>
        <el-button @click="dialog.visible = false">取消</el-button>
        <el-button type="primary" @click="submitForm" v-if="dialog.type !== 'view'">确定</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Search, Refresh, Plus } from '@element-plus/icons-vue'
import { listApp, addApp, updateApp, deleteApp, resetAppSecret, getApp } from '@/api/uas'
import { formatTimeCol } from '@/utils/format'

const loading = ref(false)
const list = ref([])
const total = ref(0)
const query = reactive({ pageNum: 1, pageSize: 10, appName: '', appType: '', status: '' })
const formRef = ref(null)
const dialog = reactive({ visible: false, title: '', type: 'add' })
const form = reactive({
  id: null, appName: '', appType: 'web', redirectUri: '', status: 1, description: '',
  appId: '', sm4Secret: '', appSecret: ''
})
const rules = {
  appName: [{ required: true, message: '请输入应用名称', trigger: 'blur' }],
  redirectUri: [{ required: true, message: '请输入回调地址', trigger: 'blur' }]
}

const appTypeText = (t) => ({ web: 'Web应用', mobile: '移动应用', desktop: '桌面应用' }[t] || t)

const loadList = async () => {
  loading.value = true
  try {
    const res = await listApp(query)
    list.value = res.rows || []
    total.value = res.total || 0
  } finally {
    loading.value = false
  }
}

const resetQuery = () => {
  query.appName = ''
  query.appType = ''
  query.status = ''
  query.pageNum = 1
  loadList()
}

const resetForm = () => {
  Object.assign(form, {
    id: null, appName: '', appType: 'web', redirectUri: '', status: 1, description: '',
    appId: '', sm4Secret: '', appSecret: ''
  })
}

const handleAdd = () => {
  resetForm()
  dialog.title = '新增应用'
  dialog.type = 'add'
  dialog.visible = true
}

const handleEdit = (row) => {
  resetForm()
  Object.assign(form, row)
  dialog.title = '编辑应用'
  dialog.type = 'edit'
  dialog.visible = true
}

const handleView = async (row) => {
  resetForm()
  const res = await getApp(row.id)
  Object.assign(form, res.data)
  dialog.title = '应用详情'
  dialog.type = 'view'
  dialog.visible = true
}

const handleDelete = async (row) => {
  try {
    await ElMessageBox.confirm(`确认删除应用「${row.appName}」？`, '提示', { type: 'warning' })
    await deleteApp(row.id)
    ElMessage.success('删除成功')
    loadList()
  } catch (e) {}
}

const handleReset = async (row) => {
  try {
    await ElMessageBox.confirm(`确认重置应用「${row.appName}」的密钥？重置后旧密钥立即失效！`, '安全提示', { type: 'warning' })
    const res = await resetAppSecret(row.id)
    ElMessageBox.alert(`新密钥：\nAppSecret: ${res.data.appSecret}\nSM4Secret: ${res.data.sm4Secret}`, '重置成功', { type: 'success' })
    loadList()
  } catch (e) {}
}

const submitForm = async () => {
  try {
    await formRef.value.validate()
    if (dialog.type === 'add') {
      const res = await addApp(form)
      ElMessageBox.alert(`应用创建成功，请妥善保存以下密钥：\n\nAppID: ${res.data.appId}\nAppSecret: ${res.data.appSecret}\nSM4Secret: ${res.data.sm4Secret}`, '创建成功', { type: 'success' })
    } else {
      await updateApp(form)
      ElMessage.success('修改成功')
    }
    dialog.visible = false
    loadList()
  } catch (e) {}
}

loadList()
</script>
