<template>
  <span class="status-badge" :class="badgeClass">
    <span class="status-dot"></span>
    {{ label || '未设置' }}
  </span>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { normalizeCode } from '@/utils/format.ts'

const props = defineProps({
  label: {
    type: String,
    default: ''
  },
  type: {
    type: String,
    default: 'default'
  },
  code: {
    type: [String, Number],
    default: ''
  }
})
const STATUS_MAP = {
  parse: {
    3: 'status-success',
    2: 'status-processing',
    4: 'status-danger',
    default: 'status-waiting'
  },
  strategy: {
    3: 'status-success',
    2: 'status-processing',
    default: 'status-waiting'
  },
  index: {
    3: 'status-success',
    2: 'status-processing',
    4: 'status-danger',
    default: 'status-waiting'
  },
  task: {
    3: 'status-success',
    1: 'status-processing',
    2: 'status-processing',
    4: 'status-danger',
    default: 'status-default'
  }
}
const badgeClass = computed(() => {
  const targetMap = STATUS_MAP[props.type]
  // 无对应type直接返回默认样式
  if (!targetMap) return 'status-default'

  const code = normalizeCode(props.code)
  // 匹配状态码，无匹配取该类型默认
  return targetMap[code] ?? targetMap.default
})
</script>

<style scoped>
.status-badge {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 4px 10px;
  border-radius: 999px;
  font-size: 12px;
  font-weight: 500;
  border: 1px solid transparent;
  white-space: nowrap;
}

.status-dot {
  width: 6px;
  height: 6px;
  flex: none;
  border-radius: 50%;
  background: currentColor;
}

.status-default {
  background: rgba(92, 108, 131, 0.1);
  color: #516072;
  border-color: rgba(92, 108, 131, 0.2);
}

.status-waiting {
  background: rgba(168, 101, 32, 0.1);
  color: #9b5d1c;
  border-color: rgba(168, 101, 32, 0.2);
}

.status-processing {
  background: rgba(37, 87, 214, 0.1);
  color: #1f4ebb;
  border-color: rgba(37, 87, 214, 0.2);
}

.status-success {
  background: rgba(21, 115, 91, 0.1);
  color: #12644f;
  border-color: rgba(21, 115, 91, 0.2);
}

.status-danger {
  background: rgba(179, 76, 47, 0.1);
  color: #9f422b;
  border-color: rgba(179, 76, 47, 0.2);
}
</style>
