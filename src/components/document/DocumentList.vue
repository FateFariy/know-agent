<script setup lang="ts">
import { ref, computed } from 'vue'
import { useDocumentStore } from '@/stores/document'
import { ElMessage, ElPagination } from 'element-plus'
import { Document, Delete, View, Search, Clock, Picture, Document as DocIcon } from '@element-plus/icons-vue'


const emit = defineEmits<{
  (e: 'view', documentId: number): void
}>()

const store = useDocumentStore()

const searchKeyword = ref('')
const currentPage = ref(1)
const pageSize = ref(10)

const documents = computed(() => {
  let docs = store.documents
  if (searchKeyword.value) {
    const keyword = searchKeyword.value.toLowerCase()
    docs = docs.filter(doc =>
      doc.documentName.toLowerCase().includes(keyword) ||
      doc.originalFileName.toLowerCase().includes(keyword)
    )
  }
  return docs
})

const total = computed(() => store.total)

function handleView(documentId: number) {
  emit('view', documentId)
}

async function handleDelete(documentId: number, documentName: string) {
  if (!confirm(`确定删除文档 "${documentName}" 吗？`)) {
    return
  }
  try {
    await store.deleteDocument(documentId)
    ElMessage.success('文档已删除')
  } catch {
    ElMessage.error('删除失败')
  }
}

function formatFileSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  return `${(bytes / (1024 * 1024)).toFixed(2)} MB`
}

function formatDate(dateStr: string): string {
  if (!dateStr) return ''
  const date = new Date(dateStr)
  return date.toLocaleDateString('zh-CN')
}

function getFileIcon(fileType: number) {
  switch (fileType) {
    case 1: return Document
    case 2: return DocIcon
    case 3: return Picture
    default: return Document
  }
}

function getStatusColor(status: number): string {
  switch (status) {
    case 0: return 'status-success'
    case 1: return 'status-processing'
    case -1: return 'status-error'
    default: return 'status-default'
  }
}

async function handlePageChange(page: number) {
  currentPage.value = page
  await store.fetchDocuments({ pageNo: page, pageSize: pageSize.value })
}

async function handlePageSizeChange(size: number) {
  pageSize.value = size
  currentPage.value = 1
  await store.fetchDocuments({ pageNo: 1, pageSize: size })
}

async function handleSearch() {
  currentPage.value = 1
  await store.fetchDocuments({ keyword: searchKeyword.value, pageNo: 1, pageSize: pageSize.value })
}
</script>

<template>
  <div class="document-list">
    <div class="list-header">
      <div class="header-left">
        <FileText class="header-icon" />
        <h3 class="header-title">文档列表</h3>
      </div>
      <div class="search-box">
        <Search class="search-icon" />
        <input v-model="searchKeyword" type="text" placeholder="搜索文档名称..." class="search-input"
          @keyup.enter="handleSearch" />
        <button class="search-btn" @click="handleSearch">搜索</button>
      </div>
    </div>

    <div v-if="documents.length === 0" class="empty-state">
      <FileText class="empty-icon" />
      <p>暂无文档</p>
      <p class="empty-hint">上传文档后，智能问答将基于这些文档进行回答</p>
    </div>

    <div v-else class="list-container">
      <table class="document-table">
        <thead>
          <tr>
            <th>文档名称</th>
            <th>文件类型</th>
            <th>文件大小</th>
            <th>解析状态</th>
            <th>索引状态</th>
            <th>创建时间</th>
            <th>操作</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="doc in documents" :key="doc.documentId">
            <td class="doc-name-cell">
              <component :is="getFileIcon(doc.fileType)" class="file-icon" />
              <span class="doc-name">{{ doc.documentName }}</span>
            </td>
            <td>{{ doc.fileTypeName }}</td>
            <td>{{ formatFileSize(doc.fileSize) }}</td>
            <td>
              <span :class="['status-badge', getStatusColor(doc.parseStatus)]">
                {{ doc.parseStatusName }}
              </span>
            </td>
            <td>
              <span :class="['status-badge', getStatusColor(doc.indexStatus)]">
                {{ doc.indexStatusName }}
              </span>
            </td>
            <td>
              <Clock class="time-icon" />
              {{ formatDate(doc.createTime) }}
            </td>
            <td class="actions-cell">
              <button class="action-btn view-btn" @click="handleView(doc.documentId)">
                <View :size="14" />
                <span>查看</span>
              </button>
              <button class="action-btn delete-btn" @click="handleDelete(doc.documentId, doc.documentName)">
                <Delete :size="14" />
                <span>删除</span>
              </button>
            </td>
          </tr>
        </tbody>
      </table>

      <div class="pagination-container">
        <ElPagination :current-page="currentPage" :page-size="pageSize" :total="total" :page-sizes="[10, 20, 50]"
          layout="total, sizes, prev, pager, next" @size-change="handlePageSizeChange"
          @current-change="handlePageChange" />
      </div>
    </div>
  </div>
