export type Library = {
  id: string;
  name: string;
  kind: string;
  root: string;
};

export type Recommendation = {
  id: string;
  action: 'review_duplicate' | 'review_oversized' | 'review_missing_subtitles';
  title: string;
  explanation: string;
  spaceSavedBytes: number;
  confidence: number;
  source: string;
  affectedPaths: string[];
  destructive: boolean;
  aiRationale?: string;
  aiTags?: string[];
  aiConfidence?: number;
  aiSource?: string;
};

export type ProviderHealth = {
  name: string;
  status: string;
  attribution: string;
  rateLimit?: string;
  checkedAt: string;
};

export type ProviderSetting = {
  provider: string;
  baseUrl?: string;
  apiKeyConfigured: boolean;
  apiKeyLast4?: string;
  updatedAt?: string;
};

export type ProviderSettingInput = {
  baseUrl?: string;
  apiKey?: string;
  clearApiKey?: boolean;
  clearBaseUrl?: boolean;
};

export type CatalogCorrectionInput = {
  title: string;
  kind: string;
  year?: number;
  canonicalKey?: string;
  provider?: string;
  providerId?: string;
  confidence?: number;
};

export type Integration = {
  id: string;
  name: string;
  kind: string;
  status: string;
  description: string;
  checkedAt: string;
};

export type IntegrationRefreshResult = {
  targetId: string;
  status: string;
  message: string;
  requestedAt: string;
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
      year?: number;
      quality?: string;
      season?: number;
      episode?: number;
      absoluteEpisode?: number;
    };
    subtitles: string[];
  }>;
};

export type CatalogItem = {
  id: string;
  libraryId: string;
  path: string;
  canonicalKey: string;
  title: string;
  kind: string;
  year?: number;
  sizeBytes: number;
  quality?: string;
  fingerprint: string;
  subtitles: string[];
  metadataProvider?: string;
  metadataProviderId?: string;
  metadataConfidence?: number;
  metadataCorrected: boolean;
  modifiedAt: string;
  scannedAt: string;
};

export type SetupStatus = {
  setupRequired: boolean;
};

export type AuthUser = {
  id: string;
  email: string;
  role: string;
  createdAt?: string;
};

export type AuthResponse = {
  user: AuthUser;
  token: string;
  expiresAt?: string;
};

export type AIStatus = {
  status: string;
  model: string;
  modelAvailable: boolean;
  checkedAt: string;
};
