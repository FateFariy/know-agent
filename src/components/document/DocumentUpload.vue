<script setup lang="ts">
import { ref, computed } from 'vue'
import { useDocumentStore } from '@/stores/document'
import { ElMessage } from 'element-plus'
import { Upload, Close, Check, Warning } from '@element-plus/icons-vue'


const store = useDocumentStore()

const isDragging = ref(false)
const fileInput = ref<HTMLInputElement | null>(null)

const allowedTypes = ['application/pdf', 'application/msword', 'application/vnd.openxmlformats-officedocument.wordprocessingml.document', 'text/plain', 'application/vnd.ms-excel', 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet', 'text/markdown', 'text/html']
const maxFileSize = 50 * 1024 * 1024
const uploadFiles = computed(() => store.uploadFiles)

function handleDragOver(e: DragEvent) {
  e.preventDefault()
  isDragging.value = true
}

function handleDragLeave() {
  isDragging.value = false
}

function handleDrop(e: DragEvent) {
  e.preventDefault()
  isDragging.value = false
  const files = e.dataTransfer?.files
  if (files) {
    processFiles(Array.from(files))
  }
}

function handleFileSelect(e: Event) {
  const target = e.target as HTMLInputElement
  const files = target.files
  if (files) {
    processFiles(Array.from(files))
  }
  target.value = ''
}

function processFiles(files: File[]) {
  for (const file of files) {
    validateAndUpload(file)
  }
}

function validateAndUpload(file: File) {
  if (!allowedTypes.includes(file.type)) {
    ElMessage.warning(`文件 ${file.name} 格式不支持，请上传PDF、Word、Excel、TXT、Markdown等格式`)
    return
  }

  if (file.size > maxFileSize) {
    ElMessage.warning(`文件 ${file.name} 超过大小限制(50MB)`)
    return
  }

  store.uploadFile(file)
}

function removeFile(fileName: string) {
  store.removeUploadFile(fileName)
}

function formatFileSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  return `${(bytes / (1024 * 1024)).toFixed(2)} MB`
}

function triggerFileInput() {
  fileInput.value?.click()
}
</script>

<template>
  <div class="upload-section">
    <div class="upload-header">
      <Upload class="header-icon" />
      <h3 class="header-title">上传文档</h3>
    </div>
    
    <div 
      class="upload-area"
      :class="{ 'is-dragging': isDragging }"
      @dragover="handleDragOver"
      @dragleave="handleDragLeave"
      @drop="handleDrop"
      @click="triggerFileInput"
    >
      <Upload class="upload-icon" :size="48" />
      <p class="upload-title">拖拽文件到此处上传</p>
      <p class="upload-hint">或点击选择文件</p>
      <p class="upload-types">支持 PDF、Word、Excel、TXT、Markdown 等格式，单个文件最大 50MB</p>
      <input 
        ref="fileInput" 
        type="file" 
        class="file-input" 
        multiple
        accept=".pdf,.doc,.docx,.txt,.xls,.xlsx,.md,.html"
        @change="handleFileSelect"
      />
    </div>
    
    <div v-if="uploadFiles.length > 0" class="upload-list">
      <h4 class="list-title">上传队列</h4>
      <div class="list-items">
        <div 
          v-for="file in uploadFiles" 
          :key="file.fileName" 
          class="upload-item"
        >
          <div class="item-info">
            <span class="item-name">{{ file.fileName }}</span>
            <span class="item-size">{{ formatFileSize(file.fileSize) }}</span>
          </div>
          
          <div class="item-status">
            <div v-if="file.status === 'uploading'" class="progress-container">
              <div class="progress-bar" :style="{ width: `${file.progress}%` }"></div>
              <span class="progress-text">{{ file.progress }}%</span>
            </div>
            <div v-else-if="file.status === 'success'" class="status-success">
              <Check :size="16" />
              <span>上传成功</span>
            </div>
            <div v-else-if="file.status === 'error'" class="status-error">
              <Warning :size="16" />
              <span>{{ file.errorMessage || '上传失败' }}</span>
            </div>
            <div v-else class="status-pending">
              <span>等待上传</span>
            </div>
          </div>
          
          <button 
            v-if="file.status !== 'success'" 
            class="remove-btn"
            @click.stop="removeFile(file.fileName)"
          >
            <Close :size="14" />
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.upload-section {
  background: #fff;
  border-radius: 12px;
  padding: 24px;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.05);
}

.upload-header {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 20px;
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

.upload-area {
  border: 2px dashed #cbd5e1;
  border-radius: 12px;
  padding: 40px;
  text-align: center;
  cursor: pointer;
  transition: all 0.3s ease;
  background: #fafbfc;
}

.upload-area:hover {
  border-color: #94a3b8;
  background: #f1f5f9;
}

.upload-area.is-dragging {
  border-color: #3b82f6;
  background: #eff6ff;
}

.upload-icon {
  width: 48px;
  height: 48px;
  margin: 0 auto 16px;
  color: #94a3b8;
}

.upload-title {
  font-size: 16px;
  font-weight: 600;
  color: #475569;
  margin: 0 0 8px 0;
}

.upload-hint {
  font-size: 14px;
  color: #64748b;
  margin: 0 0 12px 0;
}

.upload-types {
  font-size: 12px;
  color: #94a3b8;
  margin: 0;
}

.file-input {
  display: none;
}

.upload-list {
  margin-top: 24px;
  padding-top: 20px;
  border-top: 1px solid #e2e8f0;
}

.list-title {
  font-size: 14px;
  font-weight: 600;
  color: #475569;
  margin: 0 0 16px 0;
}

.list-items {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.upload-item {
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 12px 16px;
  background: #f8fafc;
  border-radius: 8px;
}

.item-info {
  flex: 1;
  min-width: 0;
}

.item-name {
  display: block;
  font-size: 14px;
  font-weight: 500;
  color: #1e293b;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.item-size {
  font-size: 12px;
  color: #94a3b8;
}

.item-status {
  display: flex;
  align-items: center;
  gap: 8px;
}

.progress-container {
  display: flex;
  align-items: center;
  gap: 8px;
  width: 150px;
}

.progress-bar {
  flex: 1;
  height: 6px;
  background: #e2e8f0;
  border-radius: 3px;
  overflow: hidden;
}

.progress-bar::after {
  content: '';
  display: block;
  height: 100%;
  background: linear-gradient(90deg, #3b82f6, #06b6d4);
  border-radius: 3px;
  transition: width 0.3s ease;
}

.progress-text {
  font-size: 12px;
  color: #64748b;
  min-width: 40px;
}

.status-success {
  display: flex;
  align-items: center;
  gap: 4px;
  color: #16a34a;
  font-size: 13px;
}

.status-error {
  display: flex;
  align-items: center;
  gap: 4px;
  color: #dc2626;
  font-size: 13px;
}

.status-pending {
  color: #94a3b8;
  font-size: 13px;
}

.remove-btn {
  background: transparent;
  border: none;
  color: #94a3b8;
  cursor: pointer;
  padding: 6px;
  border-radius: 4px;
  transition: all 0.2s;
}

.remove-btn:hover {
  background: #fee2e2;
  color: #dc2626;
}

@media (max-width: 768px) {
  .upload-area {
    padding: 24px 16px;
  }
  
  .upload-icon {
    width: 36px;
    height: 36px;
    margin-bottom: 12px;
  }
  
  .upload-title {
    font-size: 14px;
  }
  
  .upload-hint,
  .upload-types {
    font-size: 12px;
  }
  
  .progress-container {
    width: 100px;
  }
}
</style>