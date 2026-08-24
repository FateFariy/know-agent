import { computed, ref } from 'vue'

/**
 * 知识库上下文。
 *
 * 后端最新 API 已将知识范围 / 主题 / 关联 / 文档上传全部改为按 knowledgeBaseId 维度隔离，
 * 但当前 api 定义中尚未提供“知识库列表”查询接口，因此这里以模块级单例维护当前知识库ID，
 * 默认取环境变量 VITE_DEFAULT_KNOWLEDGE_BASE_ID（缺省为 '1'，即库内默认知识库）。
 *
 * 后续后端补充 KnowledgeBaseOptionResp 列表接口后，只需在此处接入即可，
 * 上层业务代码无需改动。
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

export function useKnowledgeBase() {
  function setKnowledgeBaseId(id: string): void {
    const normalized = String(id || '').trim() || DEFAULT_KNOWLEDGE_BASE_ID
    currentKnowledgeBaseId.value = normalized
    try {
      window.localStorage.setItem(STORAGE_KEY, normalized)
    } catch {
      // 忽略持久化失败（隐私模式等）
    }
  }

  return {
    knowledgeBaseId: computed<string>(() => currentKnowledgeBaseId.value),
    setKnowledgeBaseId,
    DEFAULT_KNOWLEDGE_BASE_ID
  }
}
