<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useRoute } from 'vue-router'
import { ElMessage } from 'element-plus'
import { fetchShareQuote, submitSupplyPrices, type ShareItem } from '../api/publicShare'

const route = useRoute()
const token = computed(() => String(route.params.token || '').trim())

const loading = ref(true)
const submitting = ref(false)
const submitted = ref(false)
const title = ref('')
const quoteNo = ref('')
const remark = ref('')
const previewUrl = ref('')
const showPreview = ref(false)

type Row = ShareItem & { draftPrice: number | null; draftRemark: string }

const rows = reactive<Row[]>([])

const displaySpec = (it: ShareItem) => (it.specLabel || '').trim() || '未填写规格'

function openPreview(url?: string) {
  const u = (url || '').trim()
  if (!u) return
  previewUrl.value = u
  showPreview.value = true
}

async function load() {
  if (!token.value) {
    ElMessage.error('链接无效')
    loading.value = false
    return
  }
  loading.value = true
  try {
    const q = await fetchShareQuote(token.value)
    title.value = q.title || '报价单'
    quoteNo.value = q.quoteNo || ''
    remark.value = q.remark || ''
    rows.splice(0, rows.length)
    for (const it of q.items || []) {
      rows.push({
        ...it,
        draftPrice: it.supplyPrice > 0 ? it.supplyPrice : null,
        draftRemark: it.supplyRemark || '',
      })
    }
  } catch (e) {
    ElMessage.error((e as Error).message || '加载失败')
  } finally {
    loading.value = false
  }
}

async function onSubmit() {
  const payload = rows
    .filter((r) => r.id > 0 && r.draftPrice != null && Number.isFinite(Number(r.draftPrice)))
    .map((r) => ({
      id: r.id,
      supplyPrice: Math.round(Number(r.draftPrice) * 100) / 100,
      supplyRemark: (r.draftRemark || '').trim(),
    }))
  if (!payload.length) {
    ElMessage.warning('请至少填写一行拿货价')
    return
  }
  for (const p of payload) {
    if (p.supplyPrice < 0) {
      ElMessage.warning('拿货价不能为负')
      return
    }
  }
  submitting.value = true
  try {
    await submitSupplyPrices(token.value, payload)
    submitted.value = true
    ElMessage.success('已提交，感谢填写')
    await load()
  } catch (e) {
    ElMessage.error((e as Error).message || '提交失败')
  } finally {
    submitting.value = false
  }
}

onMounted(load)
</script>

<template>
  <div class="share-page" v-loading="loading">
    <header class="hero">
      <div class="brand">报价中心</div>
      <h1>{{ title || '填写拿货价' }}</h1>
      <p v-if="quoteNo" class="meta">单号 {{ quoteNo }}</p>
      <p v-if="remark" class="remark">{{ remark }}</p>
    </header>

    <main class="list">
      <article v-for="row in rows" :key="row.id" class="card">
        <button
          v-if="row.imageUrl"
          type="button"
          class="thumb"
          @click="openPreview(row.imageUrl)"
        >
          <img :src="row.imageUrl" alt="" />
        </button>
        <div v-else class="thumb empty">无图</div>
        <div class="body">
          <div class="name">{{ displaySpec(row) }}</div>
          <label class="price-label">
            <span>拿货价（元）</span>
            <input
              v-model.number="row.draftPrice"
              type="number"
              inputmode="decimal"
              min="0"
              step="0.01"
              placeholder="请输入"
            />
          </label>
          <label class="price-label">
            <span>备注</span>
            <textarea v-model="row.draftRemark" rows="2" placeholder="选填，如交期、说明等" />
          </label>
        </div>
      </article>

      <div v-if="!loading && !rows.length" class="empty-hint">暂无规格明细</div>
    </main>

    <footer class="bar">
      <button type="button" class="submit" :disabled="submitting || !rows.length" @click="onSubmit">
        {{ submitting ? '提交中…' : submitted ? '再次提交' : '提交拿货价' }}
      </button>
    </footer>

    <teleport to="body">
      <div v-if="showPreview" class="lightbox" @click="showPreview = false">
        <img :src="previewUrl" alt="" @click.stop />
      </div>
    </teleport>
  </div>
