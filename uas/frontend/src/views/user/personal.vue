<template>
  <div>
    <el-card class="search-card" shadow="never">
      <el-form :inline="true" :model="query" class="search-form">
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
        <el-button type="danger" :icon="Delete" :disabled="!multipleSelection.length" @click="batchDelete">批量删除</el-button>
      </div>
      <el-table v-loading="loading" :data="list" border @selection-change="handleSelectionChange">
        <el-table-column type="selection" width="50" />
        <el-table-column type="index" label="序号" width="60" />
        <el-table-column prop="phone" label="手机号" width="130" />
        <el-table-column prop="realName" label="姓名" width="100" />
        <el-table-column prop="idCardNo" label="证件号" width="180" />
        <el-table-column prop="authLevel" label="认证等级" width="100">
          <template #default="{row}">
            <el-tag :type="levelTag(row.authLevel)">{{ row.authLevel }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="nickname" label="昵称" width="120" show-overflow-tooltip />
        <el-table-column prop="email" label="邮箱" width="180" show-overflow-tooltip />
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
            <el-button link type="primary" @click="handleView(row)">查看</el-button>
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

    <!-- 表单对话框 -->
    <el-dialog v-model="dialog.visible" :title="dialog.title" width="600px">
      <el-form ref="formRef" :model="form" :rules="rules" label-width="100px">
        <el-form-item label="手机号" prop="phone">
          <el-input v-model="form.phone" placeholder="请输入手机号" :disabled="dialog.type === 'view'" />
        </el-form-item>
        <el-form-item label="密码" prop="password" v-if="dialog.type === 'add'">
          <el-input v-model="form.password" placeholder="留空默认为 123456" />
        </el-form-item>
        <el-form-item label="姓名" prop="realName">
          <el-input v-model="form.realName" placeholder="请输入姓名" :disabled="dialog.type === 'view'" />
        </el-form-item>
        <el-form-item label="证件类型">
          <el-select v-model="form.idCardType" :disabled="dialog.type === 'view'">
            <el-option label="身份证" :value="1" />
            <el-option label="护照" :value="2" />
            <el-option label="军官证" :value="3" />
          </el-select>
        </el-form-item>
        <el-form-item label="证件号">
          <el-input v-model="form.idCardNo" placeholder="请输入证件号" :disabled="dialog.type === 'view'" />
        </el-form-item>
        <el-form-item label="认证等级">
          <el-select v-model="form.authLevel" :disabled="dialog.type === 'view'">
            <el-option label="L1 基础认证" value="L1" />
            <el-option label="L2 实名认证" value="L2" />
            <el-option label="L3 人脸认证" value="L3" />
          </el-select>
        </el-form-item>
        <el-form-item label="昵称">
          <el-input v-model="form.nickname" :disabled="dialog.type === 'view'" />
        </el-form-item>
        <el-form-item label="邮箱">
          <el-input v-model="form.email" :disabled="dialog.type === 'view'" />
        </el-form-item>
        <el-form-item label="状态" v-if="dialog.type !== 'add'">
          <el-radio-group v-model="form.status" :disabled="dialog.type === 'view'">
            <el-radio :value="1">启用</el-radio>
            <el-radio :value="0">禁用</el-radio>
          </el-radio-group>
        </el-form-item>
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
import { Search, Refresh, Plus, Delete } from '@element-plus/icons-vue'
import { listPersonalUser, addPersonalUser, updatePersonalUser, deletePersonalUser, changePersonalUserStatus, getPersonalUser } from '@/api/uas'
import { formatTimeCol } from '@/utils/format'

const loading = ref(false)
const list = ref([])
const total = ref(0)
const query = reactive({ pageNum: 1, pageSize: 10, phone: '', status: '' })
const multipleSelection = ref([])
const formRef = ref(null)

const dialog = reactive({ visible: false, title: '', type: 'add' })
const form = reactive({
  id: null, phone: '', password: '', realName: '', idCardType: 1,
  idCardNo: '', authLevel: 'L1', nickname: '', email: '', status: 1
})
const rules = {
  phone: [{ required: true, message: '请输入手机号', trigger: 'blur' }],
  realName: [{ required: true, message: '请输入姓名', trigger: 'blur' }]
}

const levelTag = (l) => ({ L1: 'info', L2: 'warning', L3: 'success' }[l] || 'info')
const auditText = (s) => ({ 0: '未提交', 1: '待审核', 2: '已通过', 3: '已驳回' }[s] || '未提交')
const auditTag = (s) => ({ 0: 'info', 1: 'warning', 2: 'success', 3: 'danger' }[s] || 'info')

const loadList = async () => {
  loading.value = true
  try {
    const res = await listPersonalUser(query)
    list.value = res.rows || []
    total.value = res.total || 0
  } finally {
    loading.value = false
  }
}

const resetQuery = () => {
  query.phone = ''
  query.status = ''
  query.pageNum = 1
  loadList()
}

const handleSelectionChange = (val) => { multipleSelection.value = val }

const resetForm = () => {
  Object.assign(form, {
    id: null, phone: '', password: '', realName: '', idCardType: 1,
    idCardNo: '', authLevel: 'L1', nickname: '', email: '', status: 1
  })
}

const handleAdd = () => {
  resetForm()
  dialog.title = '新增个体用户'
  dialog.type = 'add'
  dialog.visible = true
}

const handleEdit = (row) => {
  resetForm()
  Object.assign(form, row)
  dialog.title = '编辑个体用户'
  dialog.type = 'edit'
  dialog.visible = true
}

const handleView = async (row) => {
  resetForm()
  const res = await getPersonalUser(row.id)
  Object.assign(form, res.data)
  dialog.title = '查看个体用户'
  dialog.type = 'view'
  dialog.visible = true
}

const handleDelete = async (row) => {
  try {
    await ElMessageBox.confirm(`确认删除用户「${row.phone}」？`, '提示', { type: 'warning' })
    await deletePersonalUser(row.id)
    ElMessage.success('删除成功')
    loadList()
  } catch (e) {}
}

const batchDelete = async () => {
  try {
    await ElMessageBox.confirm(`确认删除选中的 ${multipleSelection.value.length} 条数据？`, '提示', { type: 'warning' })
    for (const item of multipleSelection.value) {
      await deletePersonalUser(item.id)
    }
    ElMessage.success('批量删除成功')
    loadList()
  } catch (e) {}
}

const handleStatusChange = async (row) => {
  try {
    await changePersonalUserStatus(row.id, row.status)
    ElMessage.success('状态修改成功')
  } catch (e) {
    row.status = row.status === 1 ? 0 : 1
  }
}

const submitForm = async () => {
  try {
    await formRef.value.validate()
    if (dialog.type === 'add') {
      await addPersonalUser(form)
      ElMessage.success('新增成功')
    } else {
      await updatePersonalUser(form)
      ElMessage.success('修改成功')
    }
    dialog.visible = false
    loadList()
  } catch (e) {}
}

loadList()
</script>
