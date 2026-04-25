import type { AuthResponse, AuthUser, CatalogItem, Integration, Library, ProviderHealth, Recommendation, ScanResult, SetupStatus } from '../types';

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
  async startScan(): Promise<{ scans: ScanResult[]; recommendations: Recommendation[] }> {
    return (await request<Envelope<{ scans: ScanResult[]; recommendations: Recommendation[] }>>('/api/v1/scans', {
      method: 'POST',
    })).data;
  },
  async recommendations(): Promise<Recommendation[]> {
    return (await request<Envelope<Recommendation[]>>('/api/v1/recommendations')).data;
  },
  async providers(): Promise<ProviderHealth[]> {
    return (await request<Envelope<ProviderHealth[]>>('/api/v1/providers')).data;
  },
  async integrations(): Promise<Integration[]> {
    return (await request<Envelope<Integration[]>>('/api/v1/integrations')).data;
  },
  async createBackup(): Promise<{ path: string }> {
    return (await request<Envelope<{ path: string }>>('/api/v1/backups', { method: 'POST' })).data;
  },
};
