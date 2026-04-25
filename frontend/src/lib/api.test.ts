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
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse({ data: { ok: true } }));
    vi.stubGlobal('fetch', fetchMock);

    await api.ignoreRecommendation('rec_1');
    await api.restoreRecommendation('rec_1');

    expect(fetchMock).toHaveBeenNthCalledWith(1, '/api/v1/recommendations/rec_1/ignore', expect.objectContaining({ method: 'POST' }));
    expect(fetchMock).toHaveBeenNthCalledWith(2, '/api/v1/recommendations/rec_1/restore', expect.objectContaining({ method: 'POST' }));
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
});

function jsonResponse(body: unknown): Response {
  return {
    ok: true,
    json: () => Promise.resolve(body),
  } as Response;
}
