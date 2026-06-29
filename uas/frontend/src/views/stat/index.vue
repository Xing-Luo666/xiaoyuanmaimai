<template>
  <div>
    <el-row :gutter="16" class="mb16">
      <el-col :span="6">
        <el-card shadow="never">
          <div class="stat-card">
            <div class="icon" style="background:#ecf5ff;color:#409eff"><el-icon :size="28"><User /></el-icon></div>
            <div class="info"><p class="label">个体用户</p><p class="value">{{ overview.personalCount || 0 }}</p></div>
          </div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="never">
          <div class="stat-card">
            <div class="icon" style="background:#f0f9eb;color:#67c23a"><el-icon :size="28"><OfficeBuilding /></el-icon></div>
            <div class="info"><p class="label">企业用户</p><p class="value">{{ overview.corpCount || 0 }}</p></div>
          </div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="never">
          <div class="stat-card">
            <div class="icon" style="background:#fdf6ec;color:#e6a23c"><el-icon :size="28"><Connection /></el-icon></div>
            <div class="info"><p class="label">接入应用</p><p class="value">{{ overview.appCount || 0 }}</p></div>
          </div>
        </el-card>
      </el-col>
      <el-col :span="6">
        <el-card shadow="never">
          <div class="stat-card">
            <div class="icon" style="background:#fef0f0;color:#f56c6c"><el-icon :size="28"><TrendCharts /></el-icon></div>
            <div class="info"><p class="label">今日授权次数</p><p class="value">{{ overview.todayGrantCount || 0 }}</p></div>
          </div>
        </el-card>
      </el-col>
    </el-row>

    <el-row :gutter="16" class="mb16">
      <el-col :span="16">
        <el-card shadow="never">
          <template #header><div class="card-header"><span>近7天授权趋势</span></div></template>
          <div ref="trendChart" style="height:300px"></div>
        </el-card>
      </el-col>
      <el-col :span="8">
        <el-card shadow="never">
          <template #header><div class="card-header"><span>应用类型分布</span></div></template>
          <div ref="pieChart" style="height:300px"></div>
        </el-card>
      </el-col>
    </el-row>

    <el-row :gutter="16">
      <el-col :span="12">
        <el-card shadow="never">
          <template #header><div class="card-header"><span>授权量Top10应用</span></div></template>
          <el-table :data="topApps" border size="small">
            <el-table-column type="index" label="排名" width="60" />
            <el-table-column prop="appName" label="应用名称" show-overflow-tooltip />
            <el-table-column prop="grantCount" label="授权次数" width="100" />
          </el-table>
        </el-card>
      </el-col>
      <el-col :span="12">
        <el-card shadow="never">
          <template #header><div class="card-header"><span>近7天活跃用户</span></div></template>
          <el-table :data="activeUsers" border size="small">
            <el-table-column type="index" label="排名" width="60" />
            <el-table-column prop="userName" label="用户" show-overflow-tooltip />
            <el-table-column prop="loginCount" label="登录次数" width="100" />
          </el-table>
        </el-card>
      </el-col>
    </el-row>
  </div>
</template>

<script setup>
import { ref, onMounted, nextTick } from 'vue'
import * as echarts from 'echarts'
import { User, OfficeBuilding, Connection, TrendCharts } from '@element-plus/icons-vue'
import { getStatOverview, getStatTrend, getStatAppType, getStatTopApps, getStatActiveUsers } from '@/api/stat'

const overview = ref({})
const topApps = ref([])
const activeUsers = ref([])
const trendChart = ref(null)
const pieChart = ref(null)

let trendChartInstance = null
let pieChartInstance = null

const loadOverview = async () => { try { const res = await getStatOverview(); overview.value = res.data || {} } catch (e) {} }

const loadTrend = async () => {
  try {
    const res = await getStatTrend()
    const data = res.data || []
    await nextTick()
    if (!trendChartInstance) trendChartInstance = echarts.init(trendChart.value)
    trendChartInstance.setOption({
      tooltip: { trigger: 'axis' },
      legend: { data: ['授权次数', '登录次数'] },
      grid: { left: '3%', right: '4%', bottom: '3%', containLabel: true },
      xAxis: { type: 'category', boundaryGap: false, data: data.map(d => d.date) },
      yAxis: { type: 'value' },
      series: [
        { name: '授权次数', type: 'line', smooth: true, areaStyle: {}, data: data.map(d => d.grantCount) },
        { name: '登录次数', type: 'line', smooth: true, areaStyle: {}, data: data.map(d => d.loginCount) }
      ]
    })
  } catch (e) {}
}

const loadPie = async () => {
  try {
    const res = await getStatAppType()
    const data = res.data || []
    await nextTick()
    if (!pieChartInstance) pieChartInstance = echarts.init(pieChart.value)
    pieChartInstance.setOption({
      tooltip: { trigger: 'item' },
      legend: { bottom: 0 },
      series: [{
        type: 'pie', radius: ['40%', '70%'], avoidLabelOverlap: false,
        label: { show: false, position: 'center' },
        emphasis: { label: { show: true, fontSize: 18, fontWeight: 'bold' } },
        labelLine: { show: false },
        data: data.map(d => ({ value: d.count, name: d.name }))
      }]
    })
  } catch (e) {}
}

const loadTopApps = async () => { try { const res = await getStatTopApps(); topApps.value = res.data || [] } catch (e) {} }
const loadActiveUsers = async () => { try { const res = await getStatActiveUsers(); activeUsers.value = res.data || [] } catch (e) {} }

onMounted(() => {
  loadOverview()
  loadTrend()
  loadPie()
  loadTopApps()
  loadActiveUsers()
  window.addEventListener('resize', () => {
    trendChartInstance?.resize()
    pieChartInstance?.resize()
  })
})
</script>

<style lang="scss" scoped>
.stat-card { display: flex; align-items: center; gap: 16px;
  .icon { width: 56px; height: 56px; border-radius: 8px; display: flex; align-items: center; justify-content: center; }
  .info { flex: 1;
    .label { color: #909399; font-size: 13px; margin-bottom: 4px; }
    .value { font-size: 24px; font-weight: 600; color: #303133; }
  }
}
.card-header { font-weight: 600; }
</style>
