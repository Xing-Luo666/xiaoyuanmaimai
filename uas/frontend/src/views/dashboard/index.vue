<template>
  <div class="dashboard">
    <el-row :gutter="16" class="stat-row">
      <el-col :xs="12" :sm="12" :md="6" v-for="(card, idx) in cards" :key="idx">
        <el-card shadow="hover" class="stat-card">
          <div class="stat-content">
            <div class="stat-icon" :style="{ background: card.bg }">
              <el-icon :size="28"><component :is="card.icon" /></el-icon>
            </div>
            <div class="stat-info">
              <div class="stat-label">{{ card.label }}</div>
              <div class="stat-value">{{ card.value }}</div>
            </div>
          </div>
        </el-card>
      </el-col>
    </el-row>

    <el-row :gutter="16" class="mt20">
      <el-col :xs="24" :md="16">
        <el-card shadow="never">
          <template #header>
            <div class="card-header">
              <span>近7天登录统计</span>
              <el-tag size="small" type="info">登录趋势</el-tag>
            </div>
          </template>
          <div ref="loginChartRef" class="chart"></div>
        </el-card>
      </el-col>
      <el-col :xs="24" :md="8">
        <el-card shadow="never">
          <template #header>
            <div class="card-header">
              <span>账户认证等级分布</span>
            </div>
          </template>
          <div ref="levelChartRef" class="chart"></div>
        </el-card>
      </el-col>
    </el-row>

    <el-row :gutter="16" class="mt20">
      <el-col :span="24">
        <el-card shadow="never">
          <template #header>
            <div class="card-header">
              <span>应用授权概览</span>
              <el-tag size="small" type="info">最近授权</el-tag>
            </div>
          </template>
          <el-empty v-if="!grants.length" description="暂无授权记录" :image-size="60" />
          <el-table v-else :data="grants" border>
            <el-table-column prop="appName" label="应用" min-width="160" />
            <el-table-column prop="userType" label="用户类型" width="100">
              <template #default="{row}">{{ row.userType === 'personal' ? '个体用户' : '企业用户' }}</template>
            </el-table-column>
            <el-table-column prop="grantTime" label="授权时间" width="180" />
          </el-table>
        </el-card>
      </el-col>
    </el-row>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted, nextTick } from 'vue'
import * as echarts from 'echarts'
import { statAccount, statLogin } from '@/api/stat'
import { listGrant } from '@/api/uas'

const cards = reactive([
  { label: '个体用户', value: 0, icon: 'UserFilled', bg: 'linear-gradient(135deg,#667eea,#764ba2)' },
  { label: '企业用户', value: 0, icon: 'OfficeBuilding', bg: 'linear-gradient(135deg,#f093fb,#f5576c)' },
  { label: '接入应用', value: 0, icon: 'Connection', bg: 'linear-gradient(135deg,#4facfe,#00f2fe)' },
  { label: '待审核', value: 0, icon: 'Bell', bg: 'linear-gradient(135deg,#fa709a,#fee140)' }
])

const loginChartRef = ref(null)
const levelChartRef = ref(null)
const grants = ref([])
let loginChart, levelChart

const loadStat = async () => {
  try {
    const res = await statAccount()
    const d = res.data
    cards[0].value = d.personalTotal || 0
    cards[1].value = d.corpTotal || 0
    cards[2].value = d.appActive || 0
    cards[3].value = (d.personalAudit || 0) + (d.corpAudit || 0)

    // 等级分布饼图
    nextTick(() => {
      if (levelChartRef.value) {
        levelChart = echarts.init(levelChartRef.value)
        levelChart.setOption({
          tooltip: { trigger: 'item' },
          legend: { bottom: 0 },
          series: [{
            type: 'pie',
            radius: ['40%', '70%'],
            data: [
              { value: d.personalL1 || 0, name: 'L1 基础认证' },
              { value: d.personalL2 || 0, name: 'L2 实名认证' },
              { value: d.personalL3 || 0, name: 'L3 人脸认证' }
            ]
          }]
        })
      }
    })
  } catch (e) {}
}

const loadLoginStat = async () => {
  try {
    const res = await statLogin()
    const data = res.data || []
    nextTick(() => {
      if (loginChartRef.value) {
        loginChart = echarts.init(loginChartRef.value)
        loginChart.setOption({
          tooltip: { trigger: 'axis' },
          legend: { data: ['成功', '失败'] },
          grid: { left: 40, right: 20, top: 40, bottom: 40 },
          xAxis: { type: 'category', data: data.map(d => d.date.slice(5)) },
          yAxis: { type: 'value' },
          series: [
            { name: '成功', type: 'line', smooth: true, areaStyle: {}, data: data.map(d => d.ok) },
            { name: '失败', type: 'line', smooth: true, areaStyle: {}, data: data.map(d => d.fail) }
          ]
        })
      }
    })
  } catch (e) {}
}

const loadGrants = async () => {
  try {
    const res = await listGrant({ pageNum: 1, pageSize: 5 })
    grants.value = res.rows || []
  } catch (e) {}
}

onMounted(() => {
  loadStat()
  loadLoginStat()
  loadGrants()
  window.addEventListener('resize', () => {
    loginChart?.resize()
    levelChart?.resize()
  })
})
</script>

<style lang="scss" scoped>
.dashboard { padding-bottom: 20px; }

.stat-row { margin-bottom: 0; }
.stat-card {
  .stat-content {
    display: flex;
    align-items: center;
    gap: 16px;
  }
  .stat-icon {
    width: 60px;
    height: 60px;
    border-radius: 8px;
    display: flex;
    align-items: center;
    justify-content: center;
    color: #fff;
  }
  .stat-info {
    flex: 1;
  }
  .stat-label { color: #909399; font-size: 13px; margin-bottom: 4px; }
  .stat-value { font-size: 26px; font-weight: 600; color: #303133; }
}

.card-header { display: flex; justify-content: space-between; align-items: center; }
.chart { height: 300px; }
.notice-content { color: #606266; font-size: 13px; line-height: 1.6; margin-top: 6px; }
</style>
