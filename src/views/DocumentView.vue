<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useDocumentStore } from '@/stores/document'
import DocumentUpload from '@/components/document/DocumentUpload.vue'
import DocumentList from '@/components/document/DocumentList.vue'

const store = useDocumentStore()

const activeTab = ref<'upload' | 'list'>('upload')

async function fetchDocuments() {
  await store.fetchDocuments()
}

function handleViewDocument(documentId: number) {
  console.log('View document:', documentId)
}

onMounted(() => {
  fetchDocuments()
})
</script>

<template>
  <div class="document-view">
    <div class="tabs-container">
      <button 
        class="tab-btn" 
        :class="{ active: activeTab === 'upload' }"
        @click="activeTab = 'upload'"
      >
        上传文档
      </button>
      <button 
        class="tab-btn" 
        :class="{ active: activeTab === 'list' }"
        @click="activeTab = 'list'; fetchDocuments()"
      >
        文档列表
      </button>
    </div>
    
    <div class="content-container">
      <DocumentUpload v-if="activeTab === 'upload'" />
      <DocumentList v-else @view="handleViewDocument" />
    </div>
  </div>
</template>

<style scoped>
.document-view {
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
}
</style>