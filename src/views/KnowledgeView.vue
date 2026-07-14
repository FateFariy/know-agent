<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { ElTree, ElTable, ElTableColumn, ElButton, ElInput, ElTag } from 'element-plus'
import { Plus, FolderOpened, Document } from '@element-plus/icons-vue'
import { knowledgeApi } from '@/api/knowledge'
import type { KnowledgeScope, KnowledgeTopic, TopicDocumentRelation } from '@/types'

interface TreeData {
  id: string
  label: string
  type: 'scope' | 'topic'
  scopeCode: string
  topicCode?: string
  children?: TreeData[]
}

const scopes = ref<KnowledgeScope[]>([])
const topics = ref<KnowledgeTopic[]>([])
const documents = ref<TopicDocumentRelation[]>([])
const selectedScope = ref<string | null>(null)
const selectedTopic = ref<string | null>(null)
const searchKeyword = ref('')
const expandedKeys = ref<string[]>([])

onMounted(async () => {
  await loadScopes()
})

async function loadScopes() {
  scopes.value = await knowledgeApi.listScopes()
  expandedKeys.value = scopes.value.map((s) => s.scopeCode)
}

async function handleScopeSelect(data: TreeData) {
  if (!data) return
  selectedScope.value = data.scopeCode
  selectedTopic.value = null
  documents.value = []
  if (data.type === 'scope') {
    topics.value = await knowledgeApi.listTopics({ scopeCode: data.scopeCode })
  }
}

async function handleTopicSelect(topicCode: string) {
  selectedTopic.value = topicCode
  documents.value = await knowledgeApi.listTopicDocuments({ topicCode })
}

function buildTreeData(): TreeData[] {
  const tree: TreeData[] = []
  scopes.value.forEach((scope) => {
    const scopeNode: TreeData = {
      id: scope.scopeCode,
      label: scope.scopeName,
      type: 'scope',
      scopeCode: scope.scopeCode,
      children: [],
    }
    const scopeTopics = topics.value.filter((t) => t.scopeCode === scope.scopeCode)
    scopeTopics.forEach((topic) => {
      scopeNode.children?.push({
        id: topic.topicCode,
        label: topic.topicName,
        type: 'topic',
        topicCode: topic.topicCode,
        scopeCode: scope.scopeCode,
      })
    })
    tree.push(scopeNode)
  })
  return tree
}

function getTopicByCode(topicCode: string) {
  return topics.value.find((t) => t.topicCode === topicCode)
}
</script>

