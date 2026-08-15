<script setup lang="ts">
import { FullScreen } from '@element-plus/icons-vue'
import type { QuoteItem } from '../api/quote'
import { uploadImage } from '../api/upload'
import { ElMessage } from 'element-plus'

const props = withDefaults(
  defineProps<{
    items: QuoteItem[]
    large?: boolean
    showZoomBtn?: boolean
  }>(),
  { large: false, showZoomBtn: false },
)

const emit = defineEmits<{
  zoom: []
  previewImage: [url: string]
}>()

function sameProduct(a: QuoteItem, b: QuoteItem): boolean {
  if (a.productId && b.productId) return Number(a.productId) === Number(b.productId)
  return (a.name || '').trim() !== '' && (a.name || '').trim() === (b.name || '').trim()
}

function isProductHead(idx: number): boolean {
  if (idx <= 0) return true
  return !sameProduct(props.items[idx], props.items[idx - 1])
}

function renumberSort() {
  props.items.forEach((it, i) => {
    it.sort = (i + 1) * 10
  })
}

function emptyItem(partial?: Partial<QuoteItem>): QuoteItem {
  return {
    sort: (props.items.length + 1) * 10,
    source: 'manual',
    name: '',
    specLabel: '',
    imageUrl: '',
    qty: 1,
    unit: '件',
    retailPrice: 0,
    quotePrice: 0,
    upgradeNote: '',
    paramsText: '',
    remark: '',
    ...partial,
  }
}

function addSpecRow(idx: number) {
  const base = props.items[idx]
  if (!base) return
  props.items.splice(
    idx + 1,
    0,
    emptyItem({
      source: base.source || 'manual',
      productId: base.productId ?? null,
      skuId: null,
      name: base.name,
      imageUrl: '',
      unit: base.unit || '件',
      paramsText: base.paramsText || '',
    }),
  )
  renumberSort()
}

function onProductNameInput(idx: number, val: string) {
  const old = props.items[idx].name
  props.items[idx].name = val
  const head = props.items[idx]
  for (let i = idx + 1; i < props.items.length; i++) {
    const it = props.items[i]
    if (head.productId && it.productId && Number(it.productId) === Number(head.productId)) {
      it.name = val
      continue
    }
    if (!head.productId && (it.name || '') === (old || '')) {
      it.name = val
      continue
    }
    break
  }
}

function removeRow(idx: number) {
  props.items.splice(idx, 1)
  renumberSort()
}

function moveRow(idx: number, dir: -1 | 1) {
  const j = idx + dir
  if (j < 0 || j >= props.items.length) return
  const tmp = props.items[idx]
  props.items[idx] = props.items[j]
  props.items[j] = tmp
  renumberSort()
}

async function uploadRowImage(idx: number, opt: { file: File }) {
  try {
    props.items[idx].imageUrl = await uploadImage(opt.file, 'items')
  } catch (e) {
    ElMessage.error((e as Error).message)
  }
}

function openImage(url?: string) {
  const u = (url || '').trim()
  if (!u) return
  emit('previewImage', u)
}
</script>

