import type { Integration, Library, ProviderHealth, Recommendation, ScanResult } from '../types';

type Envelope<T> = { data: T };

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const response = await fetch(path, {
    ...init,
    headers: {
      'Content-Type': 'application/json',
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
  async libraries(): Promise<Library[]> {
    return (await request<Envelope<Library[]>>('/api/v1/libraries')).data;
  },
  async scans(): Promise<ScanResult[]> {
    return (await request<Envelope<ScanResult[]>>('/api/v1/scans')).data;
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

