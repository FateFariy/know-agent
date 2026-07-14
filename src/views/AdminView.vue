<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useRouter } from 'vue-router'
import { useDocumentStore } from '@/stores/document'
import { knowledgeApi } from '@/api/knowledge'
import { ElMessage, ElTree, ElButton, ElInput, ElDialog, ElForm, ElFormItem, ElSelect, ElOption } from 'element-plus'
import {
  Files,
  Document,
  Folder,
  ArrowLeft,
  Plus,
  Edit,
  Delete,
  Search,
  Upload,
  View,
} from '@element-plus/icons-vue'
import DocumentList from '@/components/document/DocumentList.vue'
import type { KnowledgeScope, KnowledgeTopic, TopicDocumentRelation, UploadDocumentReq } from '@/types'

const router = useRouter()
const documentStore = useDocumentStore()

type MenuKey = 'document-upload' | 'document-list' | 'knowledge-scope' | 'knowledge-topic' | 'overview' | 'dialog-observation'

const activeMenu = ref<MenuKey>('document-upload')

interface NavItem {
  key: string
  label: string
  icon: typeof Files
  children?: { key: MenuKey; label: string; icon: typeof Plus }[]
}

const navItems: NavItem[] = [
  {
    key: 'overview',
    label: '运营总览',
    icon: Files,
  },
  {
    key: 'document-upload',
    label: '文档接入',
    icon: Document,
  },
  {
    key: 'knowledge-scope',
    label: '知识路由',
    icon: Folder,
  },
  {
    key: 'knowledge-topic',
    label: '路由追踪',
    icon: Document,
  },
  {
    key: 'dialog-observation',
    label: '对话观测',
    icon: Folder,
  },
]

const scopes = ref<KnowledgeScope[]>([])
const topics = ref<KnowledgeTopic[]>([])
const topicDocuments = ref<TopicDocumentRelation[]>([])

const currentScope = ref<KnowledgeScope | null>(null)
const currentTopic = ref<KnowledgeTopic | null>(null)
const selectedScopeCode = ref('')

const showScopeForm = ref(false)
const showTopicForm = ref(false)
const showSidebar = ref(true)

const documentForm = ref<UploadDocumentReq>({
  documentName: '',
  knowledgeScopeCode: '',
  knowledgeScopeName: '',
  businessCategory: '',
  documentTags: '',
})

const selectedFiles = ref<File[]>([])
const searchKeyword = ref('')
const currentPage = ref(1)

const stats = computed(() => ({
  total: documentStore.total,
  parsed: documentStore.documents.filter(d => d.parseStatus === 2).length,
  confirmed: documentStore.documents.filter(d => d.strategyStatus === 2).length,
  indexed: documentStore.documents.filter(d => d.indexStatus === 2).length,
}))

const scopeForm = ref({
  id: undefined as number | undefined,
  scopeCode: '',
  scopeName: '',
  parentScopeCode: '',
  description: '',
  aliases: '',
  examples: '',
  sortOrder: 0,
})

const topicForm = ref({
  id: undefined as number | undefined,
  topicCode: '',
  topicName: '',
  scopeCode: '',
  description: '',
  aliases: '',
  examples: '',
  answerShape: '',
  executionPreference: '',
  sortOrder: 0,
})

async function fetchDocuments() {
  await documentStore.fetchDocuments()
}

async function fetchScopes() {
  try {
    const { data } = await knowledgeApi.listScopes()
    scopes.value = data || []
  } catch {
    ElMessage.error('获取知识范围失败')
  }
}

async function fetchTopics(scopeCode?: string) {
  try {
    const { data } = await knowledgeApi.listTopics(scopeCode ? { scopeCode } : undefined)
    topics.value = data || []
  } catch {
    ElMessage.error('获取知识主题失败')
  }
}

async function fetchTopicDocuments(topicCode: string) {
  try {
    const res = await knowledgeApi.listTopicDocuments({ topicCode })
    topicDocuments.value = res.data || []
  } catch {
    ElMessage.error('获取主题关联文档失败')
  }
}

async function saveScope() {
  try {
    const data = { ...scopeForm.value }
    await knowledgeApi.saveScope(data)
    ElMessage.success('知识范围保存成功')
    showScopeForm.value = false
    await fetchScopes()
  } catch {
    ElMessage.error('保存失败')
  }
}

async function deleteScope(scopeCode: string) {
  if (!confirm('确定删除该知识范围吗？')) return
  try {
    await knowledgeApi.deleteScope({ scopeCode })
    ElMessage.success('删除成功')
    await fetchScopes()
  } catch {
    ElMessage.error('删除失败')
  }
}

async function saveTopic() {
  try {
    const data = { ...topicForm.value }
    await knowledgeApi.saveTopic(data)
    ElMessage.success('知识主题保存成功')
    showTopicForm.value = false
    await fetchTopics(selectedScopeCode.value || undefined)
  } catch {
    ElMessage.error('保存失败')
  }
}

async function deleteTopic(topicCode: string) {
  if (!confirm('确定删除该知识主题吗？')) return
  try {
    await knowledgeApi.deleteTopic({ topicCode })
    ElMessage.success('删除成功')
    await fetchTopics(selectedScopeCode.value || undefined)
  } catch {
    ElMessage.error('删除失败')
  }
}

function openScopeForm(scope?: KnowledgeScope) {
  if (scope) {
    scopeForm.value = { ...scope }
  } else {
    scopeForm.value = {
      id: undefined,
      scopeCode: '',
      scopeName: '',
      parentScopeCode: '',
      description: '',
      aliases: '',
      examples: '',
      sortOrder: 0,
    }
  }
  showScopeForm.value = true
}

