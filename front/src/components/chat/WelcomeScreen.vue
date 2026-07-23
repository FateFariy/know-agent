<script lang="ts" setup>
/**
 * WelcomeScreen：会话初次进入的欢迎页。
 * 顶部品牌标题 + 输入区 + 预置开场问题。
 * 复刻 React WelcomeScreen 的视觉与交互：渐变背景、浮动光晕、推荐卡片。
 */
import {computed, onBeforeUnmount, onMounted, ref, watch} from 'vue'
import {Aim, ChatLineSquare, MagicStick, Reading, Sunny} from '@element-plus/icons-vue'
import {useChatStore} from '@/stores/chat'

const store = useChatStore()
const value = ref('')
const isFocused = ref(false)
const isComposing = ref(false)
const textareaRef = ref<HTMLTextAreaElement | null>(null)

interface PromptPreset {
  id?: string
  title: string
  description: string
  prompt: string
  icon: 'reading' | 'check' | 'light'
}

const PRESET_ICONS: Array<'reading' | 'check' | 'light'> = ['reading', 'check', 'light']

const ICON_MAP: Record<string, any> = {
  reading: Reading,
  check: ChatLineSquare,
  light: Sunny
}

const DEFAULT_PRESETS: PromptPreset[] = [
  {
    title: '内容总结',
    description: '提炼 3-5 条关键信息与行动点',
    prompt: '请帮我总结以下内容，并列出3-5条要点：',
    icon: 'reading'
  },
  {
    title: '任务拆解',
    description: '把目标拆成可执行步骤与优先级',
    prompt: '请把下面需求拆解为步骤，并给出优先级和里程碑：',
    icon: 'check'
  },
  {
    title: '灵感扩展',
    description: '给出多个方案并比较优缺点',
    prompt: '围绕以下主题给出5-8个方案，并注明优缺点：',
    icon: 'light'
  }
]

const promptPresets = ref<PromptPreset[]>(DEFAULT_PRESETS)

const placeholder = computed(() =>
  store.deepThinkingEnabled ? '输入需要深度分析的问题...' : '输入你的问题...'
)
const hasContent = computed(() => value.value.trim().length > 0)

function focusInput() {
  const el = textareaRef.value
  if (!el) return
  el.focus({preventScroll: true})
}

function adjustHeight() {
  const el = textareaRef.value
  if (!el) return
  el.style.height = 'auto'
  const next = Math.min(el.scrollHeight, 160)
  el.style.height = `${next}px`
}

function onCompositionStart() {
  isComposing.value = true
}

function onCompositionEnd() {
  isComposing.value = false
}

async function handleSubmit() {
  if (store.isStreaming) {
    store.cancelGeneration()
    focusInput()
    return
  }
  if (!value.value.trim()) return
  const next = value.value
  value.value = ''
  focusInput()
  await store.sendMessage(next)
  focusInput()
}

function onKeyDown(e: KeyboardEvent) {
  if (e.key === 'Enter' && !e.shiftKey) {
    if (e.isComposing || isComposing.value || e.keyCode === 229) return
    e.preventDefault()
    handleSubmit()
  }
}

function applyPreset(prompt: string) {
  if (store.isStreaming) return
  value.value = prompt
  focusInput()
}

function toggleDeepThinking() {
  store.setDeepThinkingEnabled(!store.deepThinkingEnabled)
}

watch(value, () => {
  adjustHeight()
})

onMounted(() => {
  adjustHeight()
})

onBeforeUnmount(() => {
  textareaRef.value = null
})
</script>