<template>
  <div class="items-editor" :class="{ large }">
    <div v-if="showZoomBtn" class="editor-toolbar">
      <el-button type="primary" plain :icon="FullScreen" @click="emit('zoom')">放大编辑</el-button>
    </div>
    <el-table :data="items" border :size="large ? 'default' : 'small'" row-key="sort" class="items-table">
      <el-table-column label="产品" :min-width="large ? 220 : 160">
        <template #default="{ row, $index }">
          <el-input
            v-if="isProductHead($index)"
            :model-value="row.name"
            placeholder="产品名称"
            @update:model-value="(v: string) => onProductNameInput($index, v)"
          />
          <div v-else class="spec-cont">└ 同产品规格</div>
        </template>
      </el-table-column>
      <el-table-column label="规格" :min-width="large ? 200 : 130">
        <template #default="{ row }"><el-input v-model="row.specLabel" placeholder="规格" type="textarea" :rows="large ? 2 : 1" /></template>
      </el-table-column>
      <el-table-column label="图" :width="large ? 140 : 112">
        <template #default="{ row, $index }">
          <div class="img-cell">
            <button
              v-if="row.imageUrl"
              type="button"
              class="thumb-btn"
              title="预览图片"
              @click="openImage(row.imageUrl)"
            >
              <img :src="row.imageUrl" alt="" class="thumb" />
            </button>
            <div class="img-actions">
              <el-upload :show-file-list="false" :http-request="(o: any) => uploadRowImage($index, o)" accept="image/*">
                <el-button link type="primary">上传</el-button>
              </el-upload>
              <el-popover placement="bottom" :width="320" trigger="click">
                <template #reference>
                  <el-button link type="primary">链接</el-button>
                </template>
                <div class="img-url-box">
                  <el-input
                    v-model="row.imageUrl"
                    type="textarea"
                    :rows="2"
                    clearable
                    placeholder="粘贴图片链接，如 https://…"
                  />
                  <div class="img-url-tip">支持直接粘贴外链 URL</div>
                </div>
              </el-popover>
            </div>
          </div>
        </template>
      </el-table-column>
      <el-table-column label="数量" :width="large ? 120 : 90">
        <template #default="{ row }">
          <el-input-number v-model="row.qty" :min="0.01" :step="1" controls-position="right" style="width:100%" />
        </template>
      </el-table-column>
      <el-table-column label="零售价" :width="large ? 130 : 110">
        <template #default="{ row }">
          <el-input v-model.number="row.retailPrice" inputmode="decimal" placeholder="0.00" />
        </template>
      </el-table-column>
      <el-table-column label="报价" :width="large ? 130 : 110">
        <template #default="{ row }">
          <el-input v-model.number="row.quotePrice" inputmode="decimal" placeholder="0.00" />
        </template>
      </el-table-column>
      <el-table-column label="小计" :width="large ? 110 : 90" align="right">
        <template #default="{ row }">{{ (Number(row.qty || 0) * Number(row.quotePrice || 0)).toFixed(2) }}</template>
      </el-table-column>
      <el-table-column label="升级款" :min-width="large ? 160 : 120">
        <template #default="{ row }"><el-input v-model="row.upgradeNote" /></template>
      </el-table-column>
      <el-table-column label="参数" :min-width="large ? 180 : 140">
        <template #default="{ row }"><el-input v-model="row.paramsText" type="textarea" :rows="large ? 2 : 1" /></template>
      </el-table-column>
      <el-table-column label="备注" :min-width="large ? 160 : 120">
        <template #default="{ row }"><el-input v-model="row.remark" type="textarea" :rows="large ? 2 : 1" /></template>
      </el-table-column>
      <el-table-column label="操作" :width="large ? 200 : 168" fixed="right">
        <template #default="{ $index }">
          <el-button link type="primary" @click="addSpecRow($index)">加规格</el-button>
          <el-button link @click="moveRow($index, -1)">上</el-button>
          <el-button link @click="moveRow($index, 1)">下</el-button>
          <el-button link type="danger" @click="removeRow($index)">删</el-button>
        </template>
      </el-table-column>
    </el-table>
  </div>
</template>

<style scoped>
.editor-toolbar { display: flex; justify-content: flex-end; margin-bottom: 8px; }
.items-editor.large .items-table { --el-font-size-base: 14px; }
.spec-cont { color: #8f959e; font-size: 12px; padding: 0 4px; }
.img-cell { display: flex; flex-direction: column; align-items: center; gap: 2px; }
.img-actions { display: flex; gap: 2px; flex-wrap: wrap; justify-content: center; }
.img-url-box { display: flex; flex-direction: column; gap: 6px; }
.img-url-tip { font-size: 12px; color: #8f959e; }
.thumb-btn {
  padding: 0;
  border: 0;
  background: transparent;
  cursor: zoom-in;
  line-height: 0;
  border-radius: 4px;
  overflow: hidden;
  width: 36px;
  height: 36px;
}
.large .thumb-btn { width: 56px; height: 56px; }
.thumb { width: 100%; height: 100%; object-fit: cover; display: block; }
</style>