function openTopicForm(topic?: KnowledgeTopic) {
  if (topic) {
    topicForm.value = { ...topic }
  } else {
    topicForm.value = {
      id: undefined,
      topicCode: '',
      topicName: '',
      scopeCode: selectedScopeCode.value,
      description: '',
      aliases: '',
      examples: '',
      answerShape: '',
      executionPreference: '',
      sortOrder: 0,
    }
  }
  showTopicForm.value = true
}

function handleScopeSelect(data: KnowledgeScope) {
  currentScope.value = data
  selectedScopeCode.value = data.scopeCode
  fetchTopics(data.scopeCode)
}

function handleTopicSelect(data: KnowledgeTopic) {
  currentTopic.value = data
  fetchTopicDocuments(data.topicCode)
}

function handleScopeChange() {
  currentTopic.value = null
  topicDocuments.value = []
  if (selectedScopeCode.value) {
    fetchTopics(selectedScopeCode.value)
  } else {
    fetchTopics()
  }
}

function handleViewDocument(documentId: number) {
  console.log('View document:', documentId)
}

async function handleFileSelect(event: Event) {
  const target = event.target as HTMLInputElement
  const files = target.files
  if (files) {
    selectedFiles.value = Array.from(files)
  }
}

function handleRemoveFile(index: number) {
  selectedFiles.value.splice(index, 1)
}

function handleClearFiles() {
  selectedFiles.value = []
}

async function handleUploadAndParse() {
  if (selectedFiles.value.length === 0) {
    ElMessage.warning('请选择文件')
    return
  }
  try {
    for (const file of selectedFiles.value) {
      await documentStore.uploadFile(file, {
        documentName: documentForm.value.documentName || file.name,
        knowledgeScopeCode: documentForm.value.knowledgeScopeCode || undefined,
        knowledgeScopeName: documentForm.value.knowledgeScopeName || undefined,
        businessCategory: documentForm.value.businessCategory || undefined,
        documentTags: documentForm.value.documentTags || undefined,
      })
    }
    ElMessage.success('上传并解析成功')
    selectedFiles.value = []
  } catch {
    ElMessage.error('上传失败')
  }
}

async function handleSearch() {
  await documentStore.fetchDocuments({ keyword: searchKeyword.value, pageNo: 1 })
}

function handlePageChange(page: number) {
  currentPage.value = page
  documentStore.fetchDocuments({ keyword: searchKeyword.value, pageNo: page })
}

function handleDeleteDocument(documentId: number) {
  if (!confirm('确定删除该文档吗？')) return
  documentStore.deleteDocument(documentId)
}

function goBack() {
  router.push('/')
}

function selectMenu(key: MenuKey) {
  activeMenu.value = key
  if (key === 'document-list') {
    fetchDocuments()
  }
}

function getCurrentTitle() {
  const item = navItems.find(i => i.key === activeMenu.value)
  return item?.label || '管理后台'
}

onMounted(() => {
  fetchDocuments()
  fetchScopes()
  fetchTopics()
})
</script>

