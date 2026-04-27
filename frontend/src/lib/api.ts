import type {
  AIStatus,
  ActivityRollup,
  AuthResponse,
  AuthUser,
  Backup,
  BackupRestoreResult,
  Campaign,
  CampaignResult,
  CampaignRun,
  CampaignRunResponse,
  CampaignTemplate,
  CatalogCorrectionInput,
  CatalogItem,
  Integration,
  IntegrationRefreshResult,
  IntegrationSetting,
  IntegrationSettingInput,
  IntegrationDiagnostics,
  IntegrationSyncJob,
  Job,
  JobDetail,
  Library,
  MediaServerItem,
  PathMapping,
  ProviderHealth,
  ProviderSetting,
  ProviderSettingInput,
  ProtectionRequest,
  PublicationInput,
  PublicationPlan,
  Recommendation,
  RecommendationEvidence,
  RequestSignal,
  RequestSource,
  RequestSourceInput,
  ScanResult,
  SetupStatus,
  PathMappingVerification,
  StorageLedger,
  StewardshipNotification,
  SupportBundle,
  SupportBundleResult,
  WhatIfSimulation,
} from '../types';

type Envelope<T> = { data: T };

const tokenStorageKey = 'mediarr.authToken';

export function getAuthToken(): string | null {
  return localStorage.getItem(tokenStorageKey);
}

export function setAuthToken(token: string | null): void {
  if (token) {
    localStorage.setItem(tokenStorageKey, token);
    return;
  }
  localStorage.removeItem(tokenStorageKey);
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const token = getAuthToken();
  const response = await fetch(path, {
    ...init,
    headers: {
      'Content-Type': 'application/json',
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
      ...init?.headers,
    },
  });
  if (!response.ok) {
    throw new Error(await response.text());
  }
  return response.json() as Promise<T>;
}

