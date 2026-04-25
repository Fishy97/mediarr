import { useEffect, useMemo, useState } from 'react';
import type React from 'react';
import {
  Activity,
  Archive,
  Bot,
  Check,
  Database,
  FolderOpen,
  Gauge,
  HardDrive,
  HeartPulse,
  KeyRound,
  Library,
  LogOut,
  Pencil,
  PlayCircle,
  RefreshCw,
  RotateCcw,
  Save,
  SearchCheck,
  Server,
  Settings,
  ShieldCheck,
  Trash2,
  UserRound,
} from 'lucide-react';
import { api, getAuthToken } from './lib/api';
import { formatBytes, formatConfidence } from './lib/format';
import type {
  AIStatus,
  AuthUser,
  CatalogCorrectionInput,
  CatalogItem,
  Integration,
  Library as MediaLibrary,
  ProviderHealth,
  ProviderSetting,
  ProviderSettingInput,
  Recommendation,
  ScanResult,
} from './types';

type View = 'dashboard' | 'libraries' | 'catalog' | 'recommendations' | 'integrations' | 'settings';

export function App() {
  const [view, setView] = useState<View>('dashboard');
  const [libraries, setLibraries] = useState<MediaLibrary[]>([]);
  const [scans, setScans] = useState<ScanResult[]>([]);
  const [catalog, setCatalog] = useState<CatalogItem[]>([]);
  const [recommendations, setRecommendations] = useState<Recommendation[]>([]);
  const [providers, setProviders] = useState<ProviderHealth[]>([]);
  const [providerSettings, setProviderSettings] = useState<ProviderSetting[]>([]);
  const [integrations, setIntegrations] = useState<Integration[]>([]);
  const [aiStatus, setAIStatus] = useState<AIStatus | null>(null);
  const [status, setStatus] = useState('Loading');
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  const [authChecked, setAuthChecked] = useState(false);
  const [setupRequired, setSetupRequired] = useState(false);
  const [user, setUser] = useState<AuthUser | null>(null);

  async function refresh() {
    try {
      const [health, libs, catalogRows, scanRows, recs, providerRows, providerSettingRows, integrationRows, ai] = await Promise.all([
        api.health(),
        api.libraries(),
        api.catalog(),
        api.scans(),
        api.recommendations(),
        api.providers(),
        api.providerSettings(),
        api.integrations(),
        api.aiStatus(),
      ]);
      setStatus(health.status);
      setLibraries(libs);
      setCatalog(catalogRows);
      setScans(scanRows);
      setRecommendations(recs);
      setProviders(providerRows);
      setProviderSettings(providerSettingRows);
      setIntegrations(integrationRows);
      setAIStatus(ai);
      setError(null);
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : 'Unable to load Mediarr');
    }
  }

  useEffect(() => {
    void bootstrap();
  }, []);

  async function bootstrap() {
    try {
      const setup = await api.setupStatus();
      setSetupRequired(setup.setupRequired);
      if (setup.setupRequired) {
        setAuthChecked(true);
        return;
      }
      if (getAuthToken()) {
        const currentUser = await api.me();
        setUser(currentUser);
        setAuthChecked(true);
        await refresh();
        return;
      }
      setAuthChecked(true);
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : 'Unable to load Mediarr');
      setAuthChecked(true);
    }
  }

  async function startScan() {
    setBusy(true);
    try {
      const result = await api.startScan();
      setScans((current) => [...current, ...result.scans]);
      setRecommendations(result.recommendations);
      setCatalog(result.scans.flatMap((scan) => scan.items).map(toCatalogItem));
      setError(null);
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : 'Scan failed');
    } finally {
      setBusy(false);
    }
  }

  async function createBackup() {
    setBusy(true);
    try {
      await api.createBackup();
      setError(null);
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : 'Backup failed');
    } finally {
      setBusy(false);
    }
  }

  async function ignoreRecommendation(id: string) {
    setBusy(true);
    try {
      await api.ignoreRecommendation(id);
      setRecommendations((current) => current.filter((rec) => rec.id !== id));
      setError(null);
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : 'Unable to ignore recommendation');
    } finally {
      setBusy(false);
    }
  }

  async function updateProviderSetting(provider: string, setting: ProviderSettingInput) {
    setBusy(true);
    try {
      await api.updateProviderSetting(provider, setting);
      const [settings, health] = await Promise.all([api.providerSettings(), api.providers()]);
      setProviderSettings(settings);
      setProviders(health);
      setError(null);
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : 'Unable to update provider settings');
    } finally {
      setBusy(false);
    }
  }

  async function correctCatalogItem(id: string, correction: CatalogCorrectionInput) {
    setBusy(true);
    try {
      await api.correctCatalogItem(id, correction);
      const [items, recs] = await Promise.all([api.catalog(), api.recommendations()]);
      setCatalog(items);
      setRecommendations(recs);
      setError(null);
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : 'Unable to save catalog correction');
    } finally {
      setBusy(false);
    }
  }

  async function clearCatalogCorrection(id: string) {
    setBusy(true);
    try {
      await api.clearCatalogCorrection(id);
      const [items, recs] = await Promise.all([api.catalog(), api.recommendations()]);
      setCatalog(items);
      setRecommendations(recs);
      setError(null);
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : 'Unable to clear catalog correction');
    } finally {
      setBusy(false);
    }
  }

  async function refreshIntegration(id: string) {
    setBusy(true);
    try {
      await api.refreshIntegration(id);
      setIntegrations(await api.integrations());
      setError(null);
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : 'Unable to refresh integration');
    } finally {
      setBusy(false);
    }
  }

  const catalogItems = useMemo(() => catalog, [catalog]);
  const totalFiles = catalogItems.length;
  const totalSize = catalogItems.reduce((sum, item) => sum + item.sizeBytes, 0);
  const recoverable = recommendations.reduce((sum, rec) => sum + rec.spaceSavedBytes, 0);

  if (!authChecked) {
    return <LoadingScreen />;
  }

  if (setupRequired) {
    return <AuthScreen mode="setup" onAuthenticated={(nextUser) => {
      setUser(nextUser);
      setSetupRequired(false);
      void refresh();
    }} />;
  }

  if (!user) {
    return <AuthScreen mode="login" onAuthenticated={(nextUser) => {
      setUser(nextUser);
      void refresh();
    }} />;
  }

  async function logout() {
    await api.logout();
    setUser(null);
  }

  return (
    <div className="app-shell">
      <aside className="sidebar">
        <div className="brand">
          <div className="brand-mark"><SearchCheck size={20} /></div>
          <div>
            <strong>Mediarr</strong>
            <span>Library control plane</span>
          </div>
        </div>
        <nav className="nav">
          <NavButton icon={<Gauge />} label="Dashboard" active={view === 'dashboard'} onClick={() => setView('dashboard')} />
          <NavButton icon={<FolderOpen />} label="Libraries" active={view === 'libraries'} onClick={() => setView('libraries')} />
          <NavButton icon={<Library />} label="Catalog" active={view === 'catalog'} onClick={() => setView('catalog')} />
          <NavButton icon={<Trash2 />} label="Review Queue" active={view === 'recommendations'} onClick={() => setView('recommendations')} />
          <NavButton icon={<Server />} label="Integrations" active={view === 'integrations'} onClick={() => setView('integrations')} />
          <NavButton icon={<Settings />} label="Settings" active={view === 'settings'} onClick={() => setView('settings')} />
        </nav>
      </aside>

      <main className="main">
        <header className="topbar">
          <div>
            <p className="eyebrow">Self-hosted • suggest-only cleanup • read-only media mounts</p>
            <h1>{titleFor(view)}</h1>
          </div>
          <div className="actions">
            <span className="user-chip"><UserRound size={16} />{user.email}</span>
            <button className="icon-button" onClick={() => void refresh()} title="Refresh data" aria-label="Refresh data">
              <RefreshCw size={18} />
            </button>
            <button className="icon-button" onClick={() => void logout()} title="Log out" aria-label="Log out">
              <LogOut size={18} />
            </button>
            <button className="primary-button" onClick={() => void startScan()} disabled={busy}>
              <Activity size={18} />
              {busy ? 'Working' : 'Scan now'}
            </button>
          </div>
        </header>

        {error && <div className="notice error">{error}</div>}

        {view === 'dashboard' && (
          <Dashboard
            status={status}
            libraries={libraries}
            totalFiles={totalFiles}
            totalSize={totalSize}
            recoverable={recoverable}
            recommendations={recommendations}
            scans={scans}
          />
        )}
        {view === 'libraries' && <Libraries libraries={libraries} scans={scans} />}
        {view === 'catalog' && <Catalog items={catalogItems} onCorrect={(id, correction) => void correctCatalogItem(id, correction)} onClear={(id) => void clearCatalogCorrection(id)} busy={busy} />}
        {view === 'recommendations' && <RecommendationQueue recommendations={recommendations} onIgnore={(id) => void ignoreRecommendation(id)} busy={busy} />}
        {view === 'integrations' && (
          <Integrations
            providers={providers}
            providerSettings={providerSettings}
            integrations={integrations}
            aiStatus={aiStatus}
            onProviderUpdate={(provider, setting) => void updateProviderSetting(provider, setting)}
            onIntegrationRefresh={(id) => void refreshIntegration(id)}
            busy={busy}
          />
        )}
        {view === 'settings' && <SettingsView onBackup={() => void createBackup()} busy={busy} />}
      </main>
    </div>
  );
}