<template>
  <div class="admin-view">
    <aside class="admin-sidebar" :class="{ collapsed: !showSidebar }">
      <div class="sidebar-header">
        <div class="brand">
          <div class="brand-logo">SA</div>
          <div class="brand-info">
            <div class="brand-title">Super Agent</div>
          </div>
        </div>
      </div>

      <div class="sidebar-section">
        <div class="section-label">导航</div>

        <template v-for="item in navItems" :key="item.key">
          <button
            class="nav-item"
            :class="{
              active: activeMenu === item.key,
            }"
            @click="selectMenu(item.key)"
          >
            <component :is="item.icon" :size="12" />
            <span class="nav-label">{{ item.label }}</span>
          </button>
        </template>
      </div>

      <div class="sidebar-footer">
        <button class="back-btn" @click="goBack">
          <ArrowLeft :size="12" />
          <span>返回聊天</span>
        </button>
      </div>
    </aside>

    <main class="admin-main" :class="{ 'sidebar-hidden': !showSidebar }">
      <header class="admin-header">
        <h1 class="admin-title">{{ getCurrentTitle() }}</h1>
      </header>

      <div class="admin-content">
        <div v-if="activeMenu === 'document-upload'" class="section-content">
          <div class="upload-header">
            <h2 class="upload-title">文档资料录入推荐流程</h2>
          </div>

          <div class="upload-layout">
            <div class="upload-form-panel">
              <div class="form-section">
                <div class="form-row">
                  <div class="form-item">
                    <label class="form-label">文档名称</label>
                    <div class="label-hint">不填则使用原始文件名</div>
                    <input
                      v-model="documentForm.documentName"
                      type="text"
                      class="form-input"
                      placeholder="例如 operation_rule"
                    />
                  </div>
                  <div class="form-item">
                    <label class="form-label">知识编码</label>
                    <input
                      v-model="documentForm.knowledgeScopeCode"
                      type="text"
                      class="form-input"
                      placeholder="例如 operation_rule"
                    />
                  </div>
                </div>

                <div class="form-row">
                  <div class="form-item">
                    <label class="form-label">知识名称</label>
                    <input
                      v-model="documentForm.knowledgeScopeName"
                      type="text"
                      class="form-input"
                      placeholder="例如 运营规则"
                    />
                  </div>
                  <div class="form-item">
                    <label class="form-label">业务分类</label>
                    <input
                      v-model="documentForm.businessCategory"
                      type="text"
                      class="form-input"
                      placeholder="例如 手册 / 规则 / 介绍"
                    />
                  </div>
                </div>

                <div class="form-row">
                  <div class="form-item">
                    <label class="form-label">文档标签</label>
                    <div class="label-hint">多个标签用英文逗号分隔</div>
                    <input
                      v-model="documentForm.documentTags"
                      type="text"
                      class="form-input"
                      placeholder="多个标签用英文逗号分隔"
                    />
                  </div>
                  <div class="form-item">
                    <label class="form-label">选择文件</label>
                    <div class="file-select-box">
                      <button class="file-select-btn" @click="$refs.fileInput.click()">
                        <Upload :size="14" />
                        <span>{{ selectedFiles.length > 0 ? selectedFiles.length + ' 个文件已选择' : '选择文件' }}</span>
                      </button>
                      <span class="file-count">{{ selectedFiles.length > 0 ? selectedFiles.length + ' 个文件已选择' : '未选择文件' }}</span>
                    </div>
                    <input
                      ref="fileInput"
                      type="file"
                      multiple
                      accept=".pdf,.doc,.docx,.txt,.md,.html"
                      class="hidden-file-input"
                      @change="handleFileSelect"
                    />
                  </div>
                </div>

                <div class="form-row">
                  <div class="form-item full-width">
                    <label class="form-label">文件列表</label>
                    <div class="file-list">
                      <div v-if="selectedFiles.length === 0" class="file-list-empty">
                        <Document :size="24" class="empty-icon" />
                        <span>请选择文件</span>
                      </div>
                      <div v-for="(file, index) in selectedFiles" :key="index" class="file-list-item">
                        <Document :size="14" class="file-icon" />
                        <span class="file-name">{{ file.name }}</span>
                        <button class="file-remove-btn" @click="handleRemoveFile(index)">
                          <Delete :size="12" />
                        </button>
                      </div>
                    </div>
                  </div>
                </div>

                <div class="form-row">
                  <div class="form-item full-width">
                    <div class="file-types-info">
                      <span>支持 PDF / DOC / DOCX / TXT / MD / HTML</span>
                    </div>
                    <div class="form-actions">
                      <button class="btn-clear" @click="handleClearFiles">
                        <Trash :size="14" />
                        <span>清空</span>
                      </button>
                      <button class="btn-upload" @click="handleUploadAndParse">
                        <Upload :size="14" />
                        <span>上传并解析</span>
                      </button>
                    </div>
                  </div>
                </div>
              </div>
            </div>

            <div class="upload-guide-panel">
              <h3 class="guide-title">建议操作顺序</h3>
              <ul class="guide-list">
                <li class="guide-item">
                  <span class="guide-number">1</span>
                  <span class="guide-text">先上传文档，系统会自动解析并生成推荐切分策略。</span>
                </li>
                <li class="guide-item">
                  <span class="guide-number">2</span>
                  <span class="guide-text">点击任意文档，进入单篇详情页面解析策略、Chunk 和任务轨迹。</span>
                </li>
                <li class="guide-item">
                  <span class="guide-number">3</span>
                  <span class="guide-text">在详情页确认策略并构建索引，列表页专注浏览和筛选。</span>
                </li>
              </ul>
            </div>
          </div>

          <div class="document-section">
            <div class="section-header">
              <h3 class="section-title">文档列表</h3>
              <div class="search-box">
                <input
                  v-model="searchKeyword"
                  type="text"
                  class="search-input"
                  placeholder="搜索文档名称或原始文件名"
                  @keyup.enter="handleSearch"
                />
                <button class="search-btn" @click="handleSearch">
                  <Search :size="14" />
                  <span>搜索</span>
                </button>
              </div>
            </div>

            <div class="stats-cards">
              <div class="stat-card">
                <div class="stat-label">当前文档</div>
                <div class="stat-value">{{ stats.total }}</div>
              </div>
              <div class="stat-card">
                <div class="stat-label">解析完成</div>
                <div class="stat-value success">{{ stats.parsed }}</div>
              </div>
              <div class="stat-card">
                <div class="stat-label">策略确认</div>
                <div class="stat-value success">{{ stats.confirmed }}</div>
              </div>
              <div class="stat-card">
                <div class="stat-label">索引可用</div>
                <div class="stat-value success">{{ stats.indexed }}</div>
              </div>
            </div>

            <div class="document-table-container">
              <table class="document-table">
                <thead>
                  <tr>
                    <th>名称</th>
                    <th>类型</th>
                    <th>大小</th>
                    <th>更新时间</th>
                    <th>解析</th>
                    <th>策略</th>
                    <th>索引</th>
                    <th>操作</th>
                  </tr>
                </thead>
                <tbody>
                  <tr v-for="doc in documentStore.documents" :key="doc.documentId">
                    <td class="doc-name-cell">
                      <span class="doc-name">{{ doc.documentName }}</span>
                      <span class="doc-original-name">{{ doc.originalFileName }}</span>
                    </td>
                    <td>
                      <span class="type-badge">{{ doc.fileTypeName || 'PDF' }}</span>
                    </td>
                    <td>{{ (doc.fileSize / 1024).toFixed(1) }} KB</td>
                    <td>{{ doc.updateTime }}</td>
                    <td>
                      <span :class="['status-badge', doc.parseStatus === 2 ? 'status-success' : 'status-pending']">
                        {{ doc.parseStatusName || (doc.parseStatus === 2 ? '解析成功' : '待解析') }}
                      </span>
                    </td>
                    <td>
                      <span :class="['status-badge', doc.strategyStatus === 2 ? 'status-success' : 'status-pending']">
                        {{ doc.strategyStatusName || (doc.strategyStatus === 2 ? '已确认' : '待确认') }}
                      </span>
                    </td>
                    <td>
                      <span :class="['status-badge', doc.indexStatus === 2 ? 'status-success' : 'status-pending']">
                        {{ doc.indexStatusName || (doc.indexStatus === 2 ? '构建成功' : '待构建') }}
                      </span>
                    </td>
                    <td>
                      <button class="action-btn view" @click="handleViewDocument(doc.documentId)">
                        <View :size="14" />
                        <span>查看详情</span>
                      </button>
                      <button class="action-btn delete" @click="handleDeleteDocument(doc.documentId)">
                        <Delete :size="14" />
                        <span>删除</span>
                      </button>
                    </td>
                  </tr>
                </tbody>
              </table>

              <div v-if="documentStore.total === 0" class="empty-table">
                <Document :size="48" class="empty-icon" />
                <p>暂无文档</p>
              </div>
            </div>

            <div class="pagination-container">
              <span class="pagination-info">
                共 {{ documentStore.total }} 份文档，当前第 {{ documentStore.pageNo }} 页。
              </span>
              <div class="pagination">
                <button
                  class="page-btn"
                  :disabled="documentStore.pageNo <= 1"
                  @click="handlePageChange(documentStore.pageNo - 1)"
                >
                  上一页
                </button>
                <span class="page-current">{{ documentStore.pageNo }}</span>
                <button
                  class="page-btn"
                  :disabled="documentStore.pageNo >= Math.ceil(documentStore.total / documentStore.pageSize)"
                  @click="handlePageChange(documentStore.pageNo + 1)"
                >
                  下一页
                </button>
              </div>
            </div>
          </div>
        </div>

        <div v-else-if="activeMenu === 'document-list'" class="section-content">
          <DocumentList @view="handleViewDocument" />
        </div>

        <div v-else-if="activeMenu === 'knowledge-scope'" class="section-content">
          <div class="panel-header">
            <h3 class="panel-title">知识范围树</h3>
            <button class="add-btn" @click="openScopeForm()">
              <Plus :size="12" />
              <span>新增范围</span>
            </button>
          </div>

          <div class="scope-content">
            <div class="tree-container">
              <ElTree
                :data="scopes"
                :props="{ label: 'scopeName', children: 'children', value: 'scopeCode' }"
                node-key="scopeCode"
                default-expand-all
                :highlight-current="true"
                @node-click="handleScopeSelect"
                class="scope-tree"
              >
                <template #default="{ data }">
                  <span class="tree-node">
                    <span>{{ data.scopeName }}</span>
                    <div class="node-actions">
                      <button class="action-icon" @click.stop="openScopeForm(data)">
                        <Edit :size="10" />
                      </button>
                      <button class="action-icon delete" @click.stop="deleteScope(data.scopeCode)">
                        <Delete :size="10" />
                      </button>
                    </div>
                  </span>
                </template>
              </ElTree>
            </div>

            <div v-if="currentScope" class="detail-panel">
              <h4 class="detail-title">范围详情</h4>
              <div class="detail-content">
                <div class="detail-item">
                  <span class="detail-label">编码</span>
                  <span class="detail-value">{{ currentScope.scopeCode }}</span>
                </div>
                <div class="detail-item">
                  <span class="detail-label">名称</span>
                  <span class="detail-value">{{ currentScope.scopeName }}</span>
                </div>
                <div class="detail-item">
                  <span class="detail-label">父范围</span>
                  <span class="detail-value">{{ currentScope.parentScopeCode || '无' }}</span>
                </div>
                <div class="detail-item">
                  <span class="detail-label">描述</span>
                  <span class="detail-value">{{ currentScope.description || '-' }}</span>
                </div>
                <div class="detail-item">
                  <span class="detail-label">别名</span>
                  <span class="detail-value">{{ currentScope.aliases || '-' }}</span>
                </div>
                <div class="detail-item">
                  <span class="detail-label">示例</span>
                  <span class="detail-value">{{ currentScope.examples || '-' }}</span>
                </div>
              </div>
            </div>
          </div>
        </div>

        <div v-else-if="activeMenu === 'knowledge-topic'" class="section-content">
          <div class="panel-header">
            <h3 class="panel-title">知识主题</h3>
            <div class="header-right">
              <select :value="selectedScopeCode" @change="handleScopeChange" class="scope-select">
                <option value="">全部范围</option>
                <option v-for="scope in scopes" :key="scope.scopeCode" :value="scope.scopeCode">
                  {{ scope.scopeName }}
                </option>
              </select>
              <button class="add-btn" @click="openTopicForm()">
                <Plus :size="12" />
                <span>新增主题</span>
              </button>
            </div>
          </div>

          <div class="topic-content">
            <div v-if="topics.length === 0" class="empty-state">
              <Document :size="32" class="empty-icon" />
              <p>暂无主题</p>
              <p class="empty-hint">选择知识范围后查看对应主题</p>
            </div>

            <div v-else class="topic-grid">
              <div
                v-for="topic in topics"
                :key="topic.topicCode"
                class="topic-card"
                :class="{ active: currentTopic?.topicCode === topic.topicCode }"
                @click="handleTopicSelect(topic)"
              >
                <div class="topic-header">
                  <h4 class="topic-name">{{ topic.topicName }}</h4>
                  <div class="topic-actions">
                    <button class="action-icon" @click.stop="openTopicForm(topic)">
                      <Edit :size="10" />
                    </button>
                    <button class="action-icon delete" @click.stop="deleteTopic(topic.topicCode)">
                      <Delete :size="10" />
                    </button>
                  </div>
                </div>
                <p class="topic-description">{{ topic.description || '暂无描述' }}</p>
                <div class="topic-meta">
                  <span class="topic-code">{{ topic.topicCode }}</span>
                </div>
              </div>
            </div>

            <div v-if="currentTopic" class="document-panel">
              <h4 class="detail-title">关联文档</h4>
              <div v-if="topicDocuments.length === 0" class="empty-documents">
                <p>暂无关联文档</p>
              </div>
              <div v-else class="document-list">
                <div
                  v-for="doc in topicDocuments"
                  :key="`${doc.topicCode}-${doc.documentId}`"
                  class="document-item"
                >
                  <div class="doc-info">
                    <span class="doc-name">{{ doc.documentName }}</span>
                    <span class="doc-scope">{{ doc.knowledgeScopeName }}</span>
                  </div>
                  <div class="doc-score">
                    <span>关联度: {{ doc.relationScore }}</span>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </main>

    <ElDialog v-model="showScopeForm" :title="scopeForm.id ? '编辑知识范围' : '新增知识范围'" width="500px">
      <ElForm :model="scopeForm" label-width="100px">
        <ElFormItem label="范围编码" required>
          <ElInput v-model="scopeForm.scopeCode" placeholder="请输入范围编码" />
        </ElFormItem>
        <ElFormItem label="范围名称" required>
          <ElInput v-model="scopeForm.scopeName" placeholder="请输入范围名称" />
        </ElFormItem>
        <ElFormItem label="父范围">
          <ElSelect v-model="scopeForm.parentScopeCode" placeholder="请选择父范围">
            <ElOption label="无" value="" />
            <ElOption v-for="scope in scopes" :key="scope.scopeCode" :label="scope.scopeName" :value="scope.scopeCode" />
          </ElSelect>
        </ElFormItem>
        <ElFormItem label="描述">
          <ElInput v-model="scopeForm.description" type="textarea" :rows="3" />
        </ElFormItem>
        <ElFormItem label="别名">
          <ElInput v-model="scopeForm.aliases" placeholder="多个别名用逗号分隔" />
        </ElFormItem>
        <ElFormItem label="示例">
          <ElInput v-model="scopeForm.examples" type="textarea" :rows="2" />
        </ElFormItem>
        <ElFormItem label="排序">
          <ElInput v-model.number="scopeForm.sortOrder" type="number" />
        </ElFormItem>
      </ElForm>
      <template #footer>
        <ElButton @click="showScopeForm = false">取消</ElButton>
        <ElButton type="primary" @click="saveScope">确定</ElButton>
      </template>
    </ElDialog>

    <ElDialog v-model="showTopicForm" :title="topicForm.id ? '编辑知识主题' : '新增知识主题'" width="500px">
      <ElForm :model="topicForm" label-width="100px">
        <ElFormItem label="主题编码" required>
          <ElInput v-model="topicForm.topicCode" placeholder="请输入主题编码" />
        </ElFormItem>
        <ElFormItem label="主题名称" required>
          <ElInput v-model="topicForm.topicName" placeholder="请输入主题名称" />
        </ElFormItem>
        <ElFormItem label="所属范围" required>
          <ElSelect v-model="topicForm.scopeCode" placeholder="请选择知识范围">
            <ElOption v-for="scope in scopes" :key="scope.scopeCode" :label="scope.scopeName" :value="scope.scopeCode" />
          </ElSelect>
        </ElFormItem>
        <ElFormItem label="描述">
          <ElInput v-model="topicForm.description" type="textarea" :rows="3" />
        </ElFormItem>
        <ElFormItem label="别名">
          <ElInput v-model="topicForm.aliases" placeholder="多个别名用逗号分隔" />
        </ElFormItem>
        <ElFormItem label="示例">
          <ElInput v-model="topicForm.examples" type="textarea" :rows="2" />
        </ElFormItem>
        <ElFormItem label="回答形态">
          <ElInput v-model="topicForm.answerShape" />
        </ElFormItem>
        <ElFormItem label="执行偏好">
          <ElInput v-model="topicForm.executionPreference" />
        </ElFormItem>
        <ElFormItem label="排序">
          <ElInput v-model.number="topicForm.sortOrder" type="number" />
        </ElFormItem>
      </ElForm>
      <template #footer>
        <ElButton @click="showTopicForm = false">取消</ElButton>
        <ElButton type="primary" @click="saveTopic">确定</ElButton>
      </template>
    </ElDialog>
  </div>
