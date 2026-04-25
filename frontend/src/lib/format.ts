export function formatBytes(value: number): string {
  const units = ['B', 'KB', 'MB', 'GB', 'TB'];
  let size = value;
  let unit = 0;
  while (size >= 1024 && unit < units.length - 1) {
    size /= 1024;
    unit += 1;
  }
  if (unit === 0) return `${size} ${units[unit]}`;
  return `${size.toFixed(2)} ${units[unit]}`;
}

export function formatConfidence(value: number): string {
  return `${Math.round(value * 100)}%`;
}

