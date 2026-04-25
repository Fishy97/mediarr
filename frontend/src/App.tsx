import { useEffect, useMemo, useState } from 'react';
import type React from 'react';
import {
  Activity,
  Archive,
  Bot,
  Database,
  FolderOpen,
  Gauge,
  HardDrive,
  HeartPulse,
  Library,
  LogOut,
  PlayCircle,
  RefreshCw,
  SearchCheck,
  Server,
  Settings,
  ShieldCheck,
  Trash2,
  UserRound,
} from 'lucide-react';
import { api, getAuthToken } from './lib/api';
import { formatBytes, formatConfidence } from './lib/format';
import type { AuthUser, CatalogItem, Integration, Library as MediaLibrary, ProviderHealth, Recommendation, ScanResult } from './types';

type View = 'dashboard' | 'libraries' | 'catalog' | 'recommendations' | 'integrations' | 'settings';

export function App() {
  const [view, setView] = useState<View>('dashboard');
  const [libraries, setLibraries] = useState<MediaLibrary[]>([]);
  const [scans, setScans] = useState<ScanResult[]>([]);
  const [catalog, setCatalog] = useState<CatalogItem[]>([]);
  const [recommendations, setRecommendations] = useState<Recommendation[]>([]);
  const [providers, setProviders] = useState<ProviderHealth[]>([]);
  const [integrations, setIntegrations] = useState<Integration[]>([]);
  const [status, setStatus] = useState('Loading');
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  const [authChecked, setAuthChecked] = useState(false);
  const [setupRequired, setSetupRequired] = useState(false);
  const [user, setUser] = useState<AuthUser | null>(null);

  async function refresh() {
    try {
      const [health, libs, catalogRows, scanRows, recs, providerRows, integrationRows] = await Promise.all([
        api.health(),
        api.libraries(),
        api.catalog(),
        api.scans(),
        api.recommendations(),
        api.providers(),
        api.integrations(),
      ]);
      setStatus(health.status);
      setLibraries(libs);
      setCatalog(catalogRows);
      setScans(scanRows);
      setRecommendations(recs);
      setProviders(providerRows);
      setIntegrations(integrationRows);
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
        {view === 'catalog' && <Catalog items={catalogItems} />}
        {view === 'recommendations' && <RecommendationQueue recommendations={recommendations} />}
        {view === 'integrations' && <Integrations providers={providers} integrations={integrations} />}
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

function Catalog({ items }: { items: CatalogItem[] }) {
  return (
    <section className="table-panel">
      <table>
        <thead>
          <tr>
            <th>Title</th>
            <th>Kind</th>
            <th>Quality</th>
            <th>Size</th>
            <th>Subtitles</th>
            <th>Path</th>
          </tr>
        </thead>
        <tbody>
          {items.map((item) => (
            <tr key={item.id}>
              <td>{item.title}</td>
              <td>{item.kind}</td>
              <td>{item.quality || 'unknown'}</td>
              <td>{formatBytes(item.sizeBytes)}</td>
              <td>{item.subtitles.length}</td>
              <td className="path-cell">{item.path}</td>
            </tr>
          ))}
        </tbody>
      </table>
      {items.length === 0 && <EmptyState icon={<PlayCircle />} text="Run a scan to populate the catalog." />}
    </section>
  );
}

function RecommendationQueue({ recommendations }: { recommendations: Recommendation[] }) {
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
            </div>
          </div>
          <div className="rec-metrics">
            <span>{formatBytes(rec.spaceSavedBytes)}</span>
            <small>{formatConfidence(rec.confidence)} • {rec.source}</small>
          </div>
        </article>
      ))}
      {recommendations.length === 0 && <EmptyState icon={<Trash2 />} text="No cleanup recommendations are open." />}
    </section>
  );
}

function Integrations({ providers, integrations }: { providers: ProviderHealth[]; integrations: Integration[] }) {
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
          </article>
        ))}
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
    </section>
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
    sizeBytes: item.sizeBytes,
    quality: item.parsed.quality,
    fingerprint: '',
    subtitles: item.subtitles,
    modifiedAt: '',
    scannedAt: new Date().toISOString(),
  };
}