<template>
  <div class="knowledge-view">
    <div class="view-header">
      <h2 class="view-title">知识库管理</h2>
      <div class="header-actions">
        <ElInput
          v-model="searchKeyword"
          placeholder="搜索知识主题..."
          prefix-icon="Search"
          clearable
          class="search-input"
        />
        <ElButton type="primary" icon="Plus">
          <Plus :size="18" />
          <span>新建主题</span>
        </ElButton>
      </div>
    </div>

    <div class="knowledge-content">
      <div class="sidebar">
        <div class="sidebar-header">
          <FolderOpened :size="18" />
          <span>知识范围</span>
        </div>
        <ElTree
          :data="buildTreeData()"
          :props="{ label: 'label', children: 'children' }"
          :expand-on-click-node="false"
          :default-expanded-keys="expandedKeys"
          highlight-current
          node-key="id"
          @node-click="handleScopeSelect"
          class="tree"
        />
      </div>

      <div class="main-panel">
        <div v-if="!selectedScope" class="empty-panel">
          <FolderOpened :size="48" class="empty-icon" />
          <p>选择一个知识范围查看详情</p>
        </div>

        <div v-else-if="!selectedTopic" class="topics-panel">
          <div class="panel-header">
            <h3>{{ scopes.find((s) => s.scopeCode === selectedScope)?.scopeName }}</h3>
            <ElButton type="primary" icon="Plus" size="small">
              <Plus :size="16" />
              <span>新建主题</span>
            </ElButton>
          </div>

          <div v-if="topics.length === 0" class="empty-topics">
            <p>暂无知识主题，点击上方按钮创建</p>
          </div>

          <div v-else class="topics-grid">
            <div
              v-for="topic in topics"
              :key="topic.topicCode"
              :class="['topic-card', { selected: selectedTopic === topic.topicCode }]"
              @click="handleTopicSelect(topic.topicCode)"
            >
              <Document :size="24" class="topic-icon" />
              <div class="topic-info">
                <h4>{{ topic.topicName }}</h4>
                <p>{{ topic.description || '暂无描述' }}</p>
              </div>
              <div class="topic-tags">
                <ElTag v-if="topic.answerShape" size="small">{{ topic.answerShape }}</ElTag>
              </div>
            </div>
          </div>
        </div>

        <div v-else class="documents-panel">
          <div class="panel-header">
            <h3>{{ getTopicByCode(selectedTopic)?.topicName }}</h3>
            <ElButton type="primary" icon="Plus" size="small">
              <Plus :size="16" />
              <span>关联文档</span>
            </ElButton>
          </div>

          <div v-if="documents.length === 0" class="empty-documents">
            <p>暂无关联文档</p>
          </div>

          <ElTable v-else :data="documents" stripe border class="documents-table">
            <ElTableColumn prop="documentName" label="文档名称" min-width="200" />
            <ElTableColumn prop="knowledgeScopeName" label="知识范围" width="120" />
            <ElTableColumn prop="businessCategory" label="业务分类" width="120" />
            <ElTableColumn prop="documentTags" label="标签" width="150" />
            <ElTableColumn
              prop="relationScore"
              label="关联分数"
              width="100"
              :formatter="(row: { relationScore: number }) => (row.relationScore * 100).toFixed(1) + '%'"
            />
            <ElTableColumn prop="relationSource" label="关联来源" width="120" />
            <ElTableColumn prop="reason" label="关联理由" min-width="200" />
          </ElTable>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.knowledge-view {
  height: 100%;
  padding: 24px;
  display: flex;
  flex-direction: column;
}

.view-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 24px;
}

.view-title {
  font-size: 20px;
  font-weight: 600;
  color: #1e293b;
  margin: 0;
}

.header-actions {
  display: flex;
  align-items: center;
  gap: 12px;
}

.search-input {
  width: 280px;
}

.knowledge-content {
  flex: 1;
  display: flex;
  gap: 24px;
  overflow: hidden;
}

.sidebar {
  width: 280px;
  background: #fff;
  border-radius: 12px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.08);
  overflow-y: auto;
}

.sidebar-header {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 16px;
  border-bottom: 1px solid #e2e8f0;
  font-size: 14px;
  font-weight: 600;
  color: #334155;
}

.tree {
  padding: 16px;
}

.main-panel {
  flex: 1;
  background: #fff;
  border-radius: 12px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.08);
  overflow-y: auto;
  padding: 24px;
}

.empty-panel {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  height: 100%;
  color: #94a3b8;
}

.empty-icon {
  opacity: 0.5;
  margin-bottom: 16px;
}

.panel-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 24px;
}

.panel-header h3 {
  font-size: 16px;
  font-weight: 600;
  color: #1e293b;
  margin: 0;
}

.empty-topics,
.empty-documents {
  text-align: center;
  padding: 40px;
  color: #94a3b8;
}

.topics-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
  gap: 16px;
}

.topic-card {
  padding: 20px;
  border: 2px solid #e2e8f0;
  border-radius: 12px;
  cursor: pointer;
  transition: all 0.3s ease;
}

.topic-card:hover {
  border-color: #94a3b8;
}

.topic-card.selected {
  border-color: #3b82f6;
  background: #f0f9ff;
}

.topic-icon {
  color: #3b82f6;
  margin-bottom: 12px;
}

.topic-info h4 {
  font-size: 14px;
  font-weight: 600;
  color: #1e293b;
  margin: 0 0 8px 0;
}

.topic-info p {
  font-size: 13px;
  color: #64748b;
  margin: 0;
  line-height: 1.5;
}

.topic-tags {
  margin-top: 12px;
}

.documents-table {
  width: 100%;
}
</style>
