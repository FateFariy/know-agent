<script lang="ts" setup>
/**
 * MarkdownRenderer：基于 markdown-it + highlight.js 渲染助手回复。
 * - 行内 code 走浅灰胶囊样式；
 * - 多行 code 走带语言标签的代码块 + 复制按钮。
 * - 把 [数字] 形式的引用标记改写成 .md-reference-btn 按钮，方便抽屉组件展示。
 * 复刻 React MarkdownRenderer 的视觉与交互。
 *
 * 与 MarkdownView 的差别：
 * - 这里保留 [1]/[2] 引用按钮改写，事件通过 reference-click 抛出；
 * - 渲染管线与代码块增强复用 useMarkdown 中的 createMarkdownRenderer / enrichCodeBlocks；
 * - 复制代码时通过 ElMessage 给出反馈。
 */
import {computed, ref, watch} from 'vue'
import {createMarkdownRenderer, enrichCodeBlocks} from '@/composables/useMarkdown'
import {ElMessage} from 'element-plus'

const props = defineProps<{
  content: string
}>()

const emit = defineEmits<{
  (e: 'reference-click', index: number): void
}>()

const containerRef = ref<HTMLElement | null>(null)

const md = createMarkdownRenderer({allowHtml: false})

// 链接渲染：补充 target=_blank / rel=noreferrer 已在 createMarkdownRenderer 中完成。

// 将 [数字] 形式的引用标记改写为可点击按钮；负向先行 (?!\() 避免误匹配 markdown 链接。
const renderedHtml = computed(() => {
  if (!props.content) return ''
  const html = md.render(props.content)
  return html.replace(/\[(\d+)\](?!\()/g, '<span class="md-reference-btn" data-index="$1">$&</span>')
})

function handleReferenceClick(index: number) {
  emit('reference-click', index)
}

function handleContainerClick(event: MouseEvent) {
  const target = event.target as HTMLElement
  const btn = target.closest('.md-reference-btn') as HTMLElement | null
  if (btn) {
    const index = parseInt(btn.dataset.index || '0', 10)
    handleReferenceClick(index)
  } else {
    const textContent = target.textContent || ''
    const match = textContent.match(/\[(\d+)\]/)
    if (match) {
      const index = parseInt(match[1] || '0', 10)
      handleReferenceClick(index)
    }
  }
}

async function copyCode(value: string) {
  try {
    await navigator.clipboard.writeText(value)
    ElMessage.success('已复制代码')
  } catch {
    ElMessage.error('复制失败')
  }
}

async function runEnrichment() {
  enrichCodeBlocks(containerRef.value, {onCopy: copyCode})
}

watch(
  () => props.content,
  () => {
    runEnrichment()
  },
  {immediate: true}
)
</script>

<template>
  <div ref="containerRef" class="markdown-body" @click="handleContainerClick">
    <!-- eslint-disable-next-line vue/no-v-html -->
    <div v-html="renderedHtml" />
  </div>
</template>

<style scoped>
.markdown-body {
  color: #1a1a1a;
  font-size: 15px;
  line-height: 1.7;
  word-break: break-word;
}

.markdown-body :deep(p) {
  margin: 0 0 10px;
  color: #333333;
}

.markdown-body :deep(p:last-child) {
  margin-bottom: 0;
}

.markdown-body :deep(strong) {
  color: #1a1a1a;
  font-weight: 600;
}

.markdown-body :deep(h1),
.markdown-body :deep(h2),
.markdown-body :deep(h3),
.markdown-body :deep(h4),
.markdown-body :deep(h5),
.markdown-body :deep(h6) {
  color: #1a1a1a;
  font-weight: 600;
  margin: 1.2em 0 0.6em;
  line-height: 1.4;
}

.markdown-body :deep(ul),
.markdown-body :deep(ol) {
  margin: 8px 0 12px;
  padding-left: 24px;
}

.markdown-body :deep(li) {
  color: #333333;
  margin: 4px 0;
}

.markdown-body :deep(a) {
  color: #2563eb;
  text-decoration: none;
  text-underline-offset: 3px;
}

.markdown-body :deep(a:hover) {
  text-decoration: underline;
}

.markdown-body :deep(blockquote) {
  margin: 12px 0;
  padding: 8px 14px;
  border-left: 3px solid #3b82f6;
  background: #f0f7ff;
  color: #333333;
  font-style: italic;
  border-radius: 0 6px 6px 0;
}

.markdown-body :deep(table) {
  width: 100%;
  border-collapse: collapse;
  border: 1px solid #d0d7de;
  border-radius: 6px;
  margin: 12px 0;
  font-size: 14px;
}

.markdown-body :deep(thead) {
  background: #f6f8fa;
}

.markdown-body :deep(th),
.markdown-body :deep(td) {
  border-bottom: 1px solid #d0d7de;
  border-right: 1px solid #d0d7de;
  padding: 8px 12px;
  text-align: left;
  color: #1a1a1a;
}

.markdown-body :deep(th:last-child),
.markdown-body :deep(td:last-child) {
  border-right: none;
}

.markdown-body :deep(img) {
  max-width: 100%;
  border-radius: 8px;
  margin: 12px 0;
}

.markdown-body :deep(code) {
  font-family: 'SFMono-Regular', Consolas, 'Liberation Mono', Menlo, monospace;
  font-size: 13px;
  padding: 2px 6px;
  border-radius: 4px;
  background: #f6f8fa;
  color: #24292f;
}

.markdown-body :deep(pre) {
  margin: 0;
  padding: 12px 16px;
  background: transparent;
  overflow-x: auto;
  font-size: 13px;
  line-height: 1.5;
}

.markdown-body :deep(.md-code-block) {
  margin: 12px 0;
  overflow: hidden;
  border-radius: 8px;
  border: 1px solid #d0d7de;
  background: #f6f8fa;
}

.markdown-body :deep(.md-code-header) {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 6px 12px;
  border-bottom: 1px solid #d0d7de;
  background: #f6f8fa;
}

.markdown-body :deep(.md-code-lang) {
  font-family: 'SFMono-Regular', Consolas, monospace;
  font-size: 11px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  color: #57606a;
}

.markdown-body :deep(.md-code-copy) {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  border: 0;
  border-radius: 4px;
  background: transparent;
  color: #57606a;
  cursor: pointer;
  transition: background 0.15s ease;
}

.markdown-body :deep(.md-code-copy:hover) {
  background: #eaeef2;
}

.markdown-body :deep(.md-code-copy__icon) {
  display: inline-flex;
  align-items: center;
  justify-content: center;
}

.markdown-body :deep(.md-reference-btn) {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 20px;
  height: 20px;
  padding: 0 5px;
  border-radius: 4px;
  background: #e0f2fe;
  color: #0369a1;
  font-size: 12px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.15s ease;
  user-select: none;
}

.markdown-body :deep(.md-reference-btn:hover) {
  background: #bae6fd;
  transform: translateY(-1px);
  box-shadow: 0 1px 2px rgba(3, 105, 161, 0.2);
}

.markdown-body :deep(.md-reference-btn:active) {
  transform: translateY(0);
  box-shadow: none;
}
</style>
