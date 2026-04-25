export type Library = {
  id: string;
  name: string;
  kind: string;
  root: string;
};

export type Recommendation = {
  id: string;
  action:
    | 'review_duplicate'
    | 'review_oversized'
    | 'review_missing_subtitles'
    | 'review_inactive_movie'
    | 'review_never_watched_movie'
    | 'review_inactive_series'
    | 'review_abandoned_series'
    | 'review_unwatched_duplicate';
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
  serverId?: string;
  externalItemId?: string;
  lastPlayedAt?: string;
  playCount?: number;
  uniqueUsers?: number;
  favoriteCount?: number;
  verification?: string;
  evidence?: Record<string, string>;
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

export type IntegrationSetting = {
  integration: string;
  baseUrl?: string;
  apiKeyConfigured: boolean;
  apiKeyLast4?: string;
  updatedAt?: string;
};

export type IntegrationSettingInput = {
  baseUrl?: string;
  apiKey?: string;
  clearApiKey?: boolean;
  clearBaseUrl?: boolean;
};

export type IntegrationSyncJob = {
  id: string;
  serverId: string;
  status: string;
  phase?: string;
  message?: string;
  currentLabel?: string;
  processed?: number;
  total?: number;
  itemsImported: number;
  rollupsImported: number;
  unmappedItems: number;
  cursor?: string;
  error?: string;
  startedAt: string;
  completedAt?: string;
};

export type Job = {
  id: string;
  kind: string;
  targetId?: string;
  status: string;
  phase: string;
  message: string;
  currentLabel?: string;
  processed: number;
  total: number;
  itemsImported: number;
  rollupsImported: number;
  unmappedItems: number;
  error?: string;
  startedAt: string;
  updatedAt: string;
  completedAt?: string;
};

export type JobEvent = {
  id: string;
  jobId: string;
  level: string;
  phase: string;
  message: string;
  currentLabel?: string;
  processed: number;
  total: number;
  createdAt: string;
};

export type JobDetail = Job & {
  events: JobEvent[];
};

export type MediaServerItem = {
  serverId: string;
  externalId: string;
  libraryExternalId?: string;
  parentExternalId?: string;
  kind: string;
  title: string;
  year?: number;
  path?: string;
  providerIds?: Record<string, string>;
  runtimeSeconds?: number;
  dateCreated?: string;
  matchConfidence: number;
  updatedAt?: string;
};

export type ActivityRollup = {
  serverId: string;
  itemExternalId: string;
  playCount: number;
  uniqueUsers: number;
  watchedUsers: number;
  favoriteCount: number;
  lastPlayedAt?: string;
  updatedAt?: string;
};

export type PathMapping = {
  id: string;
  serverId?: string;
  serverPathPrefix: string;
  localPathPrefix: string;
  createdAt?: string;
  updatedAt?: string;
};

export type BackupRestoreResult = {
  entries?: string[];
  preRestoreBackup?: string;
  restored?: string[];
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