</template>

<style scoped>
.admin-view {
  height: 100%;
  display: flex;
  background: #f1f5f9;
}

.admin-sidebar {
  width: 180px;
  background: #ffffff;
  color: #333333;
  display: flex;
  flex-direction: column;
  position: fixed;
  left: 0;
  top: 0;
  bottom: 0;
  z-index: 100;
  transition: transform 0.3s ease;
  overflow-y: auto;
  overflow-x: hidden;
  border-right: 1px solid #f0f0f0;
}

.admin-sidebar.collapsed {
  transform: translateX(-100%);
}

.admin-sidebar::-webkit-scrollbar {
  width: 4px;
}

.admin-sidebar::-webkit-scrollbar-thumb {
  background: #334155;
  border-radius: 2px;
}

.sidebar-header {
  padding: 16px 12px;
  border-bottom: 1px solid #f0f0f0;
}

.brand {
  display: flex;
  align-items: center;
  gap: 8px;
}

.brand-logo {
  width: 24px;
  height: 24px;
  border-radius: 6px;
  background: #2563eb;
  color: #fff;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 11px;
  font-weight: 700;
  flex-shrink: 0;
}

.brand-info {
  display: flex;
  flex-direction: column;
  gap: 1px;
  min-width: 0;
}

.brand-title {
  font-size: 14px;
  font-weight: 600;
  color: #1a1a1a;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.sidebar-section {
  flex: 1;
  padding: 12px 8px;
}