function LoadingScreen() {
  return (
    <main className="auth-shell">
      <section className="auth-panel">
        <div className="brand-mark"><SearchCheck size={20} /></div>
        <h1>Mediarr</h1>
        <p>Loading local control plane.</p>
      </section>
    </main>
  );
}

function AuthScreen({ mode, onAuthenticated }: { mode: 'setup' | 'login'; onAuthenticated: (user: AuthUser) => void }) {
  const [email, setEmail] = useState('admin@mediarr.local');
  const [password, setPassword] = useState('');
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  async function submit(event: React.FormEvent) {
    event.preventDefault();
    setBusy(true);
    try {
      const result = mode === 'setup'
        ? await api.setupAdmin(email, password)
        : await api.login(email, password);
      setError(null);
      onAuthenticated(result.user);
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : 'Authentication failed');
    } finally {
      setBusy(false);
    }
  }

  return (
    <main className="auth-shell">
      <form className="auth-panel" onSubmit={(event) => void submit(event)}>
        <div className="brand-mark"><SearchCheck size={20} /></div>
        <p className="eyebrow">{mode === 'setup' ? 'First run setup' : 'Admin sign in'}</p>
        <h1>{mode === 'setup' ? 'Create the local admin account' : 'Welcome back'}</h1>
        <label>
          Email
          <input value={email} onChange={(event) => setEmail(event.target.value)} type="email" autoComplete="username" required />
        </label>
        <label>
          Password
          <input value={password} onChange={(event) => setPassword(event.target.value)} type="password" autoComplete={mode === 'setup' ? 'new-password' : 'current-password'} minLength={12} required />
        </label>
        {error && <div className="notice error">{error}</div>}
        <button className="primary-button" disabled={busy} type="submit">
          <ShieldCheck size={18} />
          {busy ? 'Working' : mode === 'setup' ? 'Create admin' : 'Sign in'}
        </button>
      </form>
    </main>
  );
}

