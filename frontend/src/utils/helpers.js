/**
 * 前端工具函数
 */

/**
 * 检查文本是否为 HTML
 * @param {string} text - 要检查的文本
 * @returns {boolean}
 */
export function isHTML(text) {
  if (!text) return false
  return (
    text.includes('<table') ||
    text.includes('<tr>') ||
    text.includes('<td>') ||
    text.includes('<div') ||
    text.includes('<p>') ||
    text.includes('<h1') ||
    text.includes('<h2')
  )
}

/**
 * 检查文本是否为 Markdown
 * @param {string} text - 要检查的文本
 * @returns {boolean}
 */
export function isMarkdown(text) {
  if (!text || isHTML(text)) return false
  return (
    text.includes('##') ||
    text.includes('**') ||
    text.includes('```') ||
    text.includes('- ') ||
    text.includes('|')
  )
}

/**
 * 缩放坐标
 * @param {number} coord - 原始坐标
 * @param {number} originalSize - 原始尺寸
 * @param {number} displaySize - 显示尺寸
 * @returns {number}
 */
export function scaleCoordinate(coord, originalSize, displaySize) {
  return (coord / originalSize) * displaySize
}

/**
 * 下载文本文件
 * @param {string} text - 文本内容
 * @param {string} filename - 文件名
 * @param {string} mimeType - MIME 类型
 */
export function downloadTextFile(text, filename = 'result.txt', mimeType = 'text/plain') {
  const blob = new Blob([text], { type: mimeType })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = filename
  a.click()
  URL.revokeObjectURL(url)
}

/**
 * 获取文件扩展名（根据模式）
 * @param {string} mode - OCR 模式
 * @returns {string}
 */
export function getFileExtension(mode) {
  const extensions = {
    plain_ocr: 'txt',
    describe: 'txt',
    find_ref: 'txt',
    freeform: 'txt',
    markdown: 'md',
    tables_csv: 'csv',
    tables_md: 'md',
    kv_json: 'json',
  }
  return extensions[mode] || 'txt'
}

/**
 * 复制文本到剪贴板
 * @param {string} text - 要复制的文本
 * @returns {Promise<boolean>}
 */
export async function copyToClipboard(text) {
  try {
    await navigator.clipboard.writeText(text)
    return true
  } catch (error) {
    console.error('Failed to copy:', error)
    return false
  }
}

/**
 * 生成随机颜色（用于边界框）
 * @param {number} index - 索引
 * @returns {string}
 */
export function getBoxColor(index) {
  const colors = [
    '#00ff00', // 绿色
    '#00ffff', // 青色
    '#ff00ff', // 品红
    '#ffff00', // 黄色
    '#ff0066', // 粉红
    '#00ff99', // 青绿
    '#ff9900', // 橙色
    '#9900ff', // 紫色
  ]
  return colors[index % colors.length]
}