.section-label {
  font-size: 10px;
  color: #999999;
  text-transform: uppercase;
  letter-spacing: 1px;
  padding: 0 8px 8px;
  font-weight: 600;
}

.nav-item {
  display: flex;
  align-items: center;
  gap: 8px;
  width: 100%;
  padding: 7px 10px;
  background: transparent;
  border: none;
  border-radius: 5px;
  font-size: 13px;
  font-weight: 500;
  color: #666666;
  cursor: pointer;
  transition: all 0.2s;
  white-space: nowrap;
  text-align: left;
  margin-bottom: 1px;
}

.nav-item:hover {
  background: #f0f5ff;
  color: #2563eb;
}

.nav-item.active {
  background: #2563eb;
  color: #fff;
}

.nav-label {
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
}



.sidebar-footer {
  padding: 12px 12px;
  border-top: 1px solid #f0f0f0;
}

.back-btn {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 7px 12px;
  background: transparent;
  border: 1px solid #d0d0d0;
  border-radius: 5px;
  font-size: 12px;
  color: #666666;
  cursor: pointer;
  transition: all 0.2s;
  width: 100%;
  white-space: nowrap;
}

.back-btn:hover {
  background: #f0f5ff;
  border-color: #2563eb;
  color: #2563eb;
}

.admin-main {
  flex: 1;
  margin-left: 180px;
  display: flex;
  flex-direction: column;
  transition: margin-left 0.3s ease;
  min-width: 0;
}

