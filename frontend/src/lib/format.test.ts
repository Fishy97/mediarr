import { describe, expect, it } from 'vitest';
import { formatBytes, formatConfidence, storageCertaintyDefinition } from './format';

describe('formatBytes', () => {
  it('formats saved storage in human units', () => {
    expect(formatBytes(8_000_000_000)).toBe('7.45 GB');
  });
});

describe('formatConfidence', () => {
  it('formats confidence as a percentage', () => {
    expect(formatConfidence(0.923)).toBe('92%');
  });
});

describe('storageCertaintyDefinition', () => {
  it('uses canonical proof language for storage certainty', () => {
    expect(storageCertaintyDefinition('server_reported')).toBe('Server estimate: Jellyfin/Plex/Emby reports this path and size. Mediarr has not verified it on disk.');
    expect(storageCertaintyDefinition('path_mapped')).toBe('Path mapped estimate: Mediarr translated the server path to a local mount, but size still needs local confirmation.');
    expect(storageCertaintyDefinition('local_verified')).toBe('Locally verified: Mediarr found the file on a read-only mount and confirmed the size.');
    expect(storageCertaintyDefinition('unmapped')).toBe('Unmapped: Mediarr cannot connect the server path to a local file path yet.');
  });
});
