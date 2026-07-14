<script setup lang="ts">
import { ref } from 'vue'
import { ElUpload, ElButton, ElProgress, ElMessage } from 'element-plus'
import { Upload, Close, Check, WarnTriangleFilled } from '@element-plus/icons-vue'

const emit = defineEmits<{
  (e: 'upload', file: File): void
}>()

const files = ref<{
  name: string
  size: number
  progress: number
  status: 'pending' | 'uploading' | 'success' | 'error'
  error?: string
}[]>([])

const MAX_FILE_SIZE = 50 * 1024 * 1024
const ALLOWED_EXTENSIONS = ['.pdf', '.doc', '.docx', '.txt', '.md', '.json', '.csv', '.xlsx', '.xls']

function checkFile(file: File) {
  const ext = file.name.slice(file.name.lastIndexOf('.')).toLowerCase()
  if (!ALLOWED_EXTENSIONS.includes(ext)) {
    ElMessage.error(`不支持的文件格式: ${ext}，支持的格式: ${ALLOWED_EXTENSIONS.join(', ')}`)
    return false
  }
  if (file.size > MAX_FILE_SIZE) {
    ElMessage.error('文件大小不能超过50MB')
    return false
  }
  return true
}

function handleFileChange(file: File) {
  if (!checkFile(file)) return

  const existingIndex = files.value.findIndex((f) => f.name === file.name)
  if (existingIndex >= 0) {
    ElMessage.warning('该文件已在上传队列中')
    return
  }

  const uploadFile: typeof files.value[0] = {
    name: file.name,
    size: file.size,
    progress: 0,
    status: 'pending',
  }
  files.value.push(uploadFile)

  uploadFile.status = 'uploading'
  simulateUpload(uploadFile, file)
}

function simulateUpload(uploadFile: typeof files.value[0], file: File) {
  let progress = 0
  const interval = setInterval(() => {
    progress += Math.random() * 15
    if (progress >= 100) {
      progress = 100
      clearInterval(interval)
      uploadFile.progress = progress
      uploadFile.status = 'success'
      emit('upload', file)
      setTimeout(() => {
        files.value = files.value.filter((f) => f.name !== uploadFile.name)
      }, 2000)
    } else {
      uploadFile.progress = Math.floor(progress)
    }
  }, 200)
}

function handleRemove(index: number) {
  files.value.splice(index, 1)
}

function handleClick() {
  ;(document.querySelector('.el-upload__input') as HTMLInputElement)?.click()
}
</script>

<template>
  <div class="file-uploader">
    <ElUpload
      class="upload-btn"
      :show-file-list="false"
      :before-upload="(f) => false"
      @change="(uploadFile) => {
        if (uploadFile.raw) handleFileChange(uploadFile.raw)
      }"
      multiple
    >
      <ElButton type="primary" size="large" icon="Upload" @click="handleClick">
        <Upload :size="20" />
        <span>选择文件上传</span>
      </ElButton>
    </ElUpload>

    <div class="upload-info">
      <span class="info-text">支持格式: PDF, DOC, DOCX, TXT, MD, JSON, CSV, XLSX, XLS</span>
      <span class="info-text">单文件最大: 50MB</span>
    </div>

    <div v-if="files.length > 0" class="upload-list">
      <div
        v-for="(file, index) in files"
        :key="file.name"
        class="upload-item"
      >
        <div class="file-info">
          <span class="file-name">{{ file.name }}</span>
          <span class="file-size">{{ (file.size / 1024 / 1024).toFixed(2) }}MB</span>
        </div>

        <div class="file-status">
          <div class="progress-bar">
            <ElProgress
              :percentage="file.progress"
              :status="file.status === 'success' ? 'success' : file.status === 'error' ? 'exception' : 'default'"
              :stroke-width="8"
              :show-text="false"
            />
          </div>

          <div class="status-icons">
            <Check v-if="file.status === 'success'" :size="16" class="success-icon" />
            <WarnTriangleFilled v-if="file.status === 'error'" :size="16" class="error-icon" />
            <span v-if="file.status === 'uploading'" class="progress-text">{{ file.progress }}%</span>
            <ElButton
              v-if="file.status !== 'success'"
              type="text"
              :icon="Close"
              :size="16"
              @click="handleRemove(index)"
            />
          </div>
        </div>

        <div v-if="file.error" class="error-message">
          {{ file.error }}
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.file-uploader {
  padding: 32px;
  background: #fff;
  border-radius: 12px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.08);
}

.upload-btn {
  width: 100%;
  display: flex;
  justify-content: center;
  padding: 40px;
  border: 2px dashed #d9d9d9;
  border-radius: 12px;
  transition: all 0.3s ease;
}

.upload-btn:hover {
  border-color: #3b82f6;
  background: #f0f9ff;
}

.upload-btn .el-button {
  width: 100%;
}

.upload-info {
  display: flex;
  justify-content: center;
  gap: 24px;
  margin-top: 16px;
}

.info-text {
  font-size: 12px;
  color: #94a3b8;
}

.upload-list {
  margin-top: 24px;
}

.upload-item {
  padding: 16px;
  background: #f8fafc;
  border-radius: 8px;
  margin-bottom: 12px;
}

.file-info {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;
}

.file-name {
  font-size: 14px;
  font-weight: 500;
  color: #1e293b;
}

.file-size {
  font-size: 12px;
  color: #64748b;
}

.file-status {
  display: flex;
  align-items: center;
  gap: 12px;
}

.progress-bar {
  flex: 1;
}

.status-icons {
  display: flex;
  align-items: center;
  gap: 8px;
}

.progress-text {
  font-size: 12px;
  color: #3b82f6;
  min-width: 36px;
}

.success-icon {
  color: #22c55e;
}

.error-icon {
  color: #ef4444;
}

.error-message {
  margin-top: 8px;
  font-size: 12px;
  color: #ef4444;
}
</style>