</template>

<style scoped>
.document-list {
  background: #fff;
  border-radius: 12px;
  padding: 24px;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.05);
}

.list-header {
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

.search-box {
  display: flex;
  align-items: center;
  gap: 8px;
}

.search-icon {
  width: 16px;
  height: 16px;
  color: #94a3b8;
  position: absolute;
  left: 12px;
}

.search-input {
  position: relative;
  padding: 8px 12px 8px 32px;
  border: 1px solid #e2e8f0;
  border-radius: 8px;
  font-size: 14px;
  outline: none;
  transition: border-color 0.2s;
  width: 200px;
}

.search-input:focus {
  border-color: #3b82f6;
}

.search-btn {
  padding: 8px 16px;
  background: #3b82f6;
  border: none;
  border-radius: 8px;
  font-size: 14px;
  color: #fff;
  cursor: pointer;
  transition: background 0.2s;
}

.search-btn:hover {
  background: #2563eb;
}

.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 60px 20px;
  color: #94a3b8;
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

.list-container {
  overflow-x: auto;
}

.document-table {
  width: 100%;
  border-collapse: collapse;
}

.document-table th,
.document-table td {
  padding: 12px 16px;
  text-align: left;
  border-bottom: 1px solid #f1f5f9;
}

.document-table th {
  background: #f8fafc;
  font-size: 14px;
  font-weight: 600;
  color: #475569;
}

.document-table td {
  font-size: 14px;
  color: #334155;
}

.doc-name-cell {
  display: flex;
  align-items: center;
  gap: 8px;
}

.file-icon {
  width: 18px;
  height: 18px;
  color: #64748b;
}

.doc-name {
  font-weight: 500;
}

.status-badge {
  display: inline-block;
  padding: 4px 10px;
  border-radius: 20px;
  font-size: 12px;
}

.status-success {
  background: #dcfce7;
  color: #16a34a;
}

.status-processing {
  background: #fef3c7;
  color: #d97706;
}

.status-error {
  background: #fee2e2;
  color: #dc2626;
}

.status-default {
  background: #f1f5f9;
  color: #64748b;
}

.time-icon {
  width: 14px;
  height: 14px;
  color: #94a3b8;
  margin-right: 4px;
}

.actions-cell {
  display: flex;
  gap: 8px;
}

.action-btn {
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 6px 12px;
  border: none;
  border-radius: 6px;
  font-size: 13px;
  cursor: pointer;
  transition: all 0.2s;
}

.view-btn {
  background: #eff6ff;
  color: #3b82f6;
}

.view-btn:hover {
  background: #dbeafe;
}

.delete-btn {
  background: #fef2f2;
  color: #dc2626;
}

.delete-btn:hover {
  background: #fee2e2;
}

.pagination-container {
  display: flex;
  justify-content: flex-end;
  margin-top: 20px;
}

@media (max-width: 768px) {
  .list-header {
    flex-direction: column;
    gap: 16px;
    align-items: stretch;
  }

  .search-box {
    width: 100%;
  }

  .search-input {
    flex: 1;
    width: auto;
  }

  .document-table {
    font-size: 12px;
  }

  .document-table th,
  .document-table td {
    padding: 8px 10px;
  }

  .actions-cell {
    flex-direction: column;
  }

  .action-btn span {
    display: none;
  }

  .action-btn {
    padding: 6px;
  }

  .pagination-container {
    justify-content: center;
  }
}

@media (max-width: 480px) {
  .document-table {
    min-width: 600px;
  }
}
</style>