<template>
  <div class="welcome-screen">
    <div aria-hidden="true" class="welcome-screen__bg welcome-screen__bg--gradient"/>
    <div aria-hidden="true" class="welcome-screen__bg welcome-screen__bg--grid"/>
    <div aria-hidden="true"
         class="welcome-screen__bg welcome-screen__bg--glow welcome-screen__bg--glow-1"/>
    <div aria-hidden="true"
         class="welcome-screen__bg welcome-screen__bg--glow welcome-screen__bg--glow-2"/>

    <div class="welcome-screen__inner">
      <div class="welcome-screen__hero">
        <span class="welcome-screen__brand">
          <el-icon><Aim/></el-icon>
          RAG 智能问答
        </span>
        <h1 class="welcome-screen__title">
          把问题变成<span class="welcome-screen__title-accent">清晰答案</span>
        </h1>
        <p class="welcome-screen__subtitle">结构化提问、知识检索与深度思考，一次对话给出可执行方案</p>
      </div>

      <div class="welcome-screen__composer">
        <div
          :class="{ 'composer-card--focused': isFocused }"
          class="composer-card"
        >
          <div class="composer-card__field">
            <textarea
              ref="textareaRef"
              v-model="value"
              :placeholder="placeholder"
              aria-label="发送消息"
              class="composer-card__input"
              rows="1"
              @blur="isFocused = false"
              @compositionend="onCompositionEnd"
              @compositionstart="onCompositionStart"
              @focus="isFocused = true"
              @keydown="onKeyDown"
            />
            <div aria-hidden="true" class="composer-card__fade"/>
          </div>
          <div class="composer-card__actions">
            <button
              :aria-pressed="store.deepThinkingEnabled"
              :class="{ 'composer-card__chip--active': store.deepThinkingEnabled }"
              :disabled="store.isStreaming"
              class="composer-card__chip"
              type="button"
              @click="toggleDeepThinking"
            >
              <el-icon class="composer-card__chip-icon">
                <MagicStick/>
              </el-icon>
              深度思考
              <span
                v-if="store.deepThinkingEnabled"
                class="composer-card__chip-dot"
              />
            </button>
            <button
              :aria-label="store.isStreaming ? '停止生成' : '发送消息'"
              :class="{
                'composer-card__submit--streaming': store.isStreaming,
                'composer-card__submit--active': hasContent
              }"
              :disabled="!hasContent && !store.isStreaming"
              class="composer-card__submit"
              type="button"
              @click="handleSubmit"
            >
              <svg
                v-if="store.isStreaming"
                fill="currentColor"
                height="16"
                viewBox="0 0 16 16"
                width="16"
              >
                <rect height="10" rx="1.5" width="10" x="3" y="3"/>
              </svg>
              <svg
                v-else
                fill="none"
                height="16"
                stroke="currentColor"
                stroke-linecap="round"
                stroke-linejoin="round"
                stroke-width="2"
                viewBox="0 0 24 24"
                width="16"
              >
                <path d="M5 12h14"/>
                <path d="m13 6 6 6-6 6"/>
              </svg>
            </button>
          </div>
        </div>

        <p v-if="store.deepThinkingEnabled"
           class="welcome-screen__hint welcome-screen__hint--accent">
          <el-icon>
            <Sunny/>
          </el-icon>
          深度思考模式已开启，AI将进行更深入的分析推理
        </p>
        <p class="welcome-screen__hint">
          <kbd>Enter</kbd> 发送
          <span class="welcome-screen__hint-sep">·</span>
          <kbd>Shift + Enter</kbd> 换行
          <span v-if="store.isStreaming" class="welcome-screen__hint-pulse">生成中...</span>
        </p>
      </div>

      <!-- <div class="welcome-screen__presets">
        <div class="welcome-screen__divider">
          <span class="welcome-screen__divider-line"/>
          <span class="welcome-screen__divider-text">试试这些开场</span>
          <span class="welcome-screen__divider-line"/>
        </div>
        <div class="welcome-screen__preset-grid">
          <button
            v-for="preset in promptPresets"
            :key="preset.id ?? preset.title"
            :disabled="store.isStreaming"
            class="preset-card"
            type="button"
            @click="applyPreset(preset.prompt)"
          >
            <div class="preset-card__head">
              <span class="preset-card__icon">
                <el-icon><component :is="ICON_MAP[preset.icon]"/></el-icon>
              </span>
              <div>
                <p class="preset-card__title">{{ preset.title }}</p>
                <p class="preset-card__desc">{{ preset.description }}</p>
              </div>
            </div>
            <div class="preset-card__foot">
              <span class="preset-card__prompt">推荐问法：{{ preset.prompt }}</span>
              <el-icon class="preset-card__arrow">
                <ArrowUpRight/>
              </el-icon>
            </div>
          </button>
        </div>
      </div> -->
    </div>
  </div>
</template>

<style scoped>
.welcome-screen {
  position: relative;
  display: flex;
  align-items: center;
  justify-content: center;
  height: 870px;
  padding: 64px 16px;
  overflow: hidden;
}

.welcome-screen__bg {
  position: absolute;
  inset: 0;
  pointer-events: none;
}

