<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { knowledgeApi } from '@/api/knowledge'
import { ElMessage, ElTree, ElButton, ElInput, ElDialog, ElForm, ElFormItem, ElSelect, ElOption } from 'element-plus'
import { Plus, Edit, Delete, Notebook, CollectionTag } from '@element-plus/icons-vue'
import type { KnowledgeScope, KnowledgeTopic, TopicDocumentRelation } from '@/types'

const activeTab = ref<'scope' | 'topic'>('scope')

const scopes = ref<KnowledgeScope[]>([])
const topics = ref<KnowledgeTopic[]>([])
const topicDocuments = ref<TopicDocumentRelation[]>([])

const currentScope = ref<KnowledgeScope | null>(null)
const currentTopic = ref<KnowledgeTopic | null>(null)
const selectedScopeCode = ref('')

const showScopeForm = ref(false)
const showTopicForm = ref(false)


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

onMounted(() => {
  fetchScopes()
  fetchTopics()
})
</script>

<template>
  <div class="knowledge-view">
    <div class="tabs-container">
      <button class="tab-btn" :class="{ active: activeTab === 'scope' }" @click="activeTab = 'scope'">
        <Notebook :size="16" />
        <span>知识范围管理</span>
      </button>
      <button class="tab-btn" :class="{ active: activeTab === 'topic' }" @click="activeTab = 'topic'">
        <CollectionTag :size="16" />
        <span>知识主题管理</span>
      </button>
    </div>

    <div class="content-container">
      <div v-if="activeTab === 'scope'" class="scope-panel">
        <div class="panel-header">
          <div class="header-left">
            <Notebook class="header-icon" />
            <h3 class="header-title">知识范围树</h3>
          </div>
          <button class="add-btn" @click="openScopeForm()">
            <Plus :size="16" />
            <span>新增范围</span>
          </button>
        </div>

        <div class="tree-container">
          <ElTree :data="scopes" :props="{ label: 'scopeName', children: 'children', value: 'scopeCode' }"
            node-key="scopeCode" default-expand-all :highlight-current="true" @node-click="handleScopeSelect"
            class="scope-tree">
            <template #default="{ data }">
              <span class="tree-node">
                <span>{{ data.scopeName }}</span>
                <div class="node-actions">
                  <button class="action-icon" @click.stop="openScopeForm(data)">
                    <Edit :size="12" />
                  </button>
                  <button class="action-icon delete" @click.stop="deleteScope(data.scopeCode)">
                    <Delete :size="12" />
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

      <div v-else class="topic-panel">
        <div class="panel-header">
          <div class="header-left">
            <CollectionTag class="header-icon" />
            <h3 class="header-title">知识主题</h3>
          </div>
          <div class="header-right">
            <select :value="selectedScopeCode" @change="handleScopeChange" class="scope-select">
              <option value="">全部范围</option>
              <option v-for="scope in scopes" :key="scope.scopeCode" :value="scope.scopeCode">
                {{ scope.scopeName }}
              </option>
            </select>
            <button class="add-btn" @click="openTopicForm()">
              <Plus :size="16" />
              <span>新增主题</span>
            </button>
          </div>
        </div>

        <div class="topic-list">
          <div v-if="topics.length === 0" class="empty-state">
            <CollectionTag :size="48" class="empty-icon" />
            <p>暂无主题</p>
            <p class="empty-hint">选择知识范围后查看对应主题</p>
          </div>

          <div v-else class="topic-grid">
            <div v-for="topic in topics" :key="topic.topicCode" class="topic-card"
              :class="{ active: currentTopic?.topicCode === topic.topicCode }" @click="handleTopicSelect(topic)">
              <div class="topic-header">
                <h4 class="topic-name">{{ topic.topicName }}</h4>
                <div class="topic-actions">
                  <button class="action-icon" @click.stop="openTopicForm(topic)">
                    <Edit :size="12" />
                  </button>
                  <button class="action-icon delete" @click.stop="deleteTopic(topic.topicCode)">
                    <Delete :size="12" />
                  </button>
                </div>
              </div>
              <p class="topic-description">{{ topic.description || '暂无描述' }}</p>
              <div class="topic-meta">
                <span class="topic-code">{{ topic.topicCode }}</span>
              </div>
            </div>
          </div>
        </div>

        <div v-if="currentTopic" class="document-panel">
          <h4 class="detail-title">关联文档</h4>
          <div v-if="topicDocuments.length === 0" class="empty-documents">
            <p>暂无关联文档</p>
          </div>
          <div v-else class="document-list">
            <div v-for="doc in topicDocuments" :key="`${doc.topicCode}-${doc.documentId}`" class="document-item">
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
            <ElOption v-for="scope in scopes" :key="scope.scopeCode" :label="scope.scopeName"
              :value="scope.scopeCode" />
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
            <ElOption v-for="scope in scopes" :key="scope.scopeCode" :label="scope.scopeName"
              :value="scope.scopeCode" />
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
.knowledge-view {
  height: 100%;
  display: flex;
  flex-direction: column;
}

