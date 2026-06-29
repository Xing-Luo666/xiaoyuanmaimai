<template>
  <div>
    <el-card class="search-card" shadow="never">
      <el-form :inline="true" :model="query" class="search-form">
        <el-form-item label="企业名称">
          <el-input v-model="query.corpName" placeholder="请输入企业名称" clearable @keyup.enter="loadList" />
        </el-form-item>
        <el-form-item label="信用代码">
          <el-input v-model="query.creditCode" placeholder="请输入信用代码" clearable @keyup.enter="loadList" />
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
        <el-table-column prop="username" label="账号" width="120" />
        <el-table-column prop="corpName" label="企业名称" width="200" show-overflow-tooltip />
        <el-table-column prop="corpType" label="类型" width="100">
          <template #default="{row}">
            {{ corpTypeText(row.corpType) }}
          </template>
        </el-table-column>
        <el-table-column prop="creditCode" label="信用代码" width="180" />
        <el-table-column prop="legalPerson" label="法定代表人" width="100" />
        <el-table-column prop="phone" label="联系电话" width="130" />
        <el-table-column label="状态" width="100">
          <template #default="{row}">
            <el-switch v-model="row.status" :active-value="1" :inactive-value="0" @change="handleStatusChange(row)" />
          </template>
        </el-table-column>
        <el-table-column label="审核状态" width="100">
          <template #default="{row}">
            <el-tag :type="auditTag(row.auditStatus)">{{ auditText(row.auditStatus) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="createTime" label="创建时间" width="170" :formatter="formatTimeCol" />
        <el-table-column label="操作" width="180" fixed="right">
          <template #default="{row}">
            <el-button link type="primary" @click="handleEdit(row)">编辑</el-button>
            <el-button link type="danger" @click="handleDelete(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
      <div class="pagination-wrap">
        <el-pagination
          v-model:current-page="query.pageNum"
          v-model:page-size="query.pageSize"
          :page-sizes="[10,20,50,100]"
          :total="total"
          layout="total, sizes, prev, pager, next, jumper"
          @size-change="loadList"
          @current-change="loadList"
        />
      </div>
    </el-card>

    <el-dialog v-model="dialog.visible" :title="dialog.title" width="640px">
      <el-form ref="formRef" :model="form" :rules="rules" label-width="120px">
        <el-row :gutter="16">
          <el-col :span="12">
            <el-form-item label="账号" prop="username">
              <el-input v-model="form.username" />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="密码" prop="password" v-if="dialog.type === 'add'">
              <el-input v-model="form.password" placeholder="留空默认123456" />
            </el-form-item>
          </el-col>
        </el-row>
        <el-row :gutter="16">
          <el-col :span="12">
            <el-form-item label="企业类型">
              <el-select v-model="form.corpType">
                <el-option label="企业" value="enterprise" />
                <el-option label="事业单位" value="institution" />
                <el-option label="政府机关" value="government" />
                <el-option label="社会团体" value="social" />
              </el-select>
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="企业名称" prop="corpName">
              <el-input v-model="form.corpName" />
            </el-form-item>
          </el-col>
        </el-row>
        <el-form-item label="统一社会信用代码" prop="creditCode">
          <el-input v-model="form.creditCode" />
        </el-form-item>
        <el-row :gutter="16">
          <el-col :span="12">
            <el-form-item label="法定代表人">
              <el-input v-model="form.legalPerson" />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="企业证件号">
              <el-input v-model="form.legalIdCard" />
            </el-form-item>
          </el-col>
        </el-row>
        <el-row :gutter="16">
          <el-col :span="12">
            <el-form-item label="经办人姓名">
              <el-input v-model="form.agentName" />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="经办人证件号">
              <el-input v-model="form.agentIdCard" />
            </el-form-item>
          </el-col>
        </el-row>
        <el-form-item label="联系电话">
          <el-input v-model="form.phone" />
        </el-form-item>
        <el-form-item label="状态" v-if="dialog.type !== 'add'">
          <el-radio-group v-model="form.status">
            <el-radio :value="1">启用</el-radio>
            <el-radio :value="0">禁用</el-radio>
          </el-radio-group>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialog.visible = false">取消</el-button>
        <el-button type="primary" @click="submitForm">确定</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Search, Refresh, Plus } from '@element-plus/icons-vue'
import { listCorpUser, addCorpUser, updateCorpUser, deleteCorpUser, changeCorpUserStatus } from '@/api/uas'
import { formatTimeCol } from '@/utils/format'

const loading = ref(false)
const list = ref([])
const total = ref(0)
const query = reactive({ pageNum: 1, pageSize: 10, corpName: '', creditCode: '', status: '' })
const formRef = ref(null)
const dialog = reactive({ visible: false, title: '', type: 'add' })
const form = reactive({
  id: null, username: '', password: '', corpType: 'enterprise', corpName: '',
  creditCode: '', legalPerson: '', legalIdCard: '', agentName: '', agentIdCard: '',
  phone: '', status: 1
})
const rules = {
  username: [{ required: true, message: '请输入账号', trigger: 'blur' }],
  corpName: [{ required: true, message: '请输入企业名称', trigger: 'blur' }],
  creditCode: [{ required: true, message: '请输入信用代码', trigger: 'blur' }]
}

const corpTypeText = (t) => ({ enterprise: '企业', institution: '事业单位', government: '政府机关', social: '社会团体' }[t] || t)
const auditText = (s) => ({ 0: '未提交', 1: '待审核', 2: '已通过', 3: '已驳回' }[s] || '未提交')
const auditTag = (s) => ({ 0: 'info', 1: 'warning', 2: 'success', 3: 'danger' }[s] || 'info')

const loadList = async () => {
  loading.value = true
  try {
    const res = await listCorpUser(query)
    list.value = res.rows || []
    total.value = res.total || 0
  } finally {
    loading.value = false
  }
}

const resetQuery = () => {
  query.corpName = ''
  query.creditCode = ''
  query.status = ''
  query.pageNum = 1
  loadList()
}

const resetForm = () => {
  Object.assign(form, {
    id: null, username: '', password: '', corpType: 'enterprise', corpName: '',
    creditCode: '', legalPerson: '', legalIdCard: '', agentName: '', agentIdCard: '',
    phone: '', status: 1
  })
}

const handleAdd = () => {
  resetForm()
  dialog.title = '新增企业用户'
  dialog.type = 'add'
  dialog.visible = true
}

const handleEdit = (row) => {
  resetForm()
  Object.assign(form, row)
  dialog.title = '编辑企业用户'
  dialog.type = 'edit'
  dialog.visible = true
}

const handleDelete = async (row) => {
  try {
    await ElMessageBox.confirm(`确认删除「${row.corpName}」？`, '提示', { type: 'warning' })
    await deleteCorpUser(row.id)
    ElMessage.success('删除成功')
    loadList()
  } catch (e) {}
}

const handleStatusChange = async (row) => {
  try {
    await changeCorpUserStatus(row.id, row.status)
    ElMessage.success('状态修改成功')
  } catch (e) {
    row.status = row.status === 1 ? 0 : 1
  }
}

const submitForm = async () => {
  try {
    await formRef.value.validate()
    if (dialog.type === 'add') {
      await addCorpUser(form)
      ElMessage.success('新增成功')
    } else {
      await updateCorpUser(form)
      ElMessage.success('修改成功')
    }
    dialog.visible = false
    loadList()
  } catch (e) {}
}

loadList()
</script>