.admin-main.sidebar-hidden {
  margin-left: 0;
}

.admin-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 16px 24px;
  background: #fff;
  border-bottom: 1px solid #e2e8f0;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.05);
}

.admin-title {
  font-size: 20px;
  font-weight: 600;
  color: #1e293b;
  margin: 0;
}

.admin-content {
  flex: 1;
  padding: 20px;
  overflow-y: auto;
}

.section-content {
  background: #fff;
  border-radius: 12px;
  padding: 24px;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.05);
}

.panel-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 20px;
  padding-bottom: 16px;
  border-bottom: 1px solid #f1f5f9;
}

.panel-title {
  font-size: 16px;
  font-weight: 600;
  color: #1e293b;
  margin: 0;
}

.header-right {
  display: flex;
  align-items: center;
  gap: 12px;
}

.scope-select {
  padding: 8px 14px;
  border: 1px solid #e2e8f0;
  border-radius: 8px;
  font-size: 14px;
  color: #475569;
  background: #fff;
  cursor: pointer;
}

.add-btn {
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 6px 12px;
  background: #3b82f6;
  border: none;
  border-radius: 6px;
  font-size: 12px;
  color: #fff;
  cursor: pointer;
  transition: background 0.2s;
}

.add-btn:hover {
  background: #2563eb;
}

.scope-content {
  display: flex;
  gap: 24px;
  min-height: 400px;
}

.tree-container {
  flex: 1;
  background: #f8fafc;
  border-radius: 12px;
  padding: 20px;
  overflow-y: auto;
}

.scope-tree {
  max-height: 100%;
}

.tree-node {
  display: flex;
  align-items: center;
  justify-content: space-between;
  width: 100%;
}

.node-actions {
  display: flex;
  gap: 4px;
}

.action-icon {
  background: transparent;
  border: none;
  color: #94a3b8;
  cursor: pointer;
  padding: 2px;
  border-radius: 3px;
  transition: all 0.2s;
  display: inline-flex;
  align-items: center;
  justify-content: center;
}

.action-icon:hover {
  background: #e2e8f0;
}

.action-icon.delete:hover {
  background: #fee2e2;
  color: #dc2626;
}

.detail-panel {
  width: 320px;
  background: #f8fafc;
  border-radius: 12px;
  padding: 20px;
  flex-shrink: 0;
}

.detail-title {
  font-size: 14px;
  font-weight: 600;
  color: #475569;
  margin: 0 0 16px 0;
  padding-bottom: 12px;
  border-bottom: 1px solid #e2e8f0;
}

.detail-content {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.detail-item {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.detail-label {
  font-size: 12px;
  color: #94a3b8;
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.detail-value {
  font-size: 14px;
  color: #334155;
  word-break: break-all;
}

.topic-content {
  min-height: 300px;
}

.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 60px 20px;
  color: #94a3b8;
  background: #f8fafc;
  border-radius: 12px;
}

.empty-icon {
  width: 40px;
  height: 40px;
  margin-bottom: 12px;
  opacity: 0.5;
}

.empty-hint {
  font-size: 14px;
  margin-top: 8px;
}

.topic-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(320px, 1fr));
  gap: 16px;
}

.topic-card {
  background: #f8fafc;
  border-radius: 12px;
  padding: 20px;
  cursor: pointer;
  transition: all 0.2s;
  border: 2px solid transparent;
}