function Dashboard({
  status,
  libraries,
  totalFiles,
  totalSize,
  recoverable,
  recommendations,
  scans,
}: {
  status: string;
  libraries: MediaLibrary[];
  totalFiles: number;
  totalSize: number;
  recoverable: number;
  recommendations: Recommendation[];
  scans: ScanResult[];
}) {
  const lastScan = scans.at(-1);
  return (
    <section className="view-grid">
      <div className="stat-strip">
        <Stat icon={<HeartPulse />} label="System" value={status.toUpperCase()} />
        <Stat icon={<FolderOpen />} label="Libraries" value={String(libraries.length)} />
        <Stat icon={<Database />} label="Indexed Files" value={String(totalFiles)} />
        <Stat icon={<HardDrive />} label="Indexed Size" value={formatBytes(totalSize)} />
        <Stat icon={<Archive />} label="Review Savings" value={formatBytes(recoverable)} />
      </div>
      <div className="split">
        <section className="panel">
          <div className="panel-heading">
            <h2>Recent Scan</h2>
            <span>{lastScan ? new Date(lastScan.completedAt).toLocaleString() : 'No scan yet'}</span>
          </div>
          <div className="scan-summary">
            <strong>{lastScan?.filesScanned ?? 0}</strong>
            <span>files indexed in the latest scan</span>
          </div>
        </section>
        <section className="panel">
          <div className="panel-heading">
            <h2>Top Recommendations</h2>
            <span>{recommendations.length} open</span>
          </div>
          <CompactRecommendations recommendations={recommendations.slice(0, 4)} />
        </section>
      </div>
    </section>
  );
}