export const api = {
  async health(): Promise<{ status: string; service: string; timestamp: string }> {
    return request('/api/v1/health');
  },
  async setupStatus(): Promise<SetupStatus> {
    return (await request<Envelope<SetupStatus>>('/api/v1/setup/status')).data;
  },
  async setupAdmin(email: string, password: string): Promise<AuthResponse> {
    const result = (await request<Envelope<AuthResponse>>('/api/v1/setup/admin', {
      method: 'POST',
      body: JSON.stringify({ email, password }),
    })).data;
    setAuthToken(result.token);
    return result;
  },
  async login(email: string, password: string): Promise<AuthResponse> {
    const result = (await request<Envelope<AuthResponse>>('/api/v1/auth/login', {
      method: 'POST',
      body: JSON.stringify({ email, password }),
    })).data;
    setAuthToken(result.token);
    return result;
  },
  async logout(): Promise<void> {
    await request<Envelope<{ ok: boolean }>>('/api/v1/auth/logout', { method: 'POST' });
    setAuthToken(null);
  },
  async me(): Promise<AuthUser> {
    return (await request<Envelope<AuthUser>>('/api/v1/auth/me')).data;
  },
  async libraries(): Promise<Library[]> {
    return (await request<Envelope<Library[]>>('/api/v1/libraries')).data;
  },
  async scans(): Promise<ScanResult[]> {
    return (await request<Envelope<ScanResult[]>>('/api/v1/scans')).data;
  },
  async catalog(): Promise<CatalogItem[]> {
    return (await request<Envelope<CatalogItem[]>>('/api/v1/catalog')).data;
  },
  async correctCatalogItem(id: string, correction: CatalogCorrectionInput): Promise<void> {
    await request<Envelope<unknown>>(`/api/v1/catalog/${encodeURIComponent(id)}/correction`, {
      method: 'PUT',
      body: JSON.stringify(correction),
    });
  },
  async clearCatalogCorrection(id: string): Promise<void> {
    await request<Envelope<{ ok: boolean }>>(`/api/v1/catalog/${encodeURIComponent(id)}/correction`, { method: 'DELETE' });
  },
  async startScan(): Promise<Job> {
    return (await request<Envelope<Job>>('/api/v1/scans', {
      method: 'POST',
    })).data;
  },
  async recommendations(): Promise<Recommendation[]> {
    return (await request<Envelope<Recommendation[]>>('/api/v1/recommendations')).data;
  },
  async ignoreRecommendation(id: string): Promise<void> {
    await request<Envelope<{ ok: boolean }>>(`/api/v1/recommendations/${encodeURIComponent(id)}/ignore`, { method: 'POST' });
  },
  async restoreRecommendation(id: string): Promise<void> {
    await request<Envelope<{ ok: boolean }>>(`/api/v1/recommendations/${encodeURIComponent(id)}/restore`, { method: 'POST' });
  },
  async protectRecommendation(id: string): Promise<void> {
    await request<Envelope<{ ok: boolean }>>(`/api/v1/recommendations/${encodeURIComponent(id)}/protect`, { method: 'POST' });
  },
  async acceptRecommendation(id: string): Promise<void> {
    await request<Envelope<{ ok: boolean }>>(`/api/v1/recommendations/${encodeURIComponent(id)}/accept-manual`, { method: 'POST' });
  },
  async recommendationEvidence(id: string): Promise<RecommendationEvidence> {
    return (await request<Envelope<RecommendationEvidence>>(`/api/v1/recommendations/${encodeURIComponent(id)}/evidence`)).data;
  },
  async providers(): Promise<ProviderHealth[]> {
    return (await request<Envelope<ProviderHealth[]>>('/api/v1/providers')).data;
  },
  async providerSettings(): Promise<ProviderSetting[]> {
    return (await request<Envelope<ProviderSetting[]>>('/api/v1/provider-settings')).data;
  },
  async updateProviderSetting(provider: string, setting: ProviderSettingInput): Promise<ProviderSetting> {
    return (await request<Envelope<ProviderSetting>>(`/api/v1/provider-settings/${encodeURIComponent(provider)}`, {
      method: 'PUT',
      body: JSON.stringify(setting),
    })).data;
  },
  async aiStatus(): Promise<AIStatus> {
    return (await request<Envelope<AIStatus>>('/api/v1/ai/status')).data;
  },
  async integrations(): Promise<Integration[]> {
    return (await request<Envelope<Integration[]>>('/api/v1/integrations')).data;
  },
  async refreshIntegration(id: string): Promise<IntegrationRefreshResult> {
    return (await request<Envelope<IntegrationRefreshResult>>(`/api/v1/integrations/${encodeURIComponent(id)}/refresh`, { method: 'POST' })).data;
  },
  async integrationSettings(): Promise<IntegrationSetting[]> {
    return (await request<Envelope<IntegrationSetting[]>>('/api/v1/integration-settings')).data;
  },
  async updateIntegrationSetting(integration: string, setting: IntegrationSettingInput): Promise<IntegrationSetting> {
    return (await request<Envelope<IntegrationSetting>>(`/api/v1/integration-settings/${encodeURIComponent(integration)}`, {
      method: 'PUT',
      body: JSON.stringify(setting),
    })).data;
  },
  async syncIntegration(id: string): Promise<IntegrationSyncJob> {
    return (await request<Envelope<IntegrationSyncJob>>(`/api/v1/integrations/${encodeURIComponent(id)}/sync`, { method: 'POST' })).data;
  },
  async syncTautulli(): Promise<Job> {
    return (await request<Envelope<Job>>('/api/v1/integrations/tautulli/sync', { method: 'POST' })).data;
  },
  async integrationSyncStatus(id: string): Promise<IntegrationSyncJob> {
    return (await request<Envelope<IntegrationSyncJob>>(`/api/v1/integrations/${encodeURIComponent(id)}/sync`)).data;
  },
  async integrationDiagnostics(id: string): Promise<IntegrationDiagnostics> {
    return (await request<Envelope<IntegrationDiagnostics>>(`/api/v1/integrations/${encodeURIComponent(id)}/diagnostics`)).data;
  },
  async jobs(filter?: { active?: boolean; kind?: string; targetId?: string; limit?: number }): Promise<Job[]> {
    const params = new URLSearchParams();
    if (filter?.active) {
      params.set('active', 'true');
    }
    if (filter?.kind) {
      params.set('kind', filter.kind);
    }
    if (filter?.targetId) {
      params.set('targetId', filter.targetId);
    }
    if (filter?.limit) {
      params.set('limit', String(filter.limit));
    }
    const suffix = params.toString() ? `?${params.toString()}` : '';
    return (await request<Envelope<Job[]>>(`/api/v1/jobs${suffix}`)).data;
  },
  async job(id: string): Promise<JobDetail> {
    return (await request<Envelope<JobDetail>>(`/api/v1/jobs/${encodeURIComponent(id)}`)).data;
  },
  async cancelJob(id: string): Promise<Job> {
    return (await request<Envelope<Job>>(`/api/v1/jobs/${encodeURIComponent(id)}/cancel`, { method: 'POST' })).data;
  },
  async retryJob(id: string): Promise<Job> {
    return (await request<Envelope<Job>>(`/api/v1/jobs/${encodeURIComponent(id)}/retry`, { method: 'POST' })).data;
  },
  async integrationItems(id: string, unmapped = false, limit?: number): Promise<MediaServerItem[]> {
    const params = new URLSearchParams();
    if (unmapped) {
      params.set('unmapped', 'true');
    }
    if (limit && limit > 0) {
      params.set('limit', String(limit));
    }
    const suffix = params.toString() ? `?${params.toString()}` : '';
    return (await request<Envelope<MediaServerItem[]>>(`/api/v1/integrations/${encodeURIComponent(id)}/items${suffix}`)).data;
  },
  async activityRollups(serverId?: string, limit?: number): Promise<ActivityRollup[]> {
    const params = new URLSearchParams();
    if (serverId) {
      params.set('serverId', serverId);
    }
    if (limit && limit > 0) {
      params.set('limit', String(limit));
    }
    const suffix = params.toString() ? `?${params.toString()}` : '';
    return (await request<Envelope<ActivityRollup[]>>(`/api/v1/activity/rollups${suffix}`)).data;
  },
  async requestSources(): Promise<RequestSource[]> {
    return (await request<Envelope<RequestSource[] | null>>('/api/v1/request-sources')).data ?? [];
  },
  async updateRequestSource(id: string, source: RequestSourceInput): Promise<RequestSource> {
    return (await request<Envelope<RequestSource>>(`/api/v1/request-sources/${encodeURIComponent(id)}`, {
      method: 'PUT',
      body: JSON.stringify(source),
    })).data;
  },
  async syncRequestSource(id: string): Promise<{ sourceId: string; imported: number }> {
    return (await request<Envelope<{ sourceId: string; imported: number }>>(`/api/v1/request-sources/${encodeURIComponent(id)}/sync`, { method: 'POST' })).data;
  },
  async requestSignals(sourceId?: string): Promise<RequestSignal[]> {
    const params = new URLSearchParams();
    if (sourceId) {
      params.set('sourceId', sourceId);
    }
    const suffix = params.toString() ? `?${params.toString()}` : '';
    return (await request<Envelope<RequestSignal[] | null>>(`/api/v1/request-signals${suffix}`)).data ?? [];
  },
  async campaignTemplates(): Promise<CampaignTemplate[]> {
    return (await request<Envelope<CampaignTemplate[]>>('/api/v1/campaign-templates')).data;
  },
  async createCampaignFromTemplate(id: string): Promise<Campaign> {
    return (await request<Envelope<Campaign>>(`/api/v1/campaign-templates/${encodeURIComponent(id)}/create`, { method: 'POST' })).data;
  },
  async campaigns(): Promise<Campaign[]> {
    return (await request<Envelope<Campaign[] | null>>('/api/v1/campaigns')).data ?? [];
  },
  async createCampaign(campaign: Campaign): Promise<Campaign> {
    return (await request<Envelope<Campaign>>('/api/v1/campaigns', {
      method: 'POST',
      body: JSON.stringify(campaign),
    })).data;
  },
  async updateCampaign(id: string, campaign: Campaign): Promise<Campaign> {
    return (await request<Envelope<Campaign>>(`/api/v1/campaigns/${encodeURIComponent(id)}`, {
      method: 'PUT',
      body: JSON.stringify(campaign),
    })).data;
  },
  async deleteCampaign(id: string): Promise<void> {
    await request<Envelope<{ ok: boolean }>>(`/api/v1/campaigns/${encodeURIComponent(id)}`, { method: 'DELETE' });
  },
  async simulateCampaign(id: string): Promise<CampaignResult> {
    return (await request<Envelope<CampaignResult>>(`/api/v1/campaigns/${encodeURIComponent(id)}/simulate`, { method: 'POST' })).data;
  },
  async whatIfCampaign(id: string): Promise<WhatIfSimulation> {
    return (await request<Envelope<WhatIfSimulation>>(`/api/v1/campaigns/${encodeURIComponent(id)}/what-if`, { method: 'POST' })).data;
  },
  async runCampaign(id: string): Promise<CampaignRunResponse> {
    return (await request<Envelope<CampaignRunResponse>>(`/api/v1/campaigns/${encodeURIComponent(id)}/run`, { method: 'POST' })).data;
  },
  async publishCampaignPreview(id: string, input: PublicationInput): Promise<PublicationPlan> {
    return (await request<Envelope<PublicationPlan>>(`/api/v1/campaigns/${encodeURIComponent(id)}/publish-preview`, {
      method: 'POST',
      body: JSON.stringify(input),
    })).data;
  },
  async publishCampaignCollection(id: string, input: PublicationInput): Promise<PublicationPlan> {
    return (await request<Envelope<PublicationPlan>>(`/api/v1/campaigns/${encodeURIComponent(id)}/publish`, {
      method: 'POST',
      body: JSON.stringify({ ...input, confirmPublish: true }),
    })).data;
  },
  async campaignRuns(id: string): Promise<CampaignRun[]> {
    return (await request<Envelope<CampaignRun[]>>(`/api/v1/campaigns/${encodeURIComponent(id)}/runs`)).data;
  },
  async storageLedger(): Promise<StorageLedger> {
    return (await request<Envelope<StorageLedger>>('/api/v1/storage-ledger')).data;
  },
  async notifications(includeRead = false): Promise<StewardshipNotification[]> {
    const suffix = includeRead ? '?includeRead=true' : '';
    return (await request<Envelope<StewardshipNotification[] | null>>(`/api/v1/notifications${suffix}`)).data ?? [];
  },
  async markNotificationRead(id: string): Promise<StewardshipNotification> {
    return (await request<Envelope<StewardshipNotification>>(`/api/v1/notifications/${encodeURIComponent(id)}/read`, { method: 'POST' })).data;
  },
  async protectionRequests(status?: string): Promise<ProtectionRequest[]> {
    const params = new URLSearchParams();
    if (status) {
      params.set('status', status);
    }
    const suffix = params.toString() ? `?${params.toString()}` : '';
    return (await request<Envelope<ProtectionRequest[] | null>>(`/api/v1/protection-requests${suffix}`)).data ?? [];
  },
  async createProtectionRequest(protectionRequest: Omit<ProtectionRequest, 'id' | 'status' | 'createdAt' | 'decidedAt'>): Promise<ProtectionRequest> {
    return (await request<Envelope<ProtectionRequest>>('/api/v1/protection-requests', {
      method: 'POST',
      body: JSON.stringify(protectionRequest),
    })).data;
  },
  async approveProtectionRequest(id: string, decisionBy: string, note: string): Promise<ProtectionRequest> {
    return (await request<Envelope<ProtectionRequest>>(`/api/v1/protection-requests/${encodeURIComponent(id)}/approve`, {
      method: 'POST',
      body: JSON.stringify({ decisionBy, note }),
    })).data;
  },
  async declineProtectionRequest(id: string, decisionBy: string, note: string): Promise<ProtectionRequest> {
    return (await request<Envelope<ProtectionRequest>>(`/api/v1/protection-requests/${encodeURIComponent(id)}/decline`, {
      method: 'POST',
      body: JSON.stringify({ decisionBy, note }),
    })).data;
  },
  async pathMappings(): Promise<PathMapping[]> {
    return (await request<Envelope<PathMapping[]>>('/api/v1/path-mappings')).data;
  },
  async unmappedPathItems(serverId?: string, limit?: number): Promise<MediaServerItem[]> {
    const params = new URLSearchParams();
    if (serverId) {
      params.set('serverId', serverId);
    }
    if (limit && limit > 0) {
      params.set('limit', String(limit));
    }
    const suffix = params.toString() ? `?${params.toString()}` : '';
    return (await request<Envelope<MediaServerItem[]>>(`/api/v1/path-mappings/unmapped${suffix}`)).data;
  },
  async upsertPathMapping(mapping: Partial<PathMapping> & Pick<PathMapping, 'serverPathPrefix' | 'localPathPrefix'>): Promise<PathMapping> {
    const id = mapping.id?.trim();
    return (await request<Envelope<PathMapping>>(id ? `/api/v1/path-mappings/${encodeURIComponent(id)}` : '/api/v1/path-mappings', {
      method: id ? 'PUT' : 'POST',
      body: JSON.stringify(mapping),
    })).data;
  },
  async deletePathMapping(id: string): Promise<void> {
    await request<Envelope<{ ok: boolean }>>(`/api/v1/path-mappings/${encodeURIComponent(id)}`, { method: 'DELETE' });
  },
  async verifyPathMapping(id: string): Promise<PathMappingVerification> {
    return (await request<Envelope<PathMappingVerification>>(`/api/v1/path-mappings/${encodeURIComponent(id)}/verify`, { method: 'POST' })).data;
  },
  async backups(): Promise<Backup[]> {
    return (await request<Envelope<Backup[]>>('/api/v1/backups')).data;
  },
  async createBackup(): Promise<Backup> {
    return (await request<Envelope<Backup>>('/api/v1/backups', { method: 'POST' })).data;
  },
  backupDownloadUrl(name: string): string {
    return `/api/v1/backups/${encodeURIComponent(name)}`;
  },
  async createSupportBundle(): Promise<SupportBundleResult> {
    return (await request<Envelope<SupportBundleResult>>('/api/v1/support/bundles', { method: 'POST' })).data;
  },
  async supportBundles(): Promise<SupportBundle[]> {
    return (await request<Envelope<SupportBundle[]>>('/api/v1/support/bundles')).data;
  },
  supportBundleDownloadUrl(name: string): string {
    return `/api/v1/support/bundles/${encodeURIComponent(name)}`;
  },
  async restoreBackup(name: string, dryRun: boolean): Promise<BackupRestoreResult> {
    return (await request<Envelope<BackupRestoreResult>>('/api/v1/backups/restore', {
      method: 'POST',
      body: JSON.stringify({ name, dryRun, confirmRestore: !dryRun }),
    })).data;
  },
};