.tabs-container {
  display: flex;
  gap: 8px;
  padding: 0 24px 16px;
}

.tab-btn {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 10px 24px;
  background: #fff;
  border: 1px solid #e2e8f0;
  border-radius: 8px;
  font-size: 14px;
  font-weight: 500;
  color: #64748b;
  cursor: pointer;
  transition: all 0.2s;
}

.tab-btn:hover {
  border-color: #3b82f6;
  color: #3b82f6;
}

.tab-btn.active {
  background: #3b82f6;
  border-color: #3b82f6;
  color: #fff;
}

.content-container {
  flex: 1;
  padding: 0 24px 24px;
  overflow-y: auto;
}

.panel-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 20px;
}

.header-left {
  display: flex;
  align-items: center;
  gap: 8px;
}

.header-icon {
  width: 20px;
  height: 20px;
  color: #3b82f6;
}

.header-title {
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
  padding: 6px 12px;
  border: 1px solid #e2e8f0;
  border-radius: 6px;
  font-size: 13px;
  color: #475569;
  background: #fff;
}

.add-btn {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 8px 16px;
  background: #3b82f6;
  border: none;
  border-radius: 8px;
  font-size: 14px;
  color: #fff;
  cursor: pointer;
  transition: background 0.2s;
}

.add-btn:hover {
  background: #2563eb;
}

.scope-panel {
  display: flex;
  gap: 20px;
  height: calc(100% - 50px);
}

.tree-container {
  flex: 1;
  background: #fff;
  border-radius: 12px;
  padding: 16px;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.05);
  overflow-y: auto;
}

.scope-tree {
  max-height: calc(100% - 20px);
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
  padding: 4px;
  border-radius: 4px;
  transition: all 0.2s;
}

.action-icon:hover {
  background: #f1f5f9;
}

.action-icon.delete:hover {
  background: #fee2e2;
  color: #dc2626;
}

.detail-panel {
  width: 300px;
  background: #fff;
  border-radius: 12px;
  padding: 20px;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.05);
  flex-shrink: 0;
}

.detail-title {
  font-size: 14px;
  font-weight: 600;
  color: #475569;
  margin: 0 0 16px 0;
  padding-bottom: 8px;
  border-bottom: 1px solid #f1f5f9;
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
}

.detail-value {
  font-size: 13px;
  color: #334155;
  word-break: break-all;
}

.topic-panel {
  height: calc(100% - 50px);
}

.topic-list {
  margin-bottom: 20px;
}

.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 60px 20px;
  color: #94a3b8;
  background: #fff;
  border-radius: 12px;
}

.empty-icon {
  width: 64px;
  height: 64px;
  margin-bottom: 16px;
  opacity: 0.5;
}

.empty-hint {
  font-size: 14px;
  margin-top: 8px;
}

.topic-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
  gap: 16px;
}

.topic-card {
  background: #fff;
  border-radius: 12px;
  padding: 20px;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.05);
  cursor: pointer;
  transition: all 0.2s;
  border: 2px solid transparent;
}

.topic-card:hover {
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.08);
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
  background: #f1f5f9;
  padding: 4px 8px;
  border-radius: 4px;
}

.document-panel {
  background: #fff;
  border-radius: 12px;
  padding: 20px;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.05);
}

.empty-documents {
  text-align: center;
  padding: 20px;
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
  padding: 12px 16px;
  background: #f8fafc;
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

@media (max-width: 768px) {
  .tabs-container {
    padding: 0 16px 12px;
  }

  .tab-btn {
    padding: 8px 16px;
    font-size: 13px;
  }

  .content-container {
    padding: 0 16px 16px;
  }

  .scope-panel {
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
}
</style>
