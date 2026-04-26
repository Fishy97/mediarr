export type AffectedPathGroup = {
  label: string;
  count: number;
  paths: string[];
  estimatedBytes?: number;
};

const seasonPattern = /^season\s+\d+/i;

export function groupAffectedPaths(paths: string[], subjectKind?: string): AffectedPathGroup[] {
  const normalizedPaths = paths.filter((path) => path.trim() !== '');
  if (normalizedPaths.length === 0) {
    return [];
  }

  if (subjectKind === 'movie') {
    return [buildGroup('File', normalizedPaths)];
  }

  const groups = new Map<string, string[]>();
  for (const path of normalizedPaths) {
    const label = groupLabelForPath(path);
    const existing = groups.get(label) || [];
    existing.push(path);
    groups.set(label, existing);
  }

  return Array.from(groups.entries())
    .sort(([left], [right]) => naturalCompare(left, right))
    .map(([label, groupPaths]) => buildGroup(label, groupPaths));
}

function groupLabelForPath(path: string): string {
  const parts = path.split(/[\\/]+/).filter(Boolean);
  const season = parts.slice(0, -1).find((part) => seasonPattern.test(part.trim()));
  if (season) {
    return season.trim();
  }

  if (parts.length >= 2) {
    return parts[parts.length - 2].trim() || 'Files';
  }
  return 'Files';
}

function buildGroup(label: string, paths: string[]): AffectedPathGroup {
  return {
    label,
    count: paths.length,
    paths: [...paths].sort(naturalCompare),
  };
}

function naturalCompare(left: string, right: string): number {
  return left.localeCompare(right, undefined, { numeric: true, sensitivity: 'base' });
}