.topic-card:hover {
  box-shadow: 0 4px 16px rgba(0, 0, 0, 0.08);
  background: #fff;
}

.topic-card.active {
  border-color: #3b82f6;
  background: #eff6ff;
}

.topic-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 12px;
}

.topic-name {
  font-size: 16px;
  font-weight: 600;
  color: #1e293b;
  margin: 0;
}

.topic-actions {
  display: flex;
  gap: 4px;
}

.topic-description {
  font-size: 14px;
  color: #64748b;
  margin: 0 0 12px 0;
  line-height: 1.5;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

.topic-meta {
  display: flex;
  align-items: center;
}

.topic-code {
  font-size: 12px;
  color: #94a3b8;
  background: #e2e8f0;
  padding: 4px 10px;
  border-radius: 4px;
}

.document-panel {
  margin-top: 20px;
  background: #f8fafc;
  border-radius: 12px;
  padding: 20px;
}

.empty-documents {
  text-align: center;
  padding: 30px;
  color: #94a3b8;
}

.document-list {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.document-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 14px 16px;
  background: #fff;
  border-radius: 8px;
}

.doc-info {
  flex: 1;
}

.doc-name {
  display: block;
  font-size: 14px;
  font-weight: 500;
  color: #1e293b;
}

.doc-scope {
  font-size: 12px;
  color: #94a3b8;
}

.doc-score {
  font-size: 13px;
  color: #3b82f6;
  font-weight: 500;
}

.upload-header {
  margin-bottom: 24px;
}

.upload-title {
  font-size: 20px;
  font-weight: 600;
  color: #1e293b;
  margin: 0;
}

.upload-layout {
  display: flex;
  gap: 24px;
  margin-bottom: 32px;
}

.upload-form-panel {
  flex: 1;
  background: #f8fafc;
  border-radius: 12px;
  padding: 24px;
}

.upload-guide-panel {
  width: 320px;
  background: #fff;
  border-radius: 12px;
  padding: 24px;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.05);
  border: 1px solid #e2e8f0;
  flex-shrink: 0;
}

.guide-title {
  font-size: 16px;
  font-weight: 600;
  color: #1e293b;
  margin: 0 0 16px 0;
  padding-bottom: 12px;
  border-bottom: 1px solid #f1f5f9;
}

.guide-list {
  list-style: none;
  padding: 0;
  margin: 0;
}

.guide-item {
  display: flex;
  gap: 12px;
  padding: 12px 0;
  border-bottom: 1px solid #f1f5f9;
}

.guide-item:last-child {
  border-bottom: none;
}

.guide-number {
  width: 24px;
  height: 24px;
  border-radius: 50%;
  background: #3b82f6;
  color: #fff;
  font-size: 12px;
  font-weight: 600;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.guide-text {
  font-size: 13px;
  color: #475569;
  line-height: 1.6;
}

.form-section {
  display: flex;
  flex-direction: column;
  gap: 0;
}

.form-row {
  display: flex;
  gap: 24px;
  margin-bottom: 20px;
}

.form-item {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.form-item.full-width {
  flex: 100%;
}

.form-label {
  font-size: 14px;
  font-weight: 500;
  color: #334155;
}

.label-hint {
  font-size: 12px;
  color: #94a3b8;
}

.form-input {
  padding: 10px 14px;
  border: 1px solid #e2e8f0;
  border-radius: 8px;
  font-size: 14px;
  color: #334155;
  background: #fff;
  transition: border-color 0.2s;
}

.form-input:focus {
  outline: none;
  border-color: #3b82f6;
}

.form-input::placeholder {
  color: #cbd5e1;
}

.file-select-box {
  display: flex;
  align-items: center;
  gap: 12px;
}

.file-select-btn {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 8px 16px;
  background: #fff;
  border: 1px solid #e2e8f0;
  border-radius: 8px;
  font-size: 14px;
  color: #3b82f6;
  cursor: pointer;
  transition: all 0.2s;
}

.file-select-btn:hover {
  background: #eff6ff;
  border-color: #3b82f6;
}

.file-count {
  font-size: 13px;
  color: #94a3b8;
}

.hidden-file-input {
  display: none;
}

.file-list {
  background: #fff;
  border: 1px solid #e2e8f0;
  border-radius: 8px;
  min-height: 100px;
  padding: 12px;
}

.file-list-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  height: 80px;
  gap: 8px;
  color: #94a3b8;
}

.file-list-empty .empty-icon {
  opacity: 0.5;
}

.file-list-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 12px;
  background: #f8fafc;
  border-radius: 6px;
  margin-bottom: 8px;
}

.file-list-item:last-child {
  margin-bottom: 0;
}

.file-icon {
  color: #64748b;
  flex-shrink: 0;
}

.file-name {
  flex: 1;
  font-size: 13px;
  color: #334155;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.file-remove-btn {
  background: transparent;
  border: none;
  color: #94a3b8;
  cursor: pointer;
  padding: 4px;
  border-radius: 4px;
  transition: all 0.2s;
}

.file-remove-btn:hover {
  background: #fee2e2;
  color: #dc2626;
}

.file-types-info {
  padding: 12px 14px;
  background: #f8fafc;
  border-radius: 8px;
  margin-bottom: 16px;
}

.file-types-info span {
  font-size: 13px;
  color: #64748b;
}

.form-actions {
  display: flex;
  gap: 12px;
}