function Libraries({ libraries, scans }: { libraries: MediaLibrary[]; scans: ScanResult[] }) {
  return (
    <section className="table-panel">
      <table>
        <thead>
          <tr>
            <th>Name</th>
            <th>Kind</th>
            <th>Root</th>
            <th>Last Files</th>
          </tr>
        </thead>
        <tbody>
          {libraries.map((library) => {
            const scan = [...scans].reverse().find((row) => row.libraryId === library.id);
            return (
              <tr key={library.id}>
                <td>{library.name || library.id}</td>
                <td>{library.kind}</td>
                <td className="path-cell">{library.root}</td>
                <td>{scan?.filesScanned ?? 0}</td>
              </tr>
            );
          })}
        </tbody>
      </table>
    </section>
  );
}

function Catalog({
  items,
  onCorrect,
  onClear,
  busy,
}: {
  items: CatalogItem[];
  onCorrect: (id: string, correction: CatalogCorrectionInput) => void;
  onClear: (id: string) => void;
  busy: boolean;
}) {
  const [selected, setSelected] = useState<CatalogItem | null>(null);
  return (
    <section className="view-grid">
      <section className="table-panel">
        <table>
          <thead>
            <tr>
              <th>Title</th>
              <th>Kind</th>
              <th>Year</th>
              <th>Quality</th>
              <th>Size</th>
              <th>Subtitles</th>
              <th>Metadata</th>
              <th>Path</th>
              <th>Actions</th>
            </tr>
          </thead>
          <tbody>
            {items.map((item) => (
              <tr key={item.id}>
                <td>{item.title}</td>
                <td>{item.kind}</td>
                <td>{item.year || 'unknown'}</td>
                <td>{item.quality || 'unknown'}</td>
                <td>{formatBytes(item.sizeBytes)}</td>
                <td>{item.subtitles.length}</td>
                <td>{item.metadataCorrected ? `${item.metadataProvider || 'manual'} ${item.metadataProviderId || ''}`.trim() : 'scan'}</td>
                <td className="path-cell">{item.path}</td>
                <td>
                  <button className="icon-button" title="Correct metadata" aria-label="Correct metadata" onClick={() => setSelected(item)}>
                    <Pencil size={16} />
                  </button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
        {items.length === 0 && <EmptyState icon={<PlayCircle />} text="Run a scan to populate the catalog." />}
      </section>
      {selected && (
        <CatalogCorrectionPanel
          item={selected}
          busy={busy}
          onClose={() => setSelected(null)}
          onCorrect={(correction) => {
            onCorrect(selected.id, correction);
            setSelected(null);
          }}
          onClear={() => {
            onClear(selected.id);
            setSelected(null);
          }}
        />
      )}
    </section>
  );
}

function CatalogCorrectionPanel({
  item,
  busy,
  onCorrect,
  onClear,
  onClose,
}: {
  item: CatalogItem;
  busy: boolean;
  onCorrect: (correction: CatalogCorrectionInput) => void;
  onClear: () => void;
  onClose: () => void;
}) {
  const [title, setTitle] = useState(item.title);
  const [kind, setKind] = useState(item.kind);
  const [year, setYear] = useState(item.year ? String(item.year) : '');
  const [provider, setProvider] = useState(item.metadataProvider || '');
  const [providerId, setProviderID] = useState(item.metadataProviderId || '');
  const [confidence, setConfidence] = useState(item.metadataConfidence ? String(item.metadataConfidence) : '1');

  useEffect(() => {
    setTitle(item.title);
    setKind(item.kind);
    setYear(item.year ? String(item.year) : '');
    setProvider(item.metadataProvider || '');
    setProviderID(item.metadataProviderId || '');
    setConfidence(item.metadataConfidence ? String(item.metadataConfidence) : '1');
  }, [item]);

  function submit(event: React.FormEvent) {
    event.preventDefault();
    const parsedYear = Number.parseInt(year, 10);
    const parsedConfidence = Number.parseFloat(confidence);
    onCorrect({
      title,
      kind,
      ...(Number.isFinite(parsedYear) && parsedYear > 0 ? { year: parsedYear } : {}),
      ...(provider.trim() ? { provider: provider.trim() } : {}),
      ...(providerId.trim() ? { providerId: providerId.trim() } : {}),
      confidence: Number.isFinite(parsedConfidence) ? parsedConfidence : 1,
    });
  }

  return (
    <form className="panel form-panel" onSubmit={(event) => void submit(event)}>
      <div className="panel-heading">
        <h2>{item.path}</h2>
        <button className="icon-button" type="button" onClick={onClose} title="Close" aria-label="Close">
          <Check size={16} />
        </button>
      </div>
      <div className="form-grid">
        <label>
          Title
          <input value={title} onChange={(event) => setTitle(event.target.value)} required />
        </label>
        <label>
          Kind
          <select value={kind} onChange={(event) => setKind(event.target.value)}>
            <option value="movie">movie</option>
            <option value="series">series</option>
            <option value="anime">anime</option>
            <option value="unknown">unknown</option>
          </select>
        </label>
        <label>
          Year
          <input value={year} onChange={(event) => setYear(event.target.value)} inputMode="numeric" />
        </label>
        <label>
          Provider
          <select value={provider} onChange={(event) => setProvider(event.target.value)}>
            <option value="">manual</option>
            <option value="tmdb">tmdb</option>
            <option value="thetvdb">thetvdb</option>
            <option value="anilist">anilist</option>
            <option value="local-sidecar">local-sidecar</option>
          </select>
        </label>
        <label>
          Provider ID
          <input value={providerId} onChange={(event) => setProviderID(event.target.value)} />
        </label>
        <label>
          Confidence
          <input value={confidence} onChange={(event) => setConfidence(event.target.value)} inputMode="decimal" />
        </label>
      </div>
      <div className="button-row">
        <button className="primary-button" type="submit" disabled={busy}>
          <Save size={18} />
          Save correction
        </button>
        {item.metadataCorrected && (
          <button className="secondary-button" type="button" disabled={busy} onClick={onClear}>
            <RotateCcw size={16} />
            Clear correction
          </button>
        )}
      </div>
    </form>
  );
}

function RecommendationQueue({ recommendations, onIgnore, busy }: { recommendations: Recommendation[]; onIgnore: (id: string) => void; busy: boolean }) {
  return (
    <section className="queue">
      {recommendations.map((rec) => (
        <article className="recommendation-card" key={rec.id}>
          <div className="rec-main">
            <div className="rec-icon"><ShieldCheck size={22} /></div>
            <div>
              <h2>{rec.title}</h2>
              <p>{rec.explanation}</p>
              <div className="path-list">
                {rec.affectedPaths.map((path) => <code key={path}>{path}</code>)}
              </div>
              {rec.aiRationale && (
                <div className="ai-note">
                  <Bot size={16} />
                  <span>{rec.aiRationale}</span>
                </div>
              )}
            </div>
          </div>
          <div className="rec-metrics">
            <span>{formatBytes(rec.spaceSavedBytes)}</span>
            <small>{formatConfidence(rec.confidence)} • {rec.source}</small>
            <button className="secondary-button" disabled={busy} onClick={() => onIgnore(rec.id)}>Ignore</button>
          </div>
        </article>
      ))}
      {recommendations.length === 0 && <EmptyState icon={<Trash2 />} text="No cleanup recommendations are open." />}
    </section>
  );
}

function Integrations({
  providers,
  providerSettings,
  integrations,
  aiStatus,
  onProviderUpdate,
  onIntegrationRefresh,
  busy,
}: {
  providers: ProviderHealth[];
  providerSettings: ProviderSetting[];
  integrations: Integration[];
  aiStatus: AIStatus | null;
  onProviderUpdate: (provider: string, setting: ProviderSettingInput) => void;
  onIntegrationRefresh: (id: string) => void;
  busy: boolean;
}) {
  return (
    <section className="view-grid">
      <div className="grid-list">
        {integrations.map((integration) => (
          <article className="status-card" key={integration.id}>
            <Bot size={20} />
            <div>
              <h2>{integration.name}</h2>
              <p>{integration.description}</p>
            </div>
            <span>{integration.status}</span>
            {integration.kind === 'media_server' && (
              <button className="icon-button" onClick={() => onIntegrationRefresh(integration.id)} disabled={busy} title="Refresh media server" aria-label={`Refresh ${integration.name}`}>
                <RefreshCw size={16} />
              </button>
            )}
          </article>
        ))}
        {aiStatus && (
          <article className="status-card">
            <Bot size={20} />
            <div>
              <h2>Local AI</h2>
              <p>{aiStatus.model || 'No model configured'}</p>
            </div>
            <span>{aiStatus.status}</span>
          </article>
        )}
      </div>
      <section className="table-panel">
        <table>
          <thead>
            <tr>
              <th>Provider</th>
              <th>Status</th>
              <th>Attribution</th>
              <th>Rate Limit</th>
            </tr>
          </thead>
          <tbody>
            {providers.map((provider) => (
              <tr key={provider.name}>
                <td>{provider.name}</td>
                <td>{provider.status}</td>
                <td>{provider.attribution}</td>
                <td>{provider.rateLimit || 'configured by provider'}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </section>
      <ProviderSettingsPanel settings={providerSettings} busy={busy} onUpdate={onProviderUpdate} />
    </section>
  );
}

function ProviderSettingsPanel({
  settings,
  busy,
  onUpdate,
}: {
  settings: ProviderSetting[];
  busy: boolean;
  onUpdate: (provider: string, setting: ProviderSettingInput) => void;
}) {
  const byProvider = new Map(settings.map((setting) => [setting.provider, setting]));
  return (
    <section className="settings-layout provider-settings">
      {['tmdb', 'thetvdb', 'opensubtitles'].map((provider) => (
        <ProviderSettingForm
          key={provider}
          provider={provider}
          setting={byProvider.get(provider)}
          busy={busy}
          onUpdate={(input) => onUpdate(provider, input)}
        />
      ))}
    </section>
  );
}

function ProviderSettingForm({
  provider,
  setting,
  busy,
  onUpdate,
}: {
  provider: string;
  setting?: ProviderSetting;
  busy: boolean;
  onUpdate: (setting: ProviderSettingInput) => void;
}) {
  const [baseUrl, setBaseURL] = useState(setting?.baseUrl || '');
  const [apiKey, setAPIKey] = useState('');

  useEffect(() => {
    setBaseURL(setting?.baseUrl || '');
    setAPIKey('');
  }, [setting]);

  function submit(event: React.FormEvent) {
    event.preventDefault();
    onUpdate({
      ...(baseUrl.trim() ? { baseUrl: baseUrl.trim() } : {}),
      ...(apiKey.trim() ? { apiKey: apiKey.trim() } : {}),
    });
  }

  return (
    <form className="panel form-panel" onSubmit={(event) => void submit(event)}>
      <div className="panel-heading">
        <h2>{provider}</h2>
        <span>{setting?.apiKeyConfigured ? `key ends ${setting.apiKeyLast4 || 'set'}` : 'not configured'}</span>
      </div>
      <label>
        Base URL
        <input value={baseUrl} onChange={(event) => setBaseURL(event.target.value)} placeholder="provider default" />
      </label>
      <label>
        API key
        <input value={apiKey} onChange={(event) => setAPIKey(event.target.value)} type="password" autoComplete="off" placeholder={setting?.apiKeyConfigured ? 'configured' : ''} />
      </label>
      <div className="button-row">
        <button className="primary-button" type="submit" disabled={busy}>
          <KeyRound size={18} />
          Save
        </button>
        {setting?.apiKeyConfigured && (
          <button className="secondary-button" type="button" disabled={busy} onClick={() => onUpdate({ clearApiKey: true })}>
            <RotateCcw size={16} />
            Clear key
          </button>
        )}
      </div>
    </form>
  );
}

function SettingsView({ onBackup, busy }: { onBackup: () => void; busy: boolean }) {
  return (
    <section className="settings-layout">
      <div className="panel">
        <div className="panel-heading">
          <h2>Backups</h2>
          <span>/config/backups</span>
        </div>
        <button className="primary-button" onClick={onBackup} disabled={busy}>
          <Archive size={18} />
          Create backup
        </button>
      </div>
      <div className="panel">
        <div className="panel-heading">
          <h2>Safety</h2>
          <span>enforced</span>
        </div>
        <ul className="plain-list">
          <li>No permanent delete endpoint</li>
          <li>Recommendations are advisory</li>
          <li>Media mounts are read-only by default</li>
        </ul>
      </div>
    </section>
  );
}

function CompactRecommendations({ recommendations }: { recommendations: Recommendation[] }) {
  if (recommendations.length === 0) {
    return <EmptyState icon={<Trash2 />} text="No recommendations yet." />;
  }
  return (
    <div className="compact-list">
      {recommendations.map((rec) => (
        <div className="compact-row" key={rec.id}>
          <span>{rec.title}</span>
          <strong>{formatBytes(rec.spaceSavedBytes)}</strong>
        </div>
      ))}
    </div>
  );
}

function Stat({ icon, label, value }: { icon: React.ReactNode; label: string; value: string }) {
  return (
    <div className="stat">
      {icon}
      <span>{label}</span>
      <strong>{value}</strong>
    </div>
  );
}

function NavButton({ icon, label, active, onClick }: { icon: React.ReactElement; label: string; active: boolean; onClick: () => void }) {
  return (
    <button className={active ? 'nav-button active' : 'nav-button'} onClick={onClick}>
      {icon}
      <span>{label}</span>
    </button>
  );
}

function EmptyState({ icon, text }: { icon: React.ReactNode; text: string }) {
  return (
    <div className="empty-state">
      {icon}
      <span>{text}</span>
    </div>
  );
}

function titleFor(view: View): string {
  return {
    dashboard: 'Dashboard',
    libraries: 'Libraries',
    catalog: 'Catalog',
    recommendations: 'Review Queue',
    integrations: 'Integrations',
    settings: 'Settings',
  }[view];
}

function toCatalogItem(item: ScanResult['items'][number]): CatalogItem {
  return {
    id: item.id,
    libraryId: '',
    path: item.path,
    canonicalKey: item.parsed.canonicalKey,
    title: item.parsed.title,
    kind: item.parsed.kind,
    year: item.parsed.year,
    sizeBytes: item.sizeBytes,
    quality: item.parsed.quality,
    fingerprint: '',
    subtitles: item.subtitles,
    metadataCorrected: false,
    modifiedAt: '',
    scannedAt: new Date().toISOString(),
  };
}