.welcome-screen__bg--gradient {
  background: linear-gradient(135deg, #f8fafc 0%, #ffffff 50%, #eff6ff 100%);
}

.welcome-screen__bg--grid {
  opacity: 0.4;
  background-image: linear-gradient(rgba(148, 163, 184, 0.08) 1px, transparent 1px),
  linear-gradient(90deg, rgba(148, 163, 184, 0.08) 1px, transparent 1px);
  background-size: 40px 40px;
}

.welcome-screen__bg--glow {
  border-radius: 50%;
  filter: blur(64px);
  animation: float 8s ease-in-out infinite;
}

.welcome-screen__bg--glow-1 {
  top: -128px;
  right: -40px;
  width: 288px;
  height: 288px;
  background: radial-gradient(circle, rgba(191, 219, 254, 0.6), transparent 70%);
}

.welcome-screen__bg--glow-2 {
  bottom: -144px;
  left: -80px;
  width: 320px;
  height: 320px;
  background: radial-gradient(circle, rgba(253, 230, 138, 0.4), transparent 70%);
  animation-delay: 2s;
}

.welcome-screen__inner {
  position: relative;
  width: 100%;
  max-width: 860px;
  display: flex;
  flex-direction: column;
  gap: 40px;
}

.welcome-screen__hero {
  text-align: center;
  animation: fade-up 0.5s ease both;
}

.welcome-screen__brand {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 4px 12px;
  border: 1px solid rgba(255, 255, 255, 0.7);
  background: rgba(255, 255, 255, 0.7);
  border-radius: 999px;
  font-size: 12px;
  font-weight: 500;
  color: #2563eb;
  box-shadow: 0 1px 2px rgba(15, 23, 42, 0.04);
}

.welcome-screen__title {
  margin: 16px 0 0;
  font-size: 40px;
  font-weight: 700;
  letter-spacing: -0.02em;
  color: #111827;
  line-height: 1.1;
}

.welcome-screen__title-accent {
  background: linear-gradient(90deg, #3b82f6, #60a5fa);
  -webkit-background-clip: text;
  background-clip: text;
  color: transparent;
}

.welcome-screen__subtitle {
  margin: 12px 0 0;
  font-size: 16px;
  color: #4b5563;
  line-height: 1.6;
}

.welcome-screen__composer {
  display: flex;
  flex-direction: column;
  gap: 12px;
  animation: fade-up 0.5s ease 80ms both;
}

.composer-card {
  position: relative;
  display: flex;
  flex-direction: column;
  gap: 12px;
  padding: 16px 20px 12px;
  border: 1px solid rgba(255, 255, 255, 0.7);
  background: rgba(255, 255, 255, 0.8);
  backdrop-filter: blur(20px);
  border-radius: 20px;
  box-shadow: 0 4px 18px rgba(15, 23, 42, 0.06);
  transition: border-color 0.2s ease, box-shadow 0.2s ease;
}

.composer-card--focused {
  border-color: #bfdbfe;
  box-shadow: 0 0 0 3px rgba(191, 219, 254, 0.5), 0 8px 32px rgba(59, 130, 246, 0.12);
}

.composer-card__field {
  position: relative;
}

.composer-card__input {
  display: block;
  width: 100%;
  min-height: 52px;
  max-height: 160px;
  padding: 8px 8px 12px;
  border: 0;
  background: transparent;
  font-size: 15px;
  line-height: 1.6;
  color: #1f2937;
  resize: none;
  outline: none;
}

.composer-card__input::placeholder {
  color: #9ca3af;
}

.composer-card__fade {
  position: absolute;
  left: 0;
  right: 0;
  bottom: 0;
  height: 10px;
  background: linear-gradient(180deg, rgba(255, 255, 255, 0), rgba(255, 255, 255, 0.4) 40%, rgba(255, 255, 255, 0.9));
  pointer-events: none;
}

.composer-card__actions {
  display: flex;
  align-items: center;
  gap: 12px;
  flex-wrap: wrap;
}

.composer-card__chip {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 6px 12px;
  border: 1px solid transparent;
  background: #f5f5f5;
  color: #6b7280;
  border-radius: 999px;
  font-size: 12px;
  font-weight: 500;
  cursor: pointer;
  transition: background 0.15s ease, color 0.15s ease, border-color 0.15s ease;
}

.composer-card__chip:hover {
  background: #eeeeee;
}

.composer-card__chip--active {
  background: #dbeafe;
  border-color: #bfdbfe;
  color: #2563eb;
}

.composer-card__chip:disabled {
  cursor: not-allowed;
  opacity: 0.6;
}

.composer-card__chip-icon {
  font-size: 14px;
}

.composer-card__chip--active .composer-card__chip-icon {
  color: #3b82f6;
}

.composer-card__chip-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: #3b82f6;
  animation: pulse 1.2s ease-in-out infinite;
}

.composer-card__submit {
  margin-left: auto;
  width: 40px;
  height: 40px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border: 0;
  border-radius: 999px;
  background: #f5f5f5;
  color: #cccccc;
  cursor: pointer;
  transition: background 0.15s ease, color 0.15s ease, transform 0.15s ease;
}

.composer-card__submit--active {
  background: #3b82f6;
  color: #ffffff;
}

.composer-card__submit--active:hover {
  background: #2563eb;
}

.composer-card__submit--streaming {
  background: #fee2e2;
  color: #ef4444;
}

.composer-card__submit--streaming:hover {
  background: #fecaca;
}

.composer-card__submit:disabled {
  cursor: not-allowed;
}

.welcome-screen__hint {
  margin: 0;
  text-align: center;
  font-size: 12px;
  color: #94a3b8;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  flex-wrap: wrap;
}

.welcome-screen__hint kbd {
  padding: 1px 6px;
  border-radius: 4px;
  background: rgba(255, 255, 255, 0.8);
  color: #6b7280;
  font-size: 11px;
  box-shadow: 0 1px 2px rgba(15, 23, 42, 0.08);
}

.welcome-screen__hint--accent {
  color: #2563eb;
  justify-content: center;
}

.welcome-screen__hint-sep {
  padding: 0 4px;
}

.welcome-screen__hint-pulse {
  margin-left: 8px;
  color: #3b82f6;
  animation: pulse 1.6s ease-in-out infinite;
}

.welcome-screen__presets {
  animation: fade-up 0.5s ease 160ms both;
}

.welcome-screen__divider {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 12px;
  color: #94a3b8;
  font-size: 12px;
  letter-spacing: 0.24em;
  text-transform: uppercase;
}

.welcome-screen__divider-line {
  width: 32px;
  height: 1px;
  background: #e5e7eb;
}

.welcome-screen__preset-grid {
  display: grid;
  gap: 12px;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  margin-top: 20px;
}

.preset-card {
  text-align: left;
  border: 1px solid rgba(255, 255, 255, 0.7);
  background: rgba(255, 255, 255, 0.7);
  border-radius: 16px;
  padding: 16px;
  cursor: pointer;
  transition: transform 0.15s ease, box-shadow 0.15s ease, border-color 0.15s ease;
  box-shadow: 0 2px 8px rgba(15, 23, 42, 0.04);
}

.preset-card:hover {
  transform: translateY(-1px);
  border-color: #bfdbfe;
  box-shadow: 0 8px 24px rgba(15, 23, 42, 0.08);
}

.preset-card:disabled {
  cursor: not-allowed;
  opacity: 0.6;
}

.preset-card__head {
  display: flex;
  align-items: center;
  gap: 12px;
}

.preset-card__icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 40px;
  height: 40px;
  border-radius: 999px;
  background: #eff6ff;
  color: #2563eb;
  font-size: 16px;
}

