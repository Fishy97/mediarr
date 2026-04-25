import { describe, expect, it } from 'vitest';
import { formatBytes, formatConfidence } from './format';

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

