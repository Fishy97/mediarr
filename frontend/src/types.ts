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
    | 'review_unwatched_duplicate'
    | 'review_campaign_match';
  state?: RecommendationState;
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

export type RecommendationState = 'new' | 'reviewing' | 'ignored' | 'protected' | 'accepted_for_manual_action';

export type RecommendationEvidence = {
  recommendationId: string;
  state: RecommendationState;
  title: string;
  explanation: string;
  confidence: number;
  destructive: boolean;
  affectedPaths: string[];
  suppressionReasons: string[];
  raw: Record<string, string>;
  storage: {
    spaceSavedBytes: number;
    estimatedSavingsBytes: number;
    verifiedSavingsBytes: number;
    verification: string;
    certainty: string;
    basis: string;
    risk: string;
  };
  activity: {
    serverId?: string;
    externalItemId?: string;
    lastPlayedAt?: string;
    playCount: number;
    uniqueUsers: number;
    favoriteCount: number;
  };
  source: {
    rule: string;
    ai?: string;
    aiConfidence?: number;
  };
  proof: Array<{
    label: string;
    value: string;
    status: string;
  }>;
};

export type CampaignRuleField =
  | 'kind'
  | 'libraryName'
  | 'verification'
  | 'storageBytes'
  | 'estimatedSavingsBytes'
  | 'verifiedSavingsBytes'
  | 'lastPlayedDays'
  | 'addedDays'
  | 'playCount'
  | 'uniqueUsers'
  | 'favoriteCount'
  | 'confidence';

export type CampaignRuleOperator =
  | 'equals'
  | 'not_equals'
  | 'in'
  | 'not_in'
  | 'greater_than'
  | 'greater_or_equal'
  | 'less_than'
  | 'less_or_equal'
  | 'is_empty'
  | 'is_not_empty';

export type CampaignRule = {
  field: CampaignRuleField;
  operator: CampaignRuleOperator;
  value?: string;
  values?: string[];
};

export type Campaign = {
  id: string;
  name: string;
  description?: string;
  enabled: boolean;
  targetKinds: string[];
  targetLibraryNames?: string[];
  rules: CampaignRule[];
  requireAllRules: boolean;
  minimumConfidence: number;
  minimumStorageBytes: number;
  createdAt?: string;
  updatedAt?: string;
  lastRunAt?: string;
};

export type CampaignCandidate = {
  key: string;
  serverId?: string;
  externalItemId?: string;
  title: string;
  kind: string;
  libraryName?: string;
  verification?: string;
  estimatedSavingsBytes: number;
  verifiedSavingsBytes: number;
  confidence: number;
  addedAt?: string;
  lastPlayedAt?: string;
  playCount: number;
  uniqueUsers: number;
  favoriteCount: number;
  affectedPaths: string[];
  evidence?: Record<string, string>;
};

export type CampaignRuleResult = {
  rule: CampaignRule;
  matched: boolean;
  reason: string;
};

export type CampaignResultItem = {
  candidate: CampaignCandidate;
  matchedRules: CampaignRuleResult[];
  suppressionReasons: string[];
  suppressed: boolean;
};

export type CampaignResult = {
  campaignId: string;
  enabled: boolean;
  matched: number;
  suppressed: number;
  totalEstimatedSavingsBytes: number;
  totalVerifiedSavingsBytes: number;
  confidenceMin: number;
  confidenceAverage: number;
  confidenceMax: number;
  items: CampaignResultItem[];
};

export type CampaignRun = {
  id: string;
  campaignId: string;
  status: string;
  matched: number;
  suppressed: number;
  estimatedSavingsBytes: number;
  verifiedSavingsBytes: number;
  error?: string;
  startedAt: string;
  completedAt?: string;
};

export type CampaignRunResponse = {
  run: CampaignRun;
  result: CampaignResult;
};

export type CampaignTemplate = {
  id: string;
  name: string;
  description: string;
  campaign: {
    id?: string;
    name: string;
    description?: string;
    enabled: boolean;
    targetKinds: string[];
    targetLibraryNames?: string[];
    rules: CampaignRule[];
    requireAllRules: boolean;
    minimumConfidence: number;
    minimumStorageBytes: number;
  };
};

export type WhatIfSimulation = {
  campaigns: number;
  matched: number;
  suppressed: number;
  estimatedBytes: number;
  verifiedBytes: number;
  blockedUnmapped: number;
  protectionConflicts: number;
  requestConflicts: number;
};

