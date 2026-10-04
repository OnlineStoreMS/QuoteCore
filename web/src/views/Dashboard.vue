<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { Document, Ticket } from '@element-plus/icons-vue'
import { fetchDashboardStats, type DashboardStats } from '../api/quote'

const router = useRouter()
const stats = ref<DashboardStats>({
  quoteCount: 0,
  draftCount: 0,
  sentCount: 0,
  wonCount: 0,
  templateCount: 0,
  monthTotalAmt: 0,
  secondEditApplyCount: 0,
})
const loading = ref(false)

async function load() {
  loading.value = true
  try {
    stats.value = await fetchDashboardStats()
  } catch (e) {
    ElMessage.error((e as Error).message || '加载失败')
  } finally {
    loading.value = false
  }
}

onMounted(load)
</script>

<template>
  <div v-loading="loading" class="dashboard">
    <h2 class="page-title">报价中心</h2>
    <p class="desc">为客户选品或手填明细，生成可复制图片 / 下载 PDF 的专业报价单。</p>

    <div class="stat-row">
      <el-card shadow="never" class="stat"><div class="n">{{ stats.quoteCount }}</div><div class="l">全部报价</div></el-card>
      <el-card shadow="never" class="stat"><div class="n">{{ stats.draftCount }}</div><div class="l">草稿</div></el-card>
      <el-card shadow="never" class="stat"><div class="n">{{ stats.sentCount }}</div><div class="l">已发送</div></el-card>
      <el-card shadow="never" class="stat"><div class="n">{{ stats.wonCount }}</div><div class="l">已成交</div></el-card>
      <el-card shadow="never" class="stat"><div class="n">¥{{ stats.monthTotalAmt.toFixed(0) }}</div><div class="l">本月金额</div></el-card>
      <el-card shadow="never" class="stat" style="cursor:pointer" @click="router.push({ path: '/quotes', query: { secondEdit: 'pending' } })">
        <div class="n">{{ stats.secondEditApplyCount || 0 }}</div>
        <div class="l">二次编辑申请</div>
      </el-card>
    </div>

    <div class="card-grid">
      <el-card shadow="hover" class="action-card" @click="router.push('/quotes/new')">
        <el-icon :size="32" color="#409eff"><Document /></el-icon>
        <h3>新建报价单</h3>
        <p>选客户、选商品或手填</p>
      </el-card>
      <el-card shadow="hover" class="action-card" @click="router.push({ path: '/quotes', query: { fromBiz: '1' } })">
        <el-icon :size="32" color="#e6a23c"><Document /></el-icon>
        <h3>从业务模板创建</h3>
        <p>组车固定行，再填规格价格</p>
      </el-card>
      <el-card shadow="hover" class="action-card" @click="router.push('/quotes')">
        <el-icon :size="32" color="#67c23a"><Document /></el-icon>
        <h3>报价单列表</h3>
        <p>查询、复制、作废</p>
      </el-card>
      <el-card shadow="hover" class="action-card" @click="router.push('/templates/layout')">
        <el-icon :size="32" color="#e6a23c"><Ticket /></el-icon>
        <h3>版式模板</h3>
        <p>Logo、店名与显示项</p>
      </el-card>
      <el-card shadow="hover" class="action-card" @click="router.push('/templates/business')">
        <el-icon :size="32" color="#f56c6c"><Ticket /></el-icon>
        <h3>业务模板</h3>
        <p>组车产品/配件骨架</p>
      </el-card>
    </div>
  </div>
</template>

<style scoped>
.page-title { margin: 0 0 8px; font-size: 20px; }
.desc { color: #909399; margin: 0 0 20px; }
.stat-row { display: flex; gap: 12px; flex-wrap: wrap; margin-bottom: 20px; }
.stat { min-width: 120px; text-align: center; }
.stat .n { font-size: 28px; font-weight: 600; color: #303133; }
.stat .l { color: #909399; font-size: 13px; }
.card-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(220px, 1fr)); gap: 16px; }
.action-card { cursor: pointer; text-align: center; padding: 12px 0; }
.action-card h3 { margin: 12px 0 6px; font-size: 16px; }
.action-card p { margin: 0; color: #909399; font-size: 13px; }
</style>
