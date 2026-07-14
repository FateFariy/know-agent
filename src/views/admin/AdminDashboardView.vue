<template>
  <section class="dashboard-page">
    <div class="hero-card">
      <div class="hero-copy">
        <h3>把文档接入、切块策略和索引构建串成一条可观察的业务流水线</h3>
        <p class="section-description">
          后台管理台聚焦在文档进入系统后的关键节点：上传、推荐策略、策略确认、索引构建和对话观测。
        </p>
        <button class="primary-link" type="button" @click="goDocuments">前往文档接入</button>
      </div>
    </div>

    <div class="metrics-grid">
      <article class="metric-card">
        <span>文档总数</span>
        <strong>{{ formatCount(summary.total) }}</strong>
        <p>已进入管理台的文档记录</p>
      </article>
      <article class="metric-card">
        <span>解析成功</span>
        <strong>{{ formatCount(summary.parseSuccess) }}</strong>
        <p>可进入策略确认阶段的文档</p>
      </article>
      <article class="metric-card">
        <span>策略已确认</span>
        <strong>{{ formatCount(summary.strategyConfirmed) }}</strong>
        <p>已经形成最终切块链路</p>
      </article>
      <article class="metric-card">
        <span>索引完成</span>
        <strong>{{ formatCount(summary.indexSuccess) }}</strong>
        <p>可直接参与 RAG 检索问答</p>
      </article>
    </div>

    <div class="dashboard-grid">
      <article class="panel-card">
        <div class="panel-header">
          <div>
            <h4>建议演示路径</h4>
          </div>
        </div>

        <ol class="flow-list">
          <li>
            <strong>上传文档</strong>
            <span>通过假登录后的管理台上传 PDF / Word / Markdown 文档。</span>
          </li>
          <li>
            <strong>查看系统推荐策略</strong>
            <span>根据文档结构与内容长度，观察结构切块、递归分块、语义分块和智能切块的组合。</span>
          </li>
          <li>
            <strong>确认并构建索引</strong>
            <span>在推荐结果基础上补充或移除策略，再触发异步构建索引。</span>
          </li>
          <li>
            <strong>做对话观测</strong>
            <span>查看真实会话在当前文档问答与开放式提问两种模式下的执行轨迹。</span>
          </li>
        </ol>
      </article>

      <article class="panel-card">
        <div class="panel-header">
          <div>
            <h4>最近接入文档</h4>
          </div>
          <button class="ghost-link" type="button" @click="loadDashboard">刷新</button>
        </div>

        <div v-if="loading" class="empty-block">正在加载后台概览...</div>
        <div v-else-if="!documents.length" class="empty-block">当前还没有文档，先去“文档接入”页面上传一份资料。</div>

        <div v-else class="recent-list">
          <article v-for="item in documents.slice(0, 6)" :key="item.documentId" class="recent-item">
            <div class="recent-item-main">
              <strong>{{ item.documentName }}</strong>
              <p>{{ item.originalFileName }}</p>
            </div>
            <div class="recent-item-meta">
              <AdminStatusBadge :label="item.parseStatusName" :code="item.parseStatus" type="parse" />
              <AdminStatusBadge :label="item.indexStatusName" :code="item.indexStatus" type="index" />
            </div>
          </article>
        </div>
      </article>
    </div>
  </section>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { manageApi } from '../../api/api'
import AdminStatusBadge from '../../components/admin/AdminStatusBadge.vue'
import { formatCount, hasCode } from '../../utils/manageFormat'

const router = useRouter()
const loading = ref(false)
const documents = ref([])
const summary = reactive({
  total: 0,
  parseSuccess: 0,
  strategyConfirmed: 0,
  indexSuccess: 0
})

