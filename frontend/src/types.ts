export type Library = {
  id: string;
  name: string;
  kind: string;
  root: string;
};

export type Recommendation = {
  id: string;
  action: 'review_duplicate' | 'review_oversized';
  title: string;
  explanation: string;
  spaceSavedBytes: number;
  confidence: number;
  source: string;
  affectedPaths: string[];
  destructive: boolean;
};

export type ProviderHealth = {
  name: string;
  status: string;
  attribution: string;
  rateLimit?: string;
  checkedAt: string;
};

export type Integration = {
  id: string;
  name: string;
  kind: string;
  status: string;
  description: string;
  checkedAt: string;
};

export type ScanResult = {
  libraryId: string;
  startedAt: string;
  completedAt: string;
  filesScanned: number;
  items: Array<{
    id: string;
    path: string;
    sizeBytes: number;
    parsed: {
      kind: string;
      title: string;
      canonicalKey: string;
      quality?: string;
      season?: number;
      episode?: number;
      absoluteEpisode?: number;
    };
    subtitles: string[];
  }>;
};

