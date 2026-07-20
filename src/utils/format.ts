/** 通用格式化函数 */
/**
 * 统一格式化日期时间字符串。
 *
 * @param value - 可解析为日期的值（字符串、数字、Date 对象等，或 falsy）
 * @returns 格式化后的日期时间字符串（如 `2026-07-15 14:30:25`），无效时返回原值或 '-'
 */
export function formatDateTime(value: unknown): string {
  if (!value) {
    return '-';
  }

  const date = new Date(value as string | number | Date); // 显式类型断言，符合 Date 构造器参数
  if (Number.isNaN(date.getTime())) {
    return String(value); // 无效时返回原始值的字符串形式
  }

  return new Intl.DateTimeFormat('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
  }).format(date);
}

/**
 * 文件大小格式化。
 *
 * @param value - 表示文件大小的数值（数字或数字字符串）
 * @returns 可读的文件大小字符串（如 `1.5 MB`），无效或非正数时返回 '-'
 */
export function formatFileSize(value: unknown): string {
  const size = Number(value ?? 0);
  if (!Number.isFinite(size) || size <= 0) {
    return '-';
  }

  if (size < 1024) {
    return `${size} B`;
  }
  if (size < 1024 * 1024) {
    return `${(size / 1024).toFixed(1)} KB`;
  }
  if (size < 1024 * 1024 * 1024) {
    return `${(size / 1024 / 1024).toFixed(1)} MB`;
  }
  return `${(size / 1024 / 1024 / 1024).toFixed(1)} GB`;
}

/**
 * 计数类展示兼容字符串数字。
 *
 * @param value - 表示计数的数值（数字或数字字符串）
 * @returns 千位分隔符格式的计数字符串（如 `1,234`），无效时返回 `'0'`
 */
export function formatCount(value: string | number): string {
  const count = Number(value ?? 0);
  if (!Number.isFinite(count)) {
    return '0';
  }
  return count.toLocaleString('zh-CN');
}

/** 通用数值格式化，空值返回'-' */
export function formatNum(val: number | null | undefined, digit: number): string {
  return !val ? '-' : val.toFixed(digit);
}
/** 百分比格式化 */
export function formatPercent(val: number | null | undefined): string {
  return !val ? '-' : `${val.toFixed(1)}%`;
}
