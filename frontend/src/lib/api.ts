import type {
  AIStatus,
  ActivityRollup,
  AuthResponse,
  AuthUser,
  BackupRestoreResult,
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
  Recommendation,
  RecommendationEvidence,
  ScanResult,
  SetupStatus,
  PathMappingVerification,
  SupportBundle,
  SupportBundleResult,
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
  async integrationItems(id: string, unmapped = false): Promise<MediaServerItem[]> {
    const suffix = unmapped ? '?unmapped=true' : '';
    return (await request<Envelope<MediaServerItem[]>>(`/api/v1/integrations/${encodeURIComponent(id)}/items${suffix}`)).data;
  },
  async activityRollups(serverId?: string): Promise<ActivityRollup[]> {
    const suffix = serverId ? `?serverId=${encodeURIComponent(serverId)}` : '';
    return (await request<Envelope<ActivityRollup[]>>(`/api/v1/activity/rollups${suffix}`)).data;
  },
  async pathMappings(): Promise<PathMapping[]> {
    return (await request<Envelope<PathMapping[]>>('/api/v1/path-mappings')).data;
  },
  async unmappedPathItems(serverId?: string): Promise<MediaServerItem[]> {
    const suffix = serverId ? `?serverId=${encodeURIComponent(serverId)}` : '';
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
  async createBackup(): Promise<{ path: string }> {
    return (await request<Envelope<{ path: string }>>('/api/v1/backups', { method: 'POST' })).data;
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
  async restoreBackup(path: string, dryRun: boolean): Promise<BackupRestoreResult> {
    return (await request<Envelope<BackupRestoreResult>>('/api/v1/backups/restore', {
      method: 'POST',
      body: JSON.stringify({ path, dryRun }),
    })).data;
  },
};
