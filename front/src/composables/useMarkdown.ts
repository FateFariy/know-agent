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

// markdown-it 子模块类型不能从默认导出直接拿，这里从实例上推导。
type MdToken = ReturnType<MarkdownIt['parse']>[number]
type MdOptions = NonNullable<Parameters<MarkdownIt['parse']>[1]>
type MdRenderer = MarkdownIt['renderer']
type MdRenderRule = (
  tokens: MdToken[],
  idx: number,
  options: MdOptions,
  env: unknown,
  self: MdRenderer
) => string

let registered = false

function registerLanguages(): void {
  if (registered) return
  hljs.registerLanguage('bash', bash)
  hljs.registerLanguage('java', java)
  hljs.registerLanguage('javascript', javascript)
  hljs.registerLanguage('json', json)
  hljs.registerLanguage('python', python)
  hljs.registerLanguage('sql', sql)
  hljs.registerLanguage('typescript', typescript)
  hljs.registerLanguage('xml', xml)
  hljs.registerLanguage('yaml', yaml)
  registered = true
}

export interface MarkdownRendererOptions {
  /**
   * 是否允许原始 HTML。默认 false 以规避 XSS。
   * 仅在完全信任数据源时可显式打开。
   */
  allowHtml?: boolean
}

/**
 * 创建一个安全默认配置的 markdown-it 实例，并补齐常用规则：
 * - 链接自动 target=_blank + rel=noreferrer；
 * - 代码块通过 highlight.js 渲染并附带 language-* class。
 */
export function createMarkdownRenderer(options: MarkdownRendererOptions = {}): MarkdownIt {
  registerLanguages()
  const allowHtml = options.allowHtml ?? false
  const md: MarkdownIt = new MarkdownIt({
    html: allowHtml,
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
          // fallback below
        }
      }
      return `<pre class="hljs-block"><code class="hljs">${md.utils.escapeHtml(str)}</code></pre>`
    }
  })

  const defaultLinkOpen: MdRenderRule =
    md.renderer.rules.link_open ||
    function (tokens, idx, renderOptions, _env, self) {
      return self.renderToken(tokens, idx, renderOptions)
    }
  md.renderer.rules.link_open = function (tokens, idx, renderOptions, env, self) {
    const token = tokens[idx]
    if (token) {
      token.attrSet('target', '_blank')
      token.attrSet('rel', 'noreferrer')
    }
    return defaultLinkOpen(tokens, idx, renderOptions, env, self)
  }

  return md
}

export interface EnrichCodeBlockOptions {
  /**
   * 自定义复制回调，传入 null 表示不绑定复制逻辑。
   * 默认使用 navigator.clipboard.writeText。
   */
  onCopy?: ((text: string) => void) | null
}

/**
 * 把 markdown-it 输出的 pre.hljs-block 改造成带语言标签 + 复制按钮的代码块。
 * 该方法应当在下一次 DOM 更新后调用（nextTick）。
 */
export function enrichCodeBlocks(
  root: HTMLElement | null,
  options: EnrichCodeBlockOptions = {}
): void {
  if (!root) return
  const onCopy = options.onCopy
  const blocks = root.querySelectorAll('pre.hljs-block')
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
    copyBtn.innerHTML =
      '<span class="md-code-copy__icon"><svg viewBox="0 0 16 16" width="14" height="14" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><rect x="5" y="5" width="9" height="9" rx="1.5"></rect><path d="M11 5V3.5A1.5 1.5 0 0 0 9.5 2H4.5A1.5 1.5 0 0 0 3 3.5V9.5A1.5 1.5 0 0 0 4.5 11H5"></path></svg></span>'
    if (onCopy !== null) {
      copyBtn.addEventListener('click', () => {
        if (onCopy) {
          onCopy(text)
        } else {
          defaultCopy(text)
        }
      })
    }
    header.appendChild(langEl)
    header.appendChild(copyBtn)
    block.parentNode?.insertBefore(wrapper, block)
    wrapper.appendChild(header)
    wrapper.appendChild(block)
    ;(block as HTMLElement).dataset.enriched = '1'
  })
}

async function defaultCopy(text: string): Promise<void> {
  try {
    if (navigator.clipboard?.writeText) {
      await navigator.clipboard.writeText(text)
    } else {
      const textarea = document.createElement('textarea')
      textarea.value = text
      textarea.style.position = 'fixed'
      textarea.style.opacity = '0'
      document.body.appendChild(textarea)
      textarea.select()
      document.execCommand('copy')
      document.body.removeChild(textarea)
    }
  } catch {
    // noop
  }
}
