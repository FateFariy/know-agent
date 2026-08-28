import { computed, ref } from 'vue'
import { knowledgeApi } from '@/api/knowledge'
import type { KnowledgeBaseItemResp } from '@/types'

/**
 * 知识库上下文。
 *
 * 后端最新 API 已将知识范围 / 主题 / 关联 / 文档上传全部改为按 knowledgeBaseId 维度隔离。
 * 此 composable 负责：
 * 1. 维护当前选中的知识库 ID（持久化到 localStorage）
 * 2. 加载并缓存可用的知识库列表
 * 3. 提供切换知识库的方法
 */
const DEFAULT_KNOWLEDGE_BASE_ID = String(
  import.meta.env.VITE_DEFAULT_KNOWLEDGE_BASE_ID || '1'
)

const STORAGE_KEY = 'know-agent:knowledgeBaseId'

function readPersisted(): string {
  try {
    return window.localStorage.getItem(STORAGE_KEY) || DEFAULT_KNOWLEDGE_BASE_ID
  } catch {
    return DEFAULT_KNOWLEDGE_BASE_ID
  }
}

const currentKnowledgeBaseId = ref<string>(readPersisted())
const knowledgeBaseList = ref<KnowledgeBaseItemResp[]>([])
const loading = ref<boolean>(false)
const loaded = ref<boolean>(false)

export function useKnowledgeBase() {
  /** 当前选中的知识库 */
  const knowledgeBaseId = computed<string>(() => currentKnowledgeBaseId.value)

  /** 当前选中的知识库详情 */
  const currentKnowledgeBase = computed<KnowledgeBaseItemResp | null>(() => {
    return knowledgeBaseList.value.find((item) => item.id === currentKnowledgeBaseId.value) || null
  })

  /** 设置当前知识库 ID */
  function setKnowledgeBaseId(id: string): void {
    const normalized = String(id || '').trim() || DEFAULT_KNOWLEDGE_BASE_ID
    currentKnowledgeBaseId.value = normalized
    try {
      window.localStorage.setItem(STORAGE_KEY, normalized)
    } catch {
      // 忽略持久化失败（隐私模式等）
    }
  }

  /** 加载知识库列表 */
  async function loadKnowledgeBases(force: boolean = false): Promise<void> {
    if (loaded.value && !force) {
      return
    }
    loading.value = true
    try {
      const { data } = await knowledgeApi.listKnowledgeBases()
      knowledgeBaseList.value = data || []
      loaded.value = true

      // 如果当前选中的 ID 不在列表中，则选择第一个或默认
      if (knowledgeBaseList.value.length > 0) {
        const exists = knowledgeBaseList.value.some((item) => item.id === currentKnowledgeBaseId.value)
        if (!exists) {
          const defaultItem = knowledgeBaseList.value.find((item) => item.isDefault === 1)
          setKnowledgeBaseId(defaultItem?.id || knowledgeBaseList?.value?.[0]?.id || DEFAULT_KNOWLEDGE_BASE_ID)
        }
      }
    } catch (error) {
      console.error('加载知识库列表失败', error)
      knowledgeBaseList.value = []
    } finally {
      loading.value = false
    }
  }

  return {
    knowledgeBaseId,
    currentKnowledgeBase,
    knowledgeBaseList,
    loading,
    loaded,
    setKnowledgeBaseId,
    loadKnowledgeBases,
    DEFAULT_KNOWLEDGE_BASE_ID
  }
}