export type PublicationPlan = {
  id: string;
  campaignId: string;
  serverId: string;
  collectionTitle: string;
  dryRun: boolean;
  status: string;
  publishableItems: number;
  blockedItems: number;
  publishableEstimatedBytes: number;
  blockedEstimatedBytes: number;
  items: Array<{
    externalItemId?: string;
    title: string;
    verification: string;
    estimatedBytes: number;
    publishable: boolean;
    blockedReason?: string;
  }>;
  createdAt?: string;
  publishedAt?: string;
  error?: string;
};

export type PublicationInput = {
  serverId: string;
  collectionTitle: string;
  minimumVerification?: string;
  confirmPublish?: boolean;
};

export type RequestSource = {
  id: string;
  kind: string;
  name: string;
  baseUrl?: string;
  apiKeyConfigured: boolean;
  apiKeyLast4?: string;
  enabled: boolean;
  lastSyncedAt?: string;
  updatedAt?: string;
};

export type RequestSourceInput = {
  kind?: string;
  name?: string;
  baseUrl?: string;
  apiKey?: string;
  clearApiKey?: boolean;
  enabled?: boolean;
};

export type RequestSignal = {
  sourceId: string;
  externalRequestId: string;
  mediaType: string;
  externalMediaId?: string;
  title?: string;
  status: string;
  availability: string;
  requestedBy?: string;
  providerIds: Record<string, string>;
  estimatedBytes?: number;
  requestedAt?: string;
  approvedAt?: string;
  availableAt?: string;
  updatedAt?: string;
};

export type StorageLedger = {
  locallyVerifiedBytes: number;
  mappedEstimateBytes: number;
  serverReportedBytes: number;
  blockedUnmappedBytes: number;
  protectedBytes: number;
  acceptedManualBytes: number;
  requestedMediaBytes: number;
  totalEstimatedBytes: number;
  totalVerifiedBytes: number;
};

export type StewardshipNotification = {
  id: string;
  level: string;
  title: string;
  body?: string;
  eventType?: string;
  fields?: Record<string, string>;
  read: boolean;
  createdAt?: string;
  readAt?: string;
};

export type ProtectionRequest = {
  id?: string;
  recommendationId?: string;
  serverId?: string;
  externalItemId?: string;
  title: string;
  path?: string;
  reason?: string;
  requestedBy: string;
  status?: string;
  decisionBy?: string;
  decisionNote?: string;
  createdAt?: string;
  decidedAt?: string;
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
  retryPolicy?: string;
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
  autoSyncEnabled: boolean;
  autoSyncIntervalMinutes: number;
  updatedAt?: string;
};

export type IntegrationSettingInput = {
  baseUrl?: string;
  apiKey?: string;
  clearApiKey?: boolean;
  clearBaseUrl?: boolean;
  autoSyncEnabled?: boolean;
  autoSyncIntervalMinutes?: number;
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

export type IntegrationDiagnostics = {
  targetId: string;
  generatedAt: string;
  server: {
    name: string;
    kind: string;
    status: string;
  };
  summary: {
    libraries: number;
    users: number;
    movies: number;
    series: number;
    episodes: number;
    videos: number;
    animeItems: number;
    files: number;
    activityRollups: number;
    recommendations: number;
    destructiveRecommendations: number;
    serverReportedBytes: number;
    locallyVerifiedBytes: number;
    recommendationBytes: number;
    unmappedFiles: number;
    filesMissingSize: number;
    acceptedForRecommendationBytes: number;
  };
  warnings: string[];
  progressSamples: Array<{
    phase: string;
    message: string;
    currentLabel?: string;
    processed: number;
    total: number;
  }>;
  topRecommendations: Array<{
    id: string;
    action: string;
    title: string;
    spaceSavedBytes: number;
    confidence: number;
    source: string;
    verification: string;
    affectedPaths?: string[];
  }>;
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

export type PathMappingVerification = {
  mapping: PathMapping;
  matchedFiles: number;
  mappedFiles: number;
  verifiedFiles: number;
  missingFiles: number;
  updatedAt?: string;
};

export type BackupRestoreResult = {
  entries?: string[];
  preRestoreBackup?: string;
  restored?: string[];
};

export type Backup = {
  name: string;
  path: string;
  sizeBytes: number;
  createdAt: string;
};

export type SupportBundle = {
  name: string;
  path: string;
  sizeBytes: number;
  createdAt: string;
};

export type SupportBundleResult = SupportBundle & {
  files: string[];
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