</template>

<style scoped>
.share-page {
  min-height: 100vh;
  background: linear-gradient(180deg, #e8f0ea 0%, #f6f7f8 28%, #f6f7f8 100%);
  padding: 20px 16px 96px;
  box-sizing: border-box;
  font-family: "PingFang SC", "Hiragino Sans GB", "Noto Sans SC", sans-serif;
  color: #1f2a24;
}
.hero {
  margin-bottom: 16px;
}
.brand {
  font-size: 13px;
  letter-spacing: 0.08em;
  color: #3d6b4f;
  font-weight: 600;
  margin-bottom: 6px;
}
h1 {
  margin: 0;
  font-size: 22px;
  line-height: 1.35;
  font-weight: 700;
}
.meta {
  margin: 6px 0 0;
  font-size: 13px;
  color: #6b7280;
}
.remark {
  margin: 8px 0 0;
  font-size: 13px;
  color: #4b5563;
  white-space: pre-wrap;
}
.list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.card {
  display: grid;
  grid-template-columns: 88px 1fr;
  gap: 12px;
  background: #fff;
  border-radius: 12px;
  padding: 12px;
  box-shadow: 0 1px 2px rgba(16, 24, 40, 0.06);
}
.thumb {
  width: 88px;
  height: 88px;
  border: 0;
  padding: 0;
  border-radius: 8px;
  overflow: hidden;
  background: #eef2ef;
  cursor: zoom-in;
}
.thumb img {
  width: 100%;
  height: 100%;
  object-fit: cover;
  display: block;
}
.thumb.empty {
  display: flex;
  align-items: center;
  justify-content: center;
  color: #9ca3af;
  font-size: 12px;
  cursor: default;
}
.name {
  font-size: 15px;
  font-weight: 600;
  line-height: 1.4;
}
.spec {
  margin-top: 4px;
  font-size: 13px;
  color: #6b7280;
  white-space: pre-wrap;
}
.price-label {
  display: flex;
  flex-direction: column;
  gap: 6px;
  margin-top: 10px;
  font-size: 12px;
  color: #4b5563;
}
.price-label input {
  height: 40px;
  border: 1px solid #d1d5db;
  border-radius: 8px;
  padding: 0 12px;
  font-size: 16px;
  outline: none;
  background: #fafafa;
}
.price-label textarea {
  border: 1px solid #d1d5db;
  border-radius: 8px;
  padding: 8px 12px;
  font-size: 14px;
  outline: none;
  background: #fafafa;
  resize: vertical;
  font-family: inherit;
  line-height: 1.4;
}
.price-label input:focus,
.price-label textarea:focus {
  border-color: #3d6b4f;
  background: #fff;
}
.empty-hint {
  text-align: center;
  color: #9ca3af;
  padding: 40px 0;
}
.bar {
  position: fixed;
  left: 0;
  right: 0;
  bottom: 0;
  padding: 12px 16px calc(12px + env(safe-area-inset-bottom));
  background: rgba(246, 247, 248, 0.92);
  backdrop-filter: blur(8px);
  border-top: 1px solid #e5e7eb;
}
.submit {
  width: 100%;
  height: 46px;
  border: 0;
  border-radius: 10px;
  background: #2f5d43;
  color: #fff;
  font-size: 16px;
  font-weight: 600;
  cursor: pointer;
}
.submit:disabled {
  opacity: 0.55;
  cursor: not-allowed;
}
.lightbox {
  position: fixed;
  inset: 0;
  z-index: 4000;
  background: rgba(0, 0, 0, 0.88);
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 16px;
}
.lightbox img {
  max-width: 100%;
  max-height: 100%;
  object-fit: contain;
}
</style>