async function loadDashboard() {
  loading.value = true

  try {
    const data = await manageApi.queryDocumentPage({
      pageNo: 1,
      pageSize: 50,
      keyword: ''
    })
    documents.value = Array.isArray(data?.records) ? data.records : []

    summary.total = Number(data?.total || documents.value.length || 0)
    summary.parseSuccess = documents.value.filter((item) => hasCode(item.parseStatus, 3)).length
    summary.strategyConfirmed = documents.value.filter((item) => hasCode(item.strategyStatus, 3)).length
    summary.indexSuccess = documents.value.filter((item) => hasCode(item.indexStatus, 3)).length
  } catch (error) {
    console.error('加载后台概览失败', error)
    documents.value = []
    summary.total = 0
    summary.parseSuccess = 0
    summary.strategyConfirmed = 0
    summary.indexSuccess = 0
  } finally {
    loading.value = false
  }
}

function goDocuments() {
  router.push('/admin/documents')
}

onMounted(loadDashboard)
</script>

<style scoped>
.dashboard-page {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.hero-card,
.metric-card,
.panel-card {
  background: #fff;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg, 12px);
  box-shadow: var(--shadow-sm);
}

.hero-card {
  padding: 28px 30px;
}

.hero-copy {
  max-width: 860px;
}

.hero-card h3 {
  margin: 0;
  font-size: 18px;
  font-weight: 600;
  color: var(--color-text-strong);
}

.section-description {
  margin: 12px 0 16px;
  color: var(--color-muted);
  line-height: 1.8;
}

.primary-link,
.ghost-link {
  border: none;
  border-radius: var(--radius-sm, 6px);
  padding: 8px 16px;
  font-weight: 600;
  cursor: pointer;
  transition: opacity 0.2s ease;
}

.primary-link {
  color: #fff;
  background: var(--color-primary);
}

.ghost-link {
  color: var(--color-text);
  background: #fff;
  border: 1px solid var(--color-border);
}

.metrics-grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 16px;
}

.metric-card {
  padding: 22px;
}

.metric-card span {
  font-size: 13px;
  color: var(--color-muted);
}

.metric-card strong {
  display: block;
  margin-top: 12px;
  font-size: 28px;
  color: var(--color-text-strong);
}

.metric-card p {
  margin: 12px 0 0;
  color: var(--color-muted);
  line-height: 1.7;
}

.dashboard-grid {
  display: grid;
  grid-template-columns: 1.05fr 0.95fr;
  gap: 16px;
}

.panel-card {
  padding: 24px 26px;
}

.panel-header {
  display: flex;
  justify-content: space-between;
  gap: 12px;
  align-items: center;
  margin-bottom: 18px;
}

.panel-header h4 {
  margin: 0;
  font-size: 16px;
  font-weight: 600;
  color: var(--color-text-strong);
}

.flow-list {
  margin: 0;
  padding-left: 20px;
  color: var(--color-muted-strong);
  display: flex;
  flex-direction: column;
  gap: 16px;
  line-height: 1.7;
}

.flow-list strong {
  display: block;
  margin-bottom: 6px;
  color: var(--color-text-strong);
}

.recent-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.recent-item {
  display: flex;
  justify-content: space-between;
  gap: 16px;
  align-items: start;
  padding: 16px 18px;
  border-radius: var(--radius-md, 8px);
  background: var(--color-surface-soft);
}

.recent-item-main strong {
  display: block;
  color: var(--color-text-strong);
}

.recent-item-main p {
  margin: 8px 0 0;
  color: var(--color-muted);
  word-break: break-all;
}

.recent-item-meta {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
  justify-content: end;
}

.empty-block {
  min-height: 220px;
  display: grid;
  place-items: center;
  text-align: center;
  color: var(--color-muted);
  border-radius: var(--radius-md, 8px);
  border: 1px dashed var(--color-border);
}

@media (max-width: 1080px) {
  .metrics-grid,
  .dashboard-grid {
    grid-template-columns: 1fr 1fr;
  }
}

@media (max-width: 768px) {
  .panel-header,
  .recent-item {
    flex-direction: column;
    align-items: stretch;
  }

  .metrics-grid,
  .dashboard-grid {
    grid-template-columns: 1fr;
  }
}
</style>
