<script lang="ts" setup>
/**
 * MarkdownRenderer：基于 markdown-it + highlight.js 渲染助手回复。
 * - 行内 code 走浅灰胶囊样式；
 * - 多行 code 走带语言标签的代码块 + 复制按钮。
 * 复刻 React MarkdownRenderer 的视觉与交互。
 */
import {computed, nextTick, ref, watch} from 'vue'
import MarkdownIt from 'markdown-it'
import hljs from 'highlight.js/lib/core'
import bash from 'highlight.js/lib/languages/bash'
import java from 'highlight.js/lib/languages/java'
import javascript from 'highlight.js/lib/languages/javascript'
import json from 'highlight.js/lib/languages/json'
import python from 'highlight.js/lib/languages/python'
import sql from 'highlight.js/lib/languages/sql'
import typescript from 'highlight.js/lib/languages/typescript'
import xml from 'highlight.js/lib/languages/xml'
import yaml from 'highlight.js/lib/languages/yaml'
import {ElMessage} from 'element-plus'

// markdown-it 子模块类型不能从默认导出直接拿，这里从实例上推导。
type MdToken = ReturnType<MarkdownIt['parse']>[number]
type MdOptions = NonNullable<Parameters<MarkdownIt['parse']>[1]>
type MdRenderer = MarkdownIt['renderer']
type MdRenderRule = (tokens: MdToken[], idx: number, options: MdOptions, env: unknown, self: MdRenderer) => string

const props = defineProps<{
  content: string
}>()

const containerRef = ref<HTMLElement | null>(null)
const imageErrors = ref<Record<number, boolean>>({})

hljs.registerLanguage('bash', bash)
hljs.registerLanguage('java', java)
hljs.registerLanguage('javascript', javascript)
hljs.registerLanguage('json', json)
hljs.registerLanguage('python', python)
hljs.registerLanguage('sql', sql)
hljs.registerLanguage('typescript', typescript)
hljs.registerLanguage('xml', xml)
hljs.registerLanguage('yaml', yaml)

const md: MarkdownIt = new MarkdownIt({
  html: false,
  linkify: true,
  breaks: true,
  highlight: (str: string, lang: string) => {
    if (lang && hljs.getLanguage(lang)) {
      try {
        return `<pre class="hljs-block"><code class="hljs language-${lang}">${hljs.highlight(str, {
          language: lang,
          ignoreIllegals: true
        }).value}</code></pre>`
      } catch {
        // fallback
      }
    }
    return `<pre class="hljs-block"><code class="hljs">${md.utils.escapeHtml(str)}</code></pre>`
  }
})

// 链接渲染：补充 target=_blank / rel=noreferrer。
const defaultLinkOpen: MdRenderRule =
  md.renderer.rules.link_open ||
  function (tokens, idx, options, _env, self) {
    return self.renderToken(tokens, idx, options)
  }
md.renderer.rules.link_open = function (tokens, idx, options, env, self) {
  const token = tokens[idx]
  if (token) {
    token.attrSet('target', '_blank')
    token.attrSet('rel', 'noreferrer')
  }
  return defaultLinkOpen(tokens, idx, options, env, self)
}

const renderedHtml = computed(() => {
  if (!props.content) return ''
  // 替换 ```lang 代码块占位：markdown-it 内部已经处理。这里只确保安全。
  return md.render(props.content)
})

function handleImageError(idx: number) {
  imageErrors.value = {...imageErrors.value, [idx]: true}
}

async function highlightAll() {
  await nextTick()
  if (!containerRef.value) return
  const blocks = containerRef.value.querySelectorAll('pre code.hljs')
  blocks.forEach((block) => {
    if (!(block as HTMLElement).dataset.highlighted) {
      hljs.highlightElement(block as HTMLElement)
    }
  })
}

async function copyCode(value: string) {
  try {
    await navigator.clipboard.writeText(value)
    ElMessage.success('已复制代码')
  } catch {
    ElMessage.error('复制失败')
  }
}

watch(
  () => props.content,
  () => {
    highlightAll()
  },
  {immediate: false}
)

// 自定义代码块渲染：增加复制按钮与语言标签。
// 这里通过 watch DOM 节点，把 markdown-it 输出的 pre.hljs-block 改造成带 header 的结构。
async function enrichCodeBlocks() {
  await nextTick()
  if (!containerRef.value) return
  const blocks = containerRef.value.querySelectorAll('pre.hljs-block')
  blocks.forEach((block) => {
    if ((block as HTMLElement).dataset.enriched) return
    const codeEl = block.querySelector('code')
    if (!codeEl) return
    const langMatch = /language-(\w+)/.exec(codeEl.className)
    const lang = langMatch?.[1] || 'text'
    const text = codeEl.textContent || ''
    const wrapper = document.createElement('div')
    wrapper.className = 'md-code-block'
    const header = document.createElement('div')
    header.className = 'md-code-header'
    const langEl = document.createElement('span')
    langEl.className = 'md-code-lang'
    langEl.textContent = lang.toUpperCase()
    const copyBtn = document.createElement('button')
    copyBtn.className = 'md-code-copy'
    copyBtn.type = 'button'
    copyBtn.setAttribute('aria-label', '复制代码')
    copyBtn.innerHTML = '<span class="md-code-copy__icon"><svg viewBox="0 0 16 16" width="14" height="14" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><rect x="5" y="5" width="9" height="9" rx="1.5"></rect><path d="M11 5V3.5A1.5 1.5 0 0 0 9.5 2H4.5A1.5 1.5 0 0 0 3 3.5V9.5A1.5 1.5 0 0 0 4.5 11H5"></path></svg></span>'
    copyBtn.addEventListener('click', () => {
      copyCode(text)
    })
    header.appendChild(langEl)
    header.appendChild(copyBtn)
    block.parentNode?.insertBefore(wrapper, block)
    wrapper.appendChild(header)
    wrapper.appendChild(block)
    ;(block as HTMLElement).dataset.enriched = '1'
  })
}

watch(
  () => props.content,
  () => {
    enrichCodeBlocks()
  },
  {immediate: true}
)
</script>

<template>
  <div ref="containerRef" class="markdown-body">
    <!-- eslint-disable-next-line vue/no-v-html -->
    <div v-html="renderedHtml"/>
    <!-- 图片兜底：markdown-it 解析出的 <img> 节点附加错误处理。这里仅提供一段提示文本提示。 -->
    <span class="markdown-body__hidden">{{ imageErrors }}</span>
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

.markdown-body__hidden {
  display: none;
}
</style>
