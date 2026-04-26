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

export function formatVerification(value?: string): string {
  switch (value) {
    case 'local_verified':
      return 'Locally verified';
    case 'path_mapped':
      return 'Path mapped estimate';
    case 'server_reported':
      return 'Server estimate';
    case 'unmapped':
      return 'Unmapped';
    default:
      return 'Unknown';
  }
}

export function storageCertaintyForVerification(value?: string): string {
  switch (value) {
    case 'local_verified':
      return 'verified';
    case 'path_mapped':
      return 'mapped_estimate';
    case 'server_reported':
      return 'estimate';
    case 'unmapped':
      return 'unmapped';
    default:
      return 'estimate';
  }
}

export function storageCertaintyDescription(value?: string): string {
  switch (value) {
    case 'verified':
      return 'Confirmed from read-only media mount.';
    case 'mapped_estimate':
      return 'Estimated from a verified path mapping. Confirm local file sizes before treating this as guaranteed disk savings.';
    case 'estimate':
      return 'Estimated from media-server data. Verify a path mapping before treating this as guaranteed disk savings.';
    case 'unmapped':
      return 'Unmapped server path. Add and verify a path mapping before treating this as reclaimable storage.';
    default:
      return 'Estimated from media-server data. Verify a path mapping before treating this as guaranteed disk savings.';
  }
}
