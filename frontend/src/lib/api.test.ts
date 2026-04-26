import { beforeEach, describe, expect, test, vi } from 'vitest';
import { api, setAuthToken } from './api';

describe('api auth helpers', () => {
  beforeEach(() => {
    vi.restoreAllMocks();
    localStorage.clear();
  });

  test('setup and login endpoints return auth payloads', async () => {
    const fetchMock = vi.fn()
      .mockResolvedValueOnce(jsonResponse({ data: { setupRequired: true } }))
      .mockResolvedValueOnce(jsonResponse({ data: { token: 'setup-token', user: { id: 'usr_1', email: 'admin@example.test', role: 'admin' } } }))
      .mockResolvedValueOnce(jsonResponse({ data: { token: 'login-token', user: { id: 'usr_1', email: 'admin@example.test', role: 'admin' } } }));
    vi.stubGlobal('fetch', fetchMock);

    await expect(api.setupStatus()).resolves.toEqual({ setupRequired: true });
    await expect(api.setupAdmin('admin@example.test', 'correct horse battery staple')).resolves.toMatchObject({ token: 'setup-token' });
    await expect(api.login('admin@example.test', 'correct horse battery staple')).resolves.toMatchObject({ token: 'login-token' });

    expect(fetchMock).toHaveBeenNthCalledWith(2, '/api/v1/setup/admin', expect.objectContaining({ method: 'POST' }));
    expect(fetchMock).toHaveBeenNthCalledWith(3, '/api/v1/auth/login', expect.objectContaining({ method: 'POST' }));
  });

  test('stored auth token is sent as bearer token', async () => {
    setAuthToken('session-token');
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse({ data: [] }));
    vi.stubGlobal('fetch', fetchMock);

    await api.catalog();

    expect(fetchMock).toHaveBeenCalledWith('/api/v1/catalog', expect.objectContaining({
      headers: expect.objectContaining({ Authorization: 'Bearer session-token' }),
    }));
  });

  test('recommendation actions call review queue endpoints', async () => {
    const fetchMock = vi.fn()
      .mockResolvedValueOnce(jsonResponse({ data: { ok: true } }))
      .mockResolvedValueOnce(jsonResponse({ data: { ok: true } }))
      .mockResolvedValueOnce(jsonResponse({ data: { ok: true } }))
      .mockResolvedValueOnce(jsonResponse({ data: { ok: true } }))
      .mockResolvedValueOnce(jsonResponse({ data: { recommendationId: 'rec_1', state: 'new', proof: [] } }));
    vi.stubGlobal('fetch', fetchMock);

    await api.ignoreRecommendation('rec_1');
    await api.restoreRecommendation('rec_1');
    await api.protectRecommendation('rec_1');
    await api.acceptRecommendation('rec_1');
    await expect(api.recommendationEvidence('rec_1')).resolves.toMatchObject({ recommendationId: 'rec_1', state: 'new' });

    expect(fetchMock).toHaveBeenNthCalledWith(1, '/api/v1/recommendations/rec_1/ignore', expect.objectContaining({ method: 'POST' }));
    expect(fetchMock).toHaveBeenNthCalledWith(2, '/api/v1/recommendations/rec_1/restore', expect.objectContaining({ method: 'POST' }));
    expect(fetchMock).toHaveBeenNthCalledWith(3, '/api/v1/recommendations/rec_1/protect', expect.objectContaining({ method: 'POST' }));
    expect(fetchMock).toHaveBeenNthCalledWith(4, '/api/v1/recommendations/rec_1/accept-manual', expect.objectContaining({ method: 'POST' }));
    expect(fetchMock).toHaveBeenNthCalledWith(5, '/api/v1/recommendations/rec_1/evidence', expect.any(Object));
  });

  test('provider settings calls redactable settings endpoints', async () => {
    const fetchMock = vi.fn()
      .mockResolvedValueOnce(jsonResponse({ data: [{ provider: 'tmdb', apiKeyConfigured: true, apiKeyLast4: 'abcd' }] }))
      .mockResolvedValueOnce(jsonResponse({ data: { provider: 'tmdb', apiKeyConfigured: true, apiKeyLast4: 'abcd' } }));
    vi.stubGlobal('fetch', fetchMock);

    await expect(api.providerSettings()).resolves.toHaveLength(1);
    await api.updateProviderSetting('tmdb', { apiKey: 'secret-token-abcd' });

    expect(fetchMock).toHaveBeenNthCalledWith(1, '/api/v1/provider-settings', expect.any(Object));
    expect(fetchMock).toHaveBeenNthCalledWith(2, '/api/v1/provider-settings/tmdb', expect.objectContaining({
      method: 'PUT',
      body: JSON.stringify({ apiKey: 'secret-token-abcd' }),
    }));
  });

  test('catalog correction calls item correction endpoints', async () => {
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse({ data: { ok: true } }));
    vi.stubGlobal('fetch', fetchMock);

    await api.correctCatalogItem('file_1', { title: 'Arrival', kind: 'movie', year: 2016 });
    await api.clearCatalogCorrection('file_1');

    expect(fetchMock).toHaveBeenNthCalledWith(1, '/api/v1/catalog/file_1/correction', expect.objectContaining({
      method: 'PUT',
      body: JSON.stringify({ title: 'Arrival', kind: 'movie', year: 2016 }),
    }));
    expect(fetchMock).toHaveBeenNthCalledWith(2, '/api/v1/catalog/file_1/correction', expect.objectContaining({ method: 'DELETE' }));
  });

  test('integration refresh calls sync target endpoint', async () => {
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse({ data: { targetId: 'jellyfin', status: 'requested' } }));
    vi.stubGlobal('fetch', fetchMock);

    await expect(api.refreshIntegration('jellyfin')).resolves.toMatchObject({ targetId: 'jellyfin', status: 'requested' });

    expect(fetchMock).toHaveBeenCalledWith('/api/v1/integrations/jellyfin/refresh', expect.objectContaining({ method: 'POST' }));
  });

  test('media server ingestion calls sync activity and mapping endpoints', async () => {
    const fetchMock = vi.fn()
      .mockResolvedValueOnce(jsonResponse({ data: { serverId: 'jellyfin', status: 'completed', itemsImported: 1 } }))
      .mockResolvedValueOnce(jsonResponse({ data: [{ serverId: 'jellyfin', externalId: 'item_1', title: 'Arrival' }] }))
      .mockResolvedValueOnce(jsonResponse({ data: [{ serverId: 'jellyfin', itemExternalId: 'item_1', playCount: 2 }] }))
      .mockResolvedValueOnce(jsonResponse({ data: [{ id: 'map_1', serverPathPrefix: '/mnt/media', localPathPrefix: '/media' }] }))
      .mockResolvedValueOnce(jsonResponse({ data: [{ serverId: 'jellyfin', externalId: 'item_2', title: 'Unmapped' }] }))
      .mockResolvedValueOnce(jsonResponse({ data: { id: 'map_1', serverPathPrefix: '/mnt/media', localPathPrefix: '/media' } }))
      .mockResolvedValueOnce(jsonResponse({ data: { mapping: { id: 'map_1' }, matchedFiles: 1, verifiedFiles: 1 } }));
    vi.stubGlobal('fetch', fetchMock);

    await expect(api.syncIntegration('jellyfin')).resolves.toMatchObject({ serverId: 'jellyfin', status: 'completed' });
    await expect(api.integrationItems('jellyfin', false, 100)).resolves.toHaveLength(1);
    await expect(api.activityRollups(undefined, 250)).resolves.toHaveLength(1);
    await expect(api.pathMappings()).resolves.toHaveLength(1);
    await expect(api.unmappedPathItems(undefined, 50)).resolves.toHaveLength(1);
    await expect(api.upsertPathMapping({ id: 'map_1', serverPathPrefix: '/mnt/media', localPathPrefix: '/media' })).resolves.toMatchObject({ id: 'map_1' });
    await expect(api.verifyPathMapping('map_1')).resolves.toMatchObject({ matchedFiles: 1, verifiedFiles: 1 });

    expect(fetchMock).toHaveBeenNthCalledWith(1, '/api/v1/integrations/jellyfin/sync', expect.objectContaining({ method: 'POST' }));
    expect(fetchMock).toHaveBeenNthCalledWith(2, '/api/v1/integrations/jellyfin/items?limit=100', expect.any(Object));
    expect(fetchMock).toHaveBeenNthCalledWith(3, '/api/v1/activity/rollups?limit=250', expect.any(Object));
    expect(fetchMock).toHaveBeenNthCalledWith(4, '/api/v1/path-mappings', expect.any(Object));
    expect(fetchMock).toHaveBeenNthCalledWith(5, '/api/v1/path-mappings/unmapped?limit=50', expect.any(Object));
    expect(fetchMock).toHaveBeenNthCalledWith(6, '/api/v1/path-mappings/map_1', expect.objectContaining({ method: 'PUT' }));
    expect(fetchMock).toHaveBeenNthCalledWith(7, '/api/v1/path-mappings/map_1/verify', expect.objectContaining({ method: 'POST' }));
  });

  test('jobs endpoints expose active background progress', async () => {
    const fetchMock = vi.fn()
      .mockResolvedValueOnce(jsonResponse({ data: [{ id: 'job_1', kind: 'filesystem_scan', status: 'running', phase: 'processing' }] }))
      .mockResolvedValueOnce(jsonResponse({ data: { id: 'job_1', kind: 'filesystem_scan', status: 'running', events: [{ id: 'evt_1', message: 'Processed Arrival' }] } }))
      .mockResolvedValueOnce(jsonResponse({ data: { id: 'job_1', status: 'canceled' } }))
      .mockResolvedValueOnce(jsonResponse({ data: { id: 'job_2', status: 'queued' } }));
    vi.stubGlobal('fetch', fetchMock);

    await expect(api.jobs({ active: true })).resolves.toHaveLength(1);
    await expect(api.job('job_1')).resolves.toMatchObject({ id: 'job_1', events: [{ message: 'Processed Arrival' }] });
    await expect(api.cancelJob('job_1')).resolves.toMatchObject({ status: 'canceled' });
    await expect(api.retryJob('job_1')).resolves.toMatchObject({ id: 'job_2', status: 'queued' });

    expect(fetchMock).toHaveBeenNthCalledWith(1, '/api/v1/jobs?active=true', expect.any(Object));
    expect(fetchMock).toHaveBeenNthCalledWith(2, '/api/v1/jobs/job_1', expect.any(Object));
    expect(fetchMock).toHaveBeenNthCalledWith(3, '/api/v1/jobs/job_1/cancel', expect.objectContaining({ method: 'POST' }));
    expect(fetchMock).toHaveBeenNthCalledWith(4, '/api/v1/jobs/job_1/retry', expect.objectContaining({ method: 'POST' }));
  });

  test('integration settings calls redactable media server setting endpoints', async () => {
    const fetchMock = vi.fn()
      .mockResolvedValueOnce(jsonResponse({ data: [{ integration: 'jellyfin', baseUrl: 'http://jellyfin:8096', apiKeyConfigured: true, apiKeyLast4: 'abcd' }] }))
      .mockResolvedValueOnce(jsonResponse({ data: { integration: 'jellyfin', baseUrl: 'http://jellyfin:8096', apiKeyConfigured: true, apiKeyLast4: 'abcd' } }));
    vi.stubGlobal('fetch', fetchMock);

    await expect(api.integrationSettings()).resolves.toHaveLength(1);
    await api.updateIntegrationSetting('jellyfin', { baseUrl: 'http://jellyfin:8096', apiKey: 'secret-abcd', autoSyncEnabled: true, autoSyncIntervalMinutes: 360 });

    expect(fetchMock).toHaveBeenNthCalledWith(1, '/api/v1/integration-settings', expect.any(Object));
    expect(fetchMock).toHaveBeenNthCalledWith(2, '/api/v1/integration-settings/jellyfin', expect.objectContaining({
      method: 'PUT',
      body: JSON.stringify({ baseUrl: 'http://jellyfin:8096', apiKey: 'secret-abcd', autoSyncEnabled: true, autoSyncIntervalMinutes: 360 }),
    }));
  });

  test('integration diagnostics call returns ingestion proof summary', async () => {
    const fetchMock = vi.fn()
      .mockResolvedValueOnce(jsonResponse({
        data: {
          targetId: 'jellyfin',
          generatedAt: '2026-04-26T10:00:00Z',
          server: { name: 'Jellyfin', kind: 'jellyfin', status: 'configured' },
          summary: {
            movies: 10,
            series: 2,
            episodes: 40,
            files: 50,
            activityRollups: 42,
            recommendations: 3,
            serverReportedBytes: 1000,
            locallyVerifiedBytes: 750,
            unmappedFiles: 1,
          },
          warnings: ['local verification is incomplete'],
          progressSamples: [],
          topRecommendations: [],
        },
      }));
    vi.stubGlobal('fetch', fetchMock);

    await expect(api.integrationDiagnostics('jellyfin')).resolves.toMatchObject({
      targetId: 'jellyfin',
      summary: { movies: 10, recommendations: 3 },
    });

    expect(fetchMock).toHaveBeenCalledWith('/api/v1/integrations/jellyfin/diagnostics', expect.any(Object));
  });

  test('backup restore can inspect and restore a config archive', async () => {
    const fetchMock = vi.fn()
      .mockResolvedValueOnce(jsonResponse({ data: { entries: ['mediarr.db'] } }))
      .mockResolvedValueOnce(jsonResponse({ data: { preRestoreBackup: '/config/backups/pre.zip', restored: ['mediarr.db'] } }));
    vi.stubGlobal('fetch', fetchMock);

    await expect(api.restoreBackup('mediarr-20260426T120000.000000000Z.zip', true)).resolves.toMatchObject({ entries: ['mediarr.db'] });
    await expect(api.restoreBackup('mediarr-20260426T120000.000000000Z.zip', false)).resolves.toMatchObject({ restored: ['mediarr.db'] });

    expect(fetchMock).toHaveBeenNthCalledWith(1, '/api/v1/backups/restore', expect.objectContaining({
      method: 'POST',
      body: JSON.stringify({ name: 'mediarr-20260426T120000.000000000Z.zip', dryRun: true, confirmRestore: false }),
    }));
    expect(fetchMock).toHaveBeenNthCalledWith(2, '/api/v1/backups/restore', expect.objectContaining({
      method: 'POST',
      body: JSON.stringify({ name: 'mediarr-20260426T120000.000000000Z.zip', dryRun: false, confirmRestore: true }),
    }));
  });

  test('backup listing exposes downloadable archive names', async () => {
    const fetchMock = vi.fn()
      .mockResolvedValueOnce(jsonResponse({
        data: [{
          name: 'mediarr-20260426T120000.000000000Z.zip',
          path: '/config/backups/mediarr-20260426T120000.000000000Z.zip',
          sizeBytes: 4096,
          createdAt: '2026-04-26T12:00:00Z',
        }],
      }))
      .mockResolvedValueOnce(jsonResponse({
        data: {
          name: 'mediarr-20260426T120000.000000000Z.zip',
          path: '/config/backups/mediarr-20260426T120000.000000000Z.zip',
          sizeBytes: 4096,
          createdAt: '2026-04-26T12:00:00Z',
        },
      }));
    vi.stubGlobal('fetch', fetchMock);

    await expect(api.backups()).resolves.toEqual([expect.objectContaining({
      name: 'mediarr-20260426T120000.000000000Z.zip',
      sizeBytes: 4096,
    })]);
    await expect(api.createBackup()).resolves.toMatchObject({
      name: 'mediarr-20260426T120000.000000000Z.zip',
      path: '/config/backups/mediarr-20260426T120000.000000000Z.zip',
    });
    expect(api.backupDownloadUrl('mediarr-20260426T120000.000000000Z.zip')).toBe('/api/v1/backups/mediarr-20260426T120000.000000000Z.zip');
    expect(fetchMock).toHaveBeenNthCalledWith(1, '/api/v1/backups', expect.any(Object));
    expect(fetchMock).toHaveBeenNthCalledWith(2, '/api/v1/backups', expect.objectContaining({ method: 'POST' }));
  });

  test('support bundle endpoint creates a redacted diagnostics archive', async () => {
    const fetchMock = vi.fn()
      .mockResolvedValueOnce(jsonResponse({
        data: {
          name: 'mediarr-support-20260426T120000.000000000Z.zip',
          path: '/config/support/mediarr-support-20260426T120000.000000000Z.zip',
          sizeBytes: 2048,
          files: ['manifest.json', 'diagnostics/jellyfin.json'],
          createdAt: '2026-04-26T12:00:00Z',
        },
      }));
    vi.stubGlobal('fetch', fetchMock);

    await expect(api.createSupportBundle()).resolves.toMatchObject({
      name: 'mediarr-support-20260426T120000.000000000Z.zip',
      path: '/config/support/mediarr-support-20260426T120000.000000000Z.zip',
      files: ['manifest.json', 'diagnostics/jellyfin.json'],
    });

    expect(fetchMock).toHaveBeenCalledWith('/api/v1/support/bundles', expect.objectContaining({ method: 'POST' }));
  });

  test('support bundle listing exposes downloadable archive names', async () => {
    const fetchMock = vi.fn()
      .mockResolvedValueOnce(jsonResponse({
        data: [{
          name: 'mediarr-support-20260426T120000.000000000Z.zip',
          path: '/config/support/mediarr-support-20260426T120000.000000000Z.zip',
          sizeBytes: 2048,
          createdAt: '2026-04-26T12:00:00Z',
        }],
      }));
    vi.stubGlobal('fetch', fetchMock);

    await expect(api.supportBundles()).resolves.toEqual([expect.objectContaining({
      name: 'mediarr-support-20260426T120000.000000000Z.zip',
      sizeBytes: 2048,
    })]);
    expect(api.supportBundleDownloadUrl('mediarr-support-20260426T120000.000000000Z.zip')).toBe('/api/v1/support/bundles/mediarr-support-20260426T120000.000000000Z.zip');
    expect(fetchMock).toHaveBeenCalledWith('/api/v1/support/bundles', expect.any(Object));
  });
});

function jsonResponse(body: unknown): Response {
  return {
    ok: true,
    json: () => Promise.resolve(body),
  } as Response;
}
