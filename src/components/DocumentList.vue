<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { ElTable, ElTableColumn, ElButton, ElPagination, ElInput, ElMessageBox, ElMessage } from 'element-plus'
import { Delete, View, Refresh, Upload } from '@element-plus/icons-vue'
import { useDocumentStore } from '@/stores/document'
import FileUploader from './FileUploader.vue'

const documentStore = useDocumentStore()
const searchKeyword = ref('')
const showUploadDialog = ref(false)

onMounted(async () => {
  await documentStore.fetchDocuments()
})

async function handleSearch() {
  await documentStore.fetchDocuments({
    keyword: searchKeyword.value,
    pageNo: 1,
  })
}

async function handlePageChange(page: number) {
  await documentStore.fetchDocuments({
    keyword: searchKeyword.value,
    pageNo: page,
    pageSize: documentStore.pageSize,
  })
}

async function handleSizeChange(size: number) {
  await documentStore.fetchDocuments({
    keyword: searchKeyword.value,
    pageNo: 1,
    pageSize: size,
  })
}

async function handleDelete(documentId: number, documentName: string) {
  try {
    await ElMessageBox.confirm(`确定要删除文档 "${documentName}" 吗？`, '确认删除', {
      type: 'warning',
    })
    await documentStore.deleteDocument(documentId)
    ElMessage.success('删除成功')
  } catch {
    // 用户取消删除
  }
}

async function handleRefresh() {
  await documentStore.fetchDocuments({
    keyword: searchKeyword.value,
  })
}

function handleFileUpload(file: File) {
  documentStore.uploadFile(file)
  ElMessage.success(`文件 "${file.name}" 上传成功`)
}

function formatFileSize(size: number) {
  if (size < 1024) return size + ' B'
  if (size < 1024 * 1024) return (size / 1024).toFixed(2) + ' KB'
  return (size / 1024 / 1024).toFixed(2) + ' MB'
}

function getStatusType(status: number) {
  return status === 0 ? 'success' : status === 1 ? 'warning' : 'danger'
}
</script>

<template>
  <div class="document-list">
    <div class="list-header">
      <div class="header-left">
        <h2 class="list-title">文档管理</h2>
      </div>
      <div class="header-right">
        <ElInput
          v-model="searchKeyword"
          placeholder="搜索文档名称..."
          prefix-icon="Search"
          clearable
          @keyup.enter="handleSearch"
          class="search-input"
        />
        <ElButton type="primary" icon="Upload" @click="showUploadDialog = true">
          <Upload :size="18" />
          <span>上传文档</span>
        </ElButton>
        <ElButton type="default" icon="Refresh" @click="handleRefresh">
          <Refresh :size="18" />
        </ElButton>
      </div>
    </div>

    <ElTable :data="documentStore.documents" stripe border class="document-table">
      <ElTableColumn prop="documentName" label="文档名称" min-width="200" />
      <ElTableColumn prop="originalFileName" label="原始文件名" min-width="150" />
      <ElTableColumn prop="fileTypeName" label="文件类型" width="100" />
      <ElTableColumn prop="fileSize" label="文件大小" width="100" :formatter="(row: { fileSize: number }) => formatFileSize(row.fileSize)" />
      <ElTableColumn prop="charCount" label="字符数" width="100" />
      <ElTableColumn prop="tokenCount" label="Token数" width="100" />
      <ElTableColumn prop="knowledgeScopeName" label="知识范围" width="120" />
      <ElTableColumn prop="parseStatusName" label="解析状态" width="100">
        <template #default="scope">
          <span :class="['el-tag', `el-tag--${getStatusType(scope.row.parseStatus)}`]">{{ scope.row.parseStatusName }}</span>
        </template>
      </ElTableColumn>
      <ElTableColumn prop="indexStatusName" label="索引状态" width="100">
        <template #default="scope">
          <span :class="['el-tag', `el-tag--${getStatusType(scope.row.indexStatus)}`]">{{ scope.row.indexStatusName }}</span>
        </template>
      </ElTableColumn>
      <ElTableColumn prop="createTime" label="创建时间" width="160" />
      <ElTableColumn label="操作" width="150" fixed="right">
        <template #default="scope">
          <ElButton type="text" icon="View" @click="() => {}">
            <View :size="16" />
          </ElButton>
          <ElButton type="text" icon="Delete" @click="handleDelete(scope.row.documentId, scope.row.documentName)">
            <Delete :size="16" />
          </ElButton>
        </template>
      </ElTableColumn>
    </ElTable>

    <div class="pagination-container">
      <ElPagination
        :current-page="documentStore.pageNo"
        :page-size="documentStore.pageSize"
        :total="documentStore.total"
        :page-sizes="[10, 20, 50, 100]"
        layout="total, sizes, prev, pager, next, jumper"
        @size-change="handleSizeChange"
        @current-change="handlePageChange"
      />
    </div>

    <div v-if="showUploadDialog" class="upload-modal" @click.self="showUploadDialog = false">
      <div class="modal-content">
        <div class="modal-header">
          <h3>上传文档</h3>
          <button class="close-btn" @click="showUploadDialog = false">×</button>
        </div>
        <div class="modal-body">
          <FileUploader @upload="handleFileUpload" />
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.document-list {
  padding: 24px;
  background: #fff;
  border-radius: 12px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.08);
}

.list-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 24px;
}

.list-title {
  font-size: 20px;
  font-weight: 600;
  color: #1e293b;
}

.header-right {
  display: flex;
  align-items: center;
  gap: 12px;
}

.search-input {
  width: 280px;
}

.document-table {
  width: 100%;
}

.pagination-container {
  display: flex;
  justify-content: flex-end;
  margin-top: 24px;
}

.upload-modal {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.5);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
}

.modal-content {
  width: 600px;
  max-width: 90%;
  background: #fff;
  border-radius: 12px;
  overflow: hidden;
}

.modal-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px 24px;
  border-bottom: 1px solid #e2e8f0;
}

.modal-header h3 {
  font-size: 16px;
  font-weight: 600;
  margin: 0;
}

.close-btn {
  width: 32px;
  height: 32px;
  border: none;
  background: transparent;
  font-size: 24px;
  cursor: pointer;
  color: #64748b;
}

.close-btn:hover {
  color: #1e293b;
}

.modal-body {
  padding: 24px;
}
</style>
