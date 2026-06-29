/**
 * 时间格式化
 * @param {string|Date} time 时间
 * @param {string} fmt 格式 默认 YYYY-MM-DD HH:mm:ss
 */
export function formatTime(time, fmt = 'YYYY-MM-DD HH:mm:ss') {
  if (!time) return ''
  const d = new Date(time)
  if (isNaN(d.getTime())) return String(time)
  const pad = (n) => String(n).padStart(2, '0')
  const map = {
    YYYY: d.getFullYear(),
    MM: pad(d.getMonth() + 1),
    DD: pad(d.getDate()),
    HH: pad(d.getHours()),
    mm: pad(d.getMinutes()),
    ss: pad(d.getSeconds())
  }
  return fmt.replace(/YYYY|MM|DD|HH|mm|ss/g, (m) => map[m])
}

/**
 * 表格列 formatter：通用时间格式化
 */
export function formatTimeCol(_row, _col, cellValue) {
  return formatTime(cellValue)
}