.preset-card__title {
  margin: 0;
  font-size: 14px;
  font-weight: 600;
  color: #1f2937;
}

.preset-card__desc {
  margin: 2px 0 0;
  font-size: 12px;
  color: #6b7280;
}

.preset-card__foot {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-top: 12px;
  font-size: 12px;
  color: #94a3b8;
}

.preset-card__prompt {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.preset-card__arrow {
  font-size: 14px;
  color: #cbd5f5;
  transition: color 0.15s ease, transform 0.15s ease;
}

.preset-card:hover .preset-card__arrow {
  color: #3b82f6;
  transform: translate(1px, -1px);
}

@keyframes float {
  0%, 100% {
    transform: translateY(0);
  }
  50% {
    transform: translateY(-12px);
  }
}

@keyframes pulse {
  0%, 100% {
    opacity: 0.45;
  }
  50% {
    opacity: 1;
  }
}

@keyframes fade-up {
  from {
    opacity: 0;
    transform: translateY(8px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

@media (max-width: 1024px) {
  .welcome-screen__preset-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (max-width: 640px) {
  .welcome-screen {
    padding: 32px 16px;
  }

  .welcome-screen__title {
    font-size: 32px;
  }

  .welcome-screen__preset-grid {
    grid-template-columns: 1fr;
  }
}
</style>