.btn-clear {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 10px 20px;
  background: #fff;
  border: 1px solid #e2e8f0;
  border-radius: 8px;
  font-size: 14px;
  color: #64748b;
  cursor: pointer;
  transition: all 0.2s;
}

.btn-clear:hover {
  background: #f1f5f9;
  border-color: #cbd5e1;
}

.btn-upload {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 10px 20px;
  background: #3b82f6;
  border: none;
  border-radius: 8px;
  font-size: 14px;
  color: #fff;
  cursor: pointer;
  transition: background 0.2s;
}

.btn-upload:hover {
  background: #2563eb;
}

.document-section {
  margin-top: 24px;
}

.section-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 20px;
}

.section-title {
  font-size: 16px;
  font-weight: 600;
  color: #1e293b;
  margin: 0;
}

.search-box {
  display: flex;
  align-items: center;
  gap: 8px;
}

.search-input {
  padding: 8px 14px;
  border: 1px solid #e2e8f0;
  border-radius: 8px;
  font-size: 14px;
  color: #334155;
  min-width: 200px;
}

.search-input:focus {
  outline: none;
  border-color: #3b82f6;
}

.search-btn {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 8px 16px;
  background: #fff;
  border: 1px solid #3b82f6;
  border-radius: 8px;
  font-size: 14px;
  color: #3b82f6;
  cursor: pointer;
  transition: all 0.2s;
}

.search-btn:hover {
  background: #eff6ff;
}

.stats-cards {
  display: flex;
  gap: 16px;
  margin-bottom: 20px;
}

.stat-card {
  flex: 1;
  background: #f8fafc;
  border-radius: 12px;
  padding: 20px;
  text-align: center;
}

.stat-label {
  font-size: 13px;
  color: #64748b;
  margin-bottom: 8px;
}

.stat-value {
  font-size: 24px;
  font-weight: 700;
  color: #1e293b;
}

.stat-value.success {
  color: #22c55e;
}

.document-table-container {
  background: #f8fafc;
  border-radius: 12px;
  padding: 20px;
  overflow-x: auto;
}

.document-table {
  width: 100%;
  border-collapse: collapse;
}

.document-table thead {
  background: #fff;
}

.document-table th {
  padding: 14px 16px;
  text-align: left;
  font-size: 13px;
  font-weight: 600;
  color: #475569;
  border-bottom: 2px solid #e2e8f0;
}

.document-table td {
  padding: 14px 16px;
  font-size: 14px;
  color: #334155;
  border-bottom: 1px solid #e2e8f0;
}

.document-table tbody tr:hover {
  background: #fff;
}

.doc-name-cell {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.doc-name {
  font-weight: 500;
  color: #1e293b;
}

.doc-original-name {
  font-size: 12px;
  color: #94a3b8;
}

.type-badge {
  display: inline-block;
  padding: 4px 10px;
  background: #e2e8f0;
  border-radius: 4px;
  font-size: 12px;
  color: #64748b;
}

.status-badge {
  display: inline-block;
  padding: 4px 10px;
  border-radius: 4px;
  font-size: 12px;
}

.status-badge.status-success {
  background: #dcfce7;
  color: #22c55e;
}

.status-badge.status-pending {
  background: #fef3c7;
  color: #f59e0b;
}

.action-btn {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 6px 12px;
  border-radius: 6px;
  font-size: 12px;
  cursor: pointer;
  border: none;
  transition: all 0.2s;
  margin-right: 8px;
}

.action-btn:last-child {
  margin-right: 0;
}

.action-btn.view {
  background: #eff6ff;
  color: #3b82f6;
}

.action-btn.view:hover {
  background: #dbeafe;
}

.action-btn.delete {
  background: #fef2f2;
  color: #dc2626;
}

.action-btn.delete:hover {
  background: #fee2e2;
}

.empty-table {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 60px 20px;
  color: #94a3b8;
}

.empty-table .empty-icon {
  opacity: 0.5;
  margin-bottom: 12px;
}

.pagination-container {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-top: 20px;
  padding: 16px 20px;
  background: #fff;
  border-radius: 12px;
}

.pagination-info {
  font-size: 13px;
  color: #64748b;
}

.pagination {
  display: flex;
  align-items: center;
  gap: 12px;
}

.page-btn {
  padding: 8px 16px;
  background: #fff;
  border: 1px solid #e2e8f0;
  border-radius: 6px;
  font-size: 13px;
  color: #475569;
  cursor: pointer;
  transition: all 0.2s;
}

.page-btn:hover:not(:disabled) {
  background: #f1f5f9;
}

.page-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.page-current {
  font-size: 14px;
  font-weight: 600;
  color: #3b82f6;
  padding: 0 8px;
}

@media (max-width: 768px) {
  .admin-sidebar {
    transform: translateX(-100%);
  }

  .admin-sidebar.collapsed {
    transform: translateX(0);
  }

  .admin-main {
    margin-left: 0;
  }

  .admin-header {
    padding: 12px 16px;
  }

  .admin-title {
    font-size: 18px;
  }

  .admin-content {
    padding: 12px;
  }

  .section-content {
    padding: 16px;
  }

  .scope-content {
    flex-direction: column;
  }

  .detail-panel {
    width: 100%;
  }

  .topic-grid {
    grid-template-columns: 1fr;
  }

  .header-right {
    flex-direction: column;
    align-items: stretch;
  }

  .scope-select {
    width: 100%;
  }

  .panel-header {
    flex-direction: column;
    align-items: flex-start;
    gap: 12px;
  }
}
</style>
