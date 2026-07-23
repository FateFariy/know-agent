<script lang="ts" setup>
/**
 * MarkdownView：通用 Markdown 安全渲染组件。
 * - 默认禁用 raw HTML，规避 XSS 风险；
 * - 支持标题、列表、链接、表格、行内/块级代码、引用、加粗、斜体等常见语法；
 * - 代码块带语言标签 + 复制按钮；
 * - 可选 size 模式（compact / normal）以适配不同区域。
 *
 * 与 chat 专用 MarkdownRenderer 的差别：
 * - 不再做 [1]/[2] 引用按钮改写；
 * - 默认 html: false，输入内容中的 <script> 等会被转义而不是被解析。
 */
import {computed, nextTick, ref, watch} from 'vue'
import {createMarkdownRenderer, enrichCodeBlocks} from '@/composables/useMarkdown'

const props = withDefaults(defineProps<{
  content: string
  /**
   * 尺寸模式：normal 用于正文，compact 用于卡片/摘要等密集排版。
   */
  size?: 'normal' | 'compact'
  /**
   * 是否允许原始 HTML；默认 false 防止 XSS。
   * 仅在完全信任数据源时打开。
   */
  allowHtml?: boolean
  /**
   * 是否启用代码块语言标签 + 复制按钮。
   */
  enrichCode?: boolean
}>(), {
  size: 'normal',
  allowHtml: false,
  enrichCode: true
})

const containerRef = ref<HTMLElement | null>(null)

const md = createMarkdownRenderer({allowHtml: props.allowHtml})

const renderedHtml = computed(() => {
  if (!props.content) return ''
  return md.render(props.content)
})

async function runEnrichment() {
  await nextTick()
  if (!props.enrichCode) return
  enrichCodeBlocks(containerRef.value)
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
  <div
    ref="containerRef"
    :class="['markdown-view', `markdown-view--${size}`]"
  >
    <!-- eslint-disable-next-line vue/no-v-html -->
    <div v-html="renderedHtml" />
  </div>
</template>

<style scoped>
.markdown-view {
  color: #1a1a1a;
  font-size: 15px;
  line-height: 1.7;
  word-break: break-word;
}

.markdown-view--compact {
  font-size: 14px;
  line-height: 1.65;
}

.markdown-view :deep(p) {
  margin: 0 0 10px;
  color: #333333;
}

.markdown-view :deep(p:last-child) {
  margin-bottom: 0;
}

.markdown-view :deep(strong) {
  color: #1a1a1a;
  font-weight: 600;
}

.markdown-view :deep(h1),
.markdown-view :deep(h2),
.markdown-view :deep(h3),
.markdown-view :deep(h4),
.markdown-view :deep(h5),
.markdown-view :deep(h6) {
  color: #1a1a1a;
  font-weight: 600;
  margin: 1.2em 0 0.6em;
  line-height: 1.4;
}

.markdown-view :deep(h1) {
  font-size: 1.4em;
}

.markdown-view :deep(h2) {
  font-size: 1.25em;
}

.markdown-view :deep(h3) {
  font-size: 1.1em;
}

.markdown-view :deep(h4),
.markdown-view :deep(h5),
.markdown-view :deep(h6) {
  font-size: 1em;
}

.markdown-view :deep(ul),
.markdown-view :deep(ol) {
  margin: 8px 0 12px;
  padding-left: 24px;
}

.markdown-view :deep(li) {
  color: #333333;
  margin: 4px 0;
}

.markdown-view :deep(a) {
  color: #2563eb;
  text-decoration: none;
  text-underline-offset: 3px;
}

.markdown-view :deep(a:hover) {
  text-decoration: underline;
}

.markdown-view :deep(blockquote) {
  margin: 12px 0;
  padding: 8px 14px;
  border-left: 3px solid #3b82f6;
  background: #f0f7ff;
  color: #333333;
  border-radius: 0 6px 6px 0;
}

.markdown-view :deep(table) {
  width: 100%;
  border-collapse: collapse;
  border: 1px solid #d0d7de;
  border-radius: 6px;
  margin: 12px 0;
  font-size: 14px;
}

.markdown-view :deep(thead) {
  background: #f6f8fa;
}

.markdown-view :deep(th),
.markdown-view :deep(td) {
  border-bottom: 1px solid #d0d7de;
  border-right: 1px solid #d0d7de;
  padding: 8px 12px;
  text-align: left;
  color: #1a1a1a;
}

.markdown-view :deep(th:last-child),
.markdown-view :deep(td:last-child) {
  border-right: none;
}

.markdown-view :deep(img) {
  max-width: 100%;
  border-radius: 8px;
  margin: 12px 0;
}

.markdown-view :deep(code) {
  font-family: 'SFMono-Regular', Consolas, 'Liberation Mono', Menlo, monospace;
  font-size: 13px;
  padding: 2px 6px;
  border-radius: 4px;
  background: #f6f8fa;
  color: #24292f;
}

.markdown-view :deep(pre) {
  margin: 0;
  padding: 12px 16px;
  background: transparent;
  overflow-x: auto;
  font-size: 13px;
  line-height: 1.5;
}

.markdown-view :deep(.md-code-block) {
  margin: 12px 0;
  overflow: hidden;
  border-radius: 8px;
  border: 1px solid #d0d7de;
  background: #f6f8fa;
}

.markdown-view :deep(.md-code-header) {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 6px 12px;
  border-bottom: 1px solid #d0d7de;
  background: #f6f8fa;
}

.markdown-view :deep(.md-code-lang) {
  font-family: 'SFMono-Regular', Consolas, monospace;
  font-size: 11px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  color: #57606a;
}

.markdown-view :deep(.md-code-copy) {
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

.markdown-view :deep(.md-code-copy:hover) {
  background: #eaeef2;
}

.markdown-view :deep(.md-code-copy__icon) {
  display: inline-flex;
  align-items: center;
  justify-content: center;
}

.markdown-view :deep(hr) {
  border: 0;
  border-top: 1px solid #d0d7de;
  margin: 16px 0;
}
</style>
