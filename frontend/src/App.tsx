import { useEffect, useMemo, useRef, useState } from 'react';
import type React from 'react';
import {
  Activity,
  Archive,
  Bot,
  Check,
  Database,
  Download,
  FolderOpen,
  Gauge,
  HardDrive,
  HeartPulse,
  KeyRound,
  Library,
  LogOut,
  Map as MapIcon,
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
  ActivityRollup,
  AuthUser,
  CatalogCorrectionInput,
  CatalogItem,
  Integration,
  IntegrationDiagnostics,
  IntegrationSyncJob,
  IntegrationSetting,
  IntegrationSettingInput,
  Job,
  JobDetail,
  JobEvent,
  Library as MediaLibrary,
  MediaServerItem,
  PathMapping,
  ProviderHealth,
  ProviderSetting,
  ProviderSettingInput,
  Recommendation,
  RecommendationEvidence,
  ScanResult,
  SupportBundle,
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
  const [integrationSettings, setIntegrationSettings] = useState<IntegrationSetting[]>([]);
  const [integrations, setIntegrations] = useState<Integration[]>([]);
  const [syncJobs, setSyncJobs] = useState<Record<string, IntegrationSyncJob | null>>({});
  const [integrationDiagnostics, setIntegrationDiagnostics] = useState<Record<string, IntegrationDiagnostics | null>>({});
  const [integrationItems, setIntegrationItems] = useState<MediaServerItem[]>([]);
  const [activityRollups, setActivityRollups] = useState<ActivityRollup[]>([]);
  const [pathMappings, setPathMappings] = useState<PathMapping[]>([]);
  const [unmappedItems, setUnmappedItems] = useState<MediaServerItem[]>([]);
  const [jobs, setJobs] = useState<Job[]>([]);
  const [jobDetails, setJobDetails] = useState<Record<string, JobDetail>>({});
  const [recommendationEvidence, setRecommendationEvidence] = useState<Record<string, RecommendationEvidence>>({});
  const [aiStatus, setAIStatus] = useState<AIStatus | null>(null);
  const [status, setStatus] = useState('Loading');
  const [error, setError] = useState<string | null>(null);
  const [backupNotice, setBackupNotice] = useState<string | null>(null);
  const [supportBundles, setSupportBundles] = useState<SupportBundle[]>([]);
  const [busy, setBusy] = useState(false);
  const [authChecked, setAuthChecked] = useState(false);
  const [setupRequired, setSetupRequired] = useState(false);
  const [user, setUser] = useState<AuthUser | null>(null);
  const activeJobIds = useRef<Set<string>>(new Set());

  async function refresh() {
    try {
      const [health, libs, catalogRows, scanRows, recs, providerRows, providerSettingRows, integrationSettingRows, integrationRows, ai, bundleRows] = await Promise.all([
        api.health(),
        api.libraries(),
        api.catalog(),
        api.scans(),
        api.recommendations(),
        api.providers(),
        api.providerSettings(),
        api.integrationSettings(),
        api.integrations(),
        api.aiStatus(),
        api.supportBundles(),
      ]);
      setStatus(health.status);
      setLibraries(libs);
      setCatalog(catalogRows);
      setScans(scanRows);
      setRecommendations(recs);
      setProviders(providerRows);
      setProviderSettings(providerSettingRows);
      setIntegrationSettings(integrationSettingRows);
      setIntegrations(integrationRows);
      setAIStatus(ai);
      setSupportBundles(bundleRows);
      await refreshIntegrationActivity(integrationRows);
      setError(null);
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : 'Unable to load Mediarr');
    }
  }

  async function refreshJobs() {
    if (!user) {
      return;
    }
    try {
      const jobRows = await api.jobs({ limit: 12 });
      const activeRows = jobRows.filter(isActiveJob);
      const previousActive = activeJobIds.current;
      const completedSinceLastPoll = jobRows.some((job) => previousActive.has(job.id) && !isActiveJob(job));
      activeJobIds.current = new Set(activeRows.map((job) => job.id));
      setJobs(jobRows);

      const details = await Promise.all(jobRows.slice(0, 8).map((job) => api.job(job.id).catch(() => null)));
      setJobDetails((current) => {
        const next = { ...current };
        details.forEach((detail) => {
          if (detail) {
            next[detail.id] = detail;
          }
        });
        return next;
      });

      if (completedSinceLastPoll) {
        await refresh();
      } else if (activeRows.some((job) => job.kind.endsWith('_sync'))) {
        await refreshIntegrationActivity();
      }
    } catch {
      // Job polling is advisory UI telemetry; the main refresh path surfaces hard failures.
    }
  }

  async function refreshIntegrationActivity(integrationRows = integrations) {
    const mediaServers = integrationRows.filter((integration) => integration.kind === 'media_server');
    const [rollupRows, mappingRows, unmappedRows, itemRows, jobRows, diagnosticRows] = await Promise.all([
      api.activityRollups().catch(() => [] as ActivityRollup[]),
      api.pathMappings().catch(() => [] as PathMapping[]),
      api.unmappedPathItems().catch(() => [] as MediaServerItem[]),
      Promise.all(mediaServers.map((integration) => api.integrationItems(integration.id).catch(() => [] as MediaServerItem[]))),
      Promise.all(mediaServers.map(async (integration) => {
        const job = await api.integrationSyncStatus(integration.id).catch(() => null);
        return [integration.id, job] as const;
      })),
      Promise.all(mediaServers.map(async (integration) => {
        const diagnostics = await api.integrationDiagnostics(integration.id).catch(() => null);
        return [integration.id, diagnostics] as const;
      })),
    ]);
    setActivityRollups(rollupRows);
    setPathMappings(mappingRows);
    setUnmappedItems(unmappedRows);
    setIntegrationItems(itemRows.flat());
    setSyncJobs(Object.fromEntries(jobRows));
    setIntegrationDiagnostics(Object.fromEntries(diagnosticRows));
  }

  useEffect(() => {
    void bootstrap();
  }, []);

  useEffect(() => {
    if (!user) {
      return undefined;
    }
    void refreshJobs();
    const timer = window.setInterval(() => {
      void refreshJobs();
    }, 1500);
    return () => window.clearInterval(timer);
  }, [user]);

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
      const job = await api.startScan();
      setJobs((current) => [job, ...current.filter((row) => row.id !== job.id)]);
      await refreshJobs();
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
      const result = await api.createBackup();
      setBackupNotice(`Backup created: ${result.path}`);
      setError(null);
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : 'Backup failed');
    } finally {
      setBusy(false);
    }
  }

  async function createSupportBundle() {
    setBusy(true);
    try {
      const result = await api.createSupportBundle();
      setBackupNotice(`Support bundle created: ${result.path} (${result.files.length} files, ${formatBytes(result.sizeBytes)}).`);
      setSupportBundles(await api.supportBundles());
      setError(null);
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : 'Support bundle failed');
    } finally {
      setBusy(false);
    }
  }

  async function restoreBackup(path: string, dryRun: boolean) {
    setBusy(true);
    try {
      const result = await api.restoreBackup(path, dryRun);
      if (dryRun) {
        setBackupNotice(`Archive contains ${result.entries?.length ?? 0} entries.`);
      } else {
        setBackupNotice(`Restored ${result.restored?.length ?? 0} entries. Pre-restore backup: ${result.preRestoreBackup}`);
      }
      setError(null);
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : 'Backup restore failed');
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

  async function protectRecommendation(id: string) {
    setBusy(true);
    try {
      await api.protectRecommendation(id);
      setRecommendations((current) => current.filter((rec) => rec.id !== id));
      setRecommendationEvidence((current) => {
        const next = { ...current };
        delete next[id];
        return next;
      });
      setError(null);
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : 'Unable to protect recommendation');
    } finally {
      setBusy(false);
    }
  }

  async function acceptRecommendation(id: string) {
    setBusy(true);
    try {
      await api.acceptRecommendation(id);
      const [recs, evidence] = await Promise.all([api.recommendations(), api.recommendationEvidence(id).catch(() => null)]);
      setRecommendations(recs);
      if (evidence) {
        setRecommendationEvidence((current) => ({ ...current, [id]: evidence }));
      }
      setError(null);
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : 'Unable to accept recommendation');
    } finally {
      setBusy(false);
    }
  }

  async function loadRecommendationEvidence(id: string) {
    try {
      const evidence = await api.recommendationEvidence(id);
      setRecommendationEvidence((current) => ({ ...current, [id]: evidence }));
      setError(null);
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : 'Unable to load recommendation evidence');
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

  async function updateIntegrationSetting(integration: string, setting: IntegrationSettingInput) {
    setBusy(true);
    try {
      await api.updateIntegrationSetting(integration, setting);
      const [settings, integrationRows] = await Promise.all([api.integrationSettings(), api.integrations()]);
      setIntegrationSettings(settings);
      setIntegrations(integrationRows);
      await refreshIntegrationActivity(integrationRows);
      setError(null);
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : 'Unable to update integration settings');
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

  async function syncIntegration(id: string) {
    setBusy(true);
    try {
      const job = await api.syncIntegration(id);
      setSyncJobs((current) => ({ ...current, [id]: job }));
      await refreshJobs();
      setError(null);
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : 'Unable to sync integration');
    } finally {
      setBusy(false);
    }
  }

  async function savePathMapping(mapping: Partial<PathMapping> & Pick<PathMapping, 'serverPathPrefix' | 'localPathPrefix'>) {
    setBusy(true);
    try {
      await api.upsertPathMapping(mapping);
      await refreshIntegrationActivity();
      setError(null);
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : 'Unable to save path mapping');
    } finally {
      setBusy(false);
    }
  }

  async function verifyPathMapping(id: string) {
    setBusy(true);
    try {
      await api.verifyPathMapping(id);
      await refreshIntegrationActivity();
      const recs = await api.recommendations();
      setRecommendations(recs);
      setError(null);
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : 'Unable to verify path mapping');
    } finally {
      setBusy(false);
    }
  }

  async function deletePathMapping(id: string) {
    setBusy(true);
    try {
      await api.deletePathMapping(id);
      await refreshIntegrationActivity();
      setError(null);
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : 'Unable to remove path mapping');
    } finally {
      setBusy(false);
    }
  }

  async function cancelJob(id: string) {
    try {
      const job = await api.cancelJob(id);
      setJobs((current) => [job, ...current.filter((row) => row.id !== id)]);
      await refreshJobs();
      setError(null);
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : 'Unable to cancel job');
    }
  }

  async function retryJob(id: string) {
    try {
      const job = await api.retryJob(id);
      setJobs((current) => [job, ...current.filter((row) => row.id !== job.id)]);
      await refreshJobs();
      setError(null);
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : 'Unable to retry job');
    }
  }

  const catalogItems = useMemo(() => catalog, [catalog]);
  const totalFiles = catalogItems.length;
  const totalSize = catalogItems.reduce((sum, item) => sum + item.sizeBytes, 0);
  const recoverable = recommendations.reduce((sum, rec) => sum + rec.spaceSavedBytes, 0);
  const scanning = jobs.some((job) => isActiveJob(job) && job.kind === 'filesystem_scan');

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
            <button className="primary-button" onClick={() => void startScan()} disabled={busy || scanning}>
              <Activity size={18} />
              {scanning ? 'Scanning' : busy ? 'Working' : 'Scan now'}
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
            activityRollups={activityRollups}
            jobs={jobs}
            jobDetails={jobDetails}
            onCancelJob={(id) => void cancelJob(id)}
            onRetryJob={(id) => void retryJob(id)}
          />
        )}
        {view === 'libraries' && <Libraries libraries={libraries} scans={scans} />}
        {view === 'catalog' && <Catalog items={catalogItems} onCorrect={(id, correction) => void correctCatalogItem(id, correction)} onClear={(id) => void clearCatalogCorrection(id)} busy={busy} />}
        {view === 'recommendations' && (
          <RecommendationQueue
            recommendations={recommendations}
            evidence={recommendationEvidence}
            onEvidence={(id) => void loadRecommendationEvidence(id)}
            onAccept={(id) => void acceptRecommendation(id)}
            onProtect={(id) => void protectRecommendation(id)}
            onIgnore={(id) => void ignoreRecommendation(id)}
            busy={busy}
          />
        )}
        {view === 'integrations' && (
          <Integrations
            providers={providers}
            providerSettings={providerSettings}
            integrations={integrations}
            integrationSettings={integrationSettings}
            aiStatus={aiStatus}
            syncJobs={syncJobs}
            integrationDiagnostics={integrationDiagnostics}
            integrationItems={integrationItems}
            activityRollups={activityRollups}
            pathMappings={pathMappings}
            unmappedItems={unmappedItems}
            jobs={jobs}
            jobDetails={jobDetails}
            onProviderUpdate={(provider, setting) => void updateProviderSetting(provider, setting)}
            onIntegrationUpdate={(integration, setting) => void updateIntegrationSetting(integration, setting)}
            onIntegrationRefresh={(id) => void refreshIntegration(id)}
            onIntegrationSync={(id) => void syncIntegration(id)}
            onPathMappingSave={(mapping) => void savePathMapping(mapping)}
            onPathMappingVerify={(id) => void verifyPathMapping(id)}
            onPathMappingDelete={(id) => void deletePathMapping(id)}
            onCancelJob={(id) => void cancelJob(id)}
            onRetryJob={(id) => void retryJob(id)}
            busy={busy}
          />
        )}
        {view === 'settings' && (
          <SettingsView
            onBackup={() => void createBackup()}
            onSupportBundle={() => void createSupportBundle()}
            onRestore={(path, dryRun) => void restoreBackup(path, dryRun)}
            supportBundles={supportBundles}
            notice={backupNotice}
            busy={busy}
          />
        )}
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
  activityRollups,
  jobs,
  jobDetails,
  onCancelJob,
  onRetryJob,
}: {
  status: string;
  libraries: MediaLibrary[];
  totalFiles: number;
  totalSize: number;
  recoverable: number;
  recommendations: Recommendation[];
  scans: ScanResult[];
  activityRollups: ActivityRollup[];
  jobs: Job[];
  jobDetails: Record<string, JobDetail>;
  onCancelJob: (id: string) => void;
  onRetryJob: (id: string) => void;
}) {
  const lastScan = scans.at(-1);
  const activityRecommendations = recommendations.filter((rec) => rec.serverId);
  const neverWatched = recommendations.filter((rec) => rec.action === 'review_never_watched_movie').length;
  const verifiedSavings = recommendations
    .filter((rec) => rec.verification === 'local_verified' || rec.verification === 'path_mapped')
    .reduce((sum, rec) => sum + rec.spaceSavedBytes, 0);
  return (
    <section className="view-grid">
      <JobTelemetry jobs={jobs} jobDetails={jobDetails} onCancel={onCancelJob} onRetry={onRetryJob} />
      <div className="stat-strip">
        <Stat icon={<HeartPulse />} label="System" value={status.toUpperCase()} />
        <Stat icon={<FolderOpen />} label="Libraries" value={String(libraries.length)} />
        <Stat icon={<Database />} label="Indexed Files" value={String(totalFiles)} />
        <Stat icon={<HardDrive />} label="Indexed Size" value={formatBytes(totalSize)} />
        <Stat icon={<Archive />} label="Review Savings" value={formatBytes(recoverable)} />
      </div>
      <div className="stat-strip activity-strip">
        <Stat icon={<Server />} label="Activity Items" value={String(activityRollups.length)} />
        <Stat icon={<Trash2 />} label="Cold Suggestions" value={String(activityRecommendations.length)} />
        <Stat icon={<PlayCircle />} label="Never Watched" value={String(neverWatched)} />
        <Stat icon={<ShieldCheck />} label="Verified Savings" value={formatBytes(verifiedSavings)} />
        <Stat icon={<Bot />} label="AI Mode" value="Advisory" />
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

function RecommendationQueue({
  recommendations,
  evidence,
  onEvidence,
  onAccept,
  onProtect,
  onIgnore,
  busy,
}: {
  recommendations: Recommendation[];
  evidence: Record<string, RecommendationEvidence>;
  onEvidence: (id: string) => void;
  onAccept: (id: string) => void;
  onProtect: (id: string) => void;
  onIgnore: (id: string) => void;
  busy: boolean;
}) {
  return (
    <section className="queue">
      {recommendations.map((rec) => {
        const proof = evidence[rec.id];
        return (
          <article className="recommendation-card" key={rec.id}>
            <div className="rec-main">
              <div className="rec-icon"><ShieldCheck size={22} /></div>
              <div>
                <div className="rec-title-row">
                  <h2>{rec.title}</h2>
                  <span className="status-pill">{formatRecommendationState(rec.state)}</span>
                </div>
                <p>{rec.explanation}</p>
                <div className="path-list">
                  {rec.affectedPaths.map((path) => <code key={path}>{path}</code>)}
                </div>
                <div className="rec-evidence">
                  <Signal label="Source" value={rec.serverId || 'Local scan'} />
                  <Signal label="Last Played" value={rec.lastPlayedAt ? formatDateTime(rec.lastPlayedAt) : rec.serverId ? 'Never' : 'N/A'} />
                  <Signal label="Plays" value={String(rec.playCount ?? 0)} />
                  <Signal label="Users" value={String(rec.uniqueUsers ?? 0)} />
                  <Signal label="Evidence" value={formatVerification(rec.verification)} />
                </div>
                {proof && <RecommendationEvidencePanel evidence={proof} />}
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
              <div className="button-column">
                <button className="secondary-button" disabled={busy} onClick={() => onEvidence(rec.id)}>
                  <SearchCheck size={16} />
                  Proof
                </button>
                <button className="secondary-button" disabled={busy} onClick={() => onAccept(rec.id)}>
                  <Check size={16} />
                  Manual
                </button>
                <button className="secondary-button" disabled={busy} onClick={() => onProtect(rec.id)}>
                  <ShieldCheck size={16} />
                  Protect
                </button>
                <button className="secondary-button" disabled={busy} onClick={() => onIgnore(rec.id)}>Ignore</button>
              </div>
            </div>
          </article>
        );
      })}
      {recommendations.length === 0 && <EmptyState icon={<Trash2 />} text="No cleanup recommendations are open." />}
    </section>
  );
}

function RecommendationEvidencePanel({ evidence }: { evidence: RecommendationEvidence }) {
  return (
    <div className="evidence-panel">
      <div className="signal-grid">
        <Signal label="Storage" value={formatVerification(evidence.storage.verification)} />
        <Signal label="Risk" value={evidence.storage.risk} />
        <Signal label="Saved" value={formatBytes(evidence.storage.spaceSavedBytes)} />
        <Signal label="Rule" value={evidence.source.rule.replace('rule:', '')} />
      </div>
      <div className="proof-grid">
        {evidence.proof.map((point) => (
          <div className="proof-point" key={`${point.label}-${point.value}`}>
            <span>{point.label}</span>
            <strong>{point.value}</strong>
            <small>{point.status}</small>
          </div>
        ))}
      </div>
      {evidence.suppressionReasons.length > 0 && (
        <div className="notice warning">{evidence.suppressionReasons.join(', ')}</div>
      )}
    </div>
  );
}

function Integrations({
  providers,
  providerSettings,
  integrations,
  integrationSettings,
  aiStatus,
  syncJobs,
  integrationDiagnostics,
  integrationItems,
  activityRollups,
  pathMappings,
  unmappedItems,
  jobs,
  jobDetails,
  onProviderUpdate,
  onIntegrationUpdate,
  onIntegrationRefresh,
  onIntegrationSync,
  onPathMappingSave,
  onPathMappingVerify,
  onPathMappingDelete,
  onCancelJob,
  onRetryJob,
  busy,
}: {
  providers: ProviderHealth[];
  providerSettings: ProviderSetting[];
  integrations: Integration[];
  integrationSettings: IntegrationSetting[];
  aiStatus: AIStatus | null;
  syncJobs: Record<string, IntegrationSyncJob | null>;
  integrationDiagnostics: Record<string, IntegrationDiagnostics | null>;
  integrationItems: MediaServerItem[];
  activityRollups: ActivityRollup[];
  pathMappings: PathMapping[];
  unmappedItems: MediaServerItem[];
  jobs: Job[];
  jobDetails: Record<string, JobDetail>;
  onProviderUpdate: (provider: string, setting: ProviderSettingInput) => void;
  onIntegrationUpdate: (integration: string, setting: IntegrationSettingInput) => void;
  onIntegrationRefresh: (id: string) => void;
  onIntegrationSync: (id: string) => void;
  onPathMappingSave: (mapping: Partial<PathMapping> & Pick<PathMapping, 'serverPathPrefix' | 'localPathPrefix'>) => void;
  onPathMappingVerify: (id: string) => void;
  onPathMappingDelete: (id: string) => void;
  onCancelJob: (id: string) => void;
  onRetryJob: (id: string) => void;
  busy: boolean;
}) {
  const mediaServers = integrations.filter((integration) => integration.kind === 'media_server');
  const activeSyncJobs = jobs.filter((job) => isActiveJob(job) && job.kind.endsWith('_sync'));
  return (
    <section className="view-grid">
      <div className="integration-grid">
        {mediaServers.map((integration) => {
          const activeJob = activeSyncJobs.find((job) => job.targetId === integration.id);
          return (
            <MediaServerCard
              key={integration.id}
              integration={integration}
              setting={integrationSettings.find((setting) => setting.integration === integration.id)}
              job={syncJobs[integration.id] ?? null}
              diagnostics={integrationDiagnostics[integration.id] ?? null}
              importedItems={integrationItems.filter((item) => item.serverId === integration.id).length}
              activityCount={activityRollups.filter((rollup) => rollup.serverId === integration.id).length}
              pathMappingCount={pathMappings.filter((mapping) => !mapping.serverId || mapping.serverId === integration.id).length}
              activeJob={activeJob}
              jobDetail={activeJob ? jobDetails[activeJob.id] : undefined}
              busy={busy}
              onUpdate={onIntegrationUpdate}
              onRefresh={onIntegrationRefresh}
              onSync={onIntegrationSync}
              onCancelJob={onCancelJob}
              onRetryJob={onRetryJob}
            />
          );
        })}
      </div>
      <PathMappingWorkbench
        integrations={mediaServers}
        mappings={pathMappings}
        unmappedItems={unmappedItems}
        busy={busy}
        onSave={onPathMappingSave}
        onVerify={onPathMappingVerify}
        onDelete={onPathMappingDelete}
      />
      <div className="grid-list">
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

function MediaServerCard({
  integration,
  setting,
  job,
  diagnostics,
  activeJob,
  jobDetail,
  importedItems,
  activityCount,
  pathMappingCount,
  busy,
  onUpdate,
  onRefresh,
  onSync,
  onCancelJob,
  onRetryJob,
}: {
  integration: Integration;
  setting?: IntegrationSetting;
  job: IntegrationSyncJob | null;
  diagnostics: IntegrationDiagnostics | null;
  activeJob?: Job;
  jobDetail?: JobDetail;
  importedItems: number;
  activityCount: number;
  pathMappingCount: number;
  busy: boolean;
  onUpdate: (integration: string, setting: IntegrationSettingInput) => void;
  onRefresh: (id: string) => void;
  onSync: (id: string) => void;
  onCancelJob: (id: string) => void;
  onRetryJob: (id: string) => void;
}) {
  const [baseUrl, setBaseURL] = useState(setting?.baseUrl || '');
  const [apiKey, setAPIKey] = useState('');
  const [autoSyncEnabled, setAutoSyncEnabled] = useState(setting?.autoSyncEnabled ?? true);
  const [autoSyncIntervalMinutes, setAutoSyncIntervalMinutes] = useState(String(setting?.autoSyncIntervalMinutes ?? 360));
  const displayJob = activeJob ?? syncJobToJob(job);
  const effectiveAutoSyncEnabled = setting?.autoSyncEnabled ?? autoSyncEnabled;
  const effectiveAutoSyncInterval = setting?.autoSyncIntervalMinutes ?? (Number.parseInt(autoSyncIntervalMinutes, 10) || 360);
  const nextSync = nextAutoSyncAt(job?.completedAt, effectiveAutoSyncInterval, effectiveAutoSyncEnabled && (setting?.apiKeyConfigured || integration.status === 'configured'));

  useEffect(() => {
    setBaseURL(setting?.baseUrl || '');
    setAPIKey('');
    setAutoSyncEnabled(setting?.autoSyncEnabled ?? true);
    setAutoSyncIntervalMinutes(String(setting?.autoSyncIntervalMinutes ?? 360));
  }, [setting?.baseUrl, setting?.apiKeyConfigured, setting?.autoSyncEnabled, setting?.autoSyncIntervalMinutes]);

  function submit(event: React.FormEvent) {
    event.preventDefault();
    const interval = clampAutoSyncInterval(Number.parseInt(autoSyncIntervalMinutes, 10));
    onUpdate(integration.id, {
      baseUrl,
      apiKey,
      clearApiKey: apiKey === '' ? undefined : false,
      autoSyncEnabled,
      autoSyncIntervalMinutes: interval,
    });
    setAutoSyncIntervalMinutes(String(interval));
    setAPIKey('');
  }

  return (
    <article className="media-server-card">
      <div className="server-mark"><Server size={22} /></div>
      <div className="server-card-main">
        <div className="panel-heading">
          <div>
            <h2>{integration.name}</h2>
            <p>{integration.description}</p>
          </div>
          <span className="status-pill">{integration.status}</span>
        </div>
        <div className="signal-grid">
          <Signal label="Imported" value={String(job?.itemsImported ?? importedItems)} />
          <Signal label="Activity" value={String(job?.rollupsImported ?? activityCount)} />
          <Signal label="Unmapped" value={String(job?.unmappedItems ?? 0)} />
          <Signal label="Credential" value={setting?.apiKeyConfigured ? `Key ...${setting.apiKeyLast4 || ''}` : 'Not set'} />
          <Signal label="Backoff" value={integration.retryPolicy || 'standard'} />
          <Signal label="Auto Sync" value={effectiveAutoSyncEnabled ? `Every ${formatMinutes(effectiveAutoSyncInterval)}` : 'Disabled'} />
          <Signal label="Next Sync" value={nextSync || 'After connect'} />
          <Signal label="Verified" value={diagnostics ? formatBytes(diagnostics.summary.locallyVerifiedBytes) : 'Pending'} />
          <Signal label="Suggestions" value={diagnostics ? String(diagnostics.summary.recommendations) : 'Pending'} />
        </div>
        <form className="integration-config" onSubmit={submit}>
          <label>
            Server URL
            <input value={baseUrl} onChange={(event) => setBaseURL(event.target.value)} placeholder={integration.id === 'plex' ? 'http://plex:32400' : 'http://jellyfin:8096'} />
          </label>
          <label>
            API key or token
            <input value={apiKey} onChange={(event) => setAPIKey(event.target.value)} type="password" placeholder={setting?.apiKeyConfigured ? `Configured ...${setting.apiKeyLast4 || ''}` : 'Paste token'} />
          </label>
          <label className="checkbox-row">
            <input checked={autoSyncEnabled} onChange={(event) => setAutoSyncEnabled(event.target.checked)} type="checkbox" />
            Auto-sync after connection and on schedule
          </label>
          <label>
            Auto-sync interval
            <input value={autoSyncIntervalMinutes} onChange={(event) => setAutoSyncIntervalMinutes(event.target.value)} type="number" min={15} max={10080} step={15} />
          </label>
          <div className="button-row">
            <button className="secondary-button" type="submit" disabled={busy || (!baseUrl && !apiKey && !setting)}>
              <Save size={16} />
              Save
            </button>
            <button className="secondary-button" type="button" disabled={busy || !setting?.apiKeyConfigured} onClick={() => onUpdate(integration.id, { baseUrl, clearApiKey: true })}>
              <RotateCcw size={16} />
              Clear key
            </button>
            <button className="secondary-button" type="button" disabled={busy || (!setting?.baseUrl && !setting?.apiKeyConfigured)} onClick={() => onUpdate(integration.id, { clearApiKey: true, clearBaseUrl: true })}>
              <LogOut size={16} />
              Disconnect
            </button>
          </div>
        </form>
        <div className="server-sync-row">
          <span>{job?.completedAt ? `Last sync ${formatDateTime(job.completedAt)}` : `No inventory sync yet • ${pathMappingCount} path mappings`}</span>
          <div className="button-row">
            <button className="secondary-button" onClick={() => onRefresh(integration.id)} disabled={busy}>
              <RefreshCw size={16} />
              Refresh
            </button>
            <button className="primary-button" onClick={() => onSync(integration.id)} disabled={busy}>
              <Database size={16} />
              {activeJob ? 'Syncing' : 'Sync'}
            </button>
          </div>
        </div>
        {diagnostics && <IntegrationDiagnosticsPanel diagnostics={diagnostics} />}
        {displayJob && <ProgressPanel job={displayJob} events={jobDetail?.events || []} compact onCancel={onCancelJob} onRetry={onRetryJob} />}
      </div>
    </article>
  );
}

function IntegrationDiagnosticsPanel({ diagnostics }: { diagnostics: IntegrationDiagnostics }) {
  const summary = diagnostics.summary;
  const warningCount = diagnostics.warnings.length;
  return (
    <div className="diagnostics-panel">
      <div className="diagnostics-header">
        <span>Ingestion Proof</span>
        <span className={warningCount > 0 ? 'status-pill warning' : 'status-pill'}>{warningCount > 0 ? `${warningCount} warnings` : 'Trusted'}</span>
      </div>
      <div className="diagnostics-grid">
        <Signal label="Movies" value={String(summary.movies)} />
        <Signal label="Series" value={String(summary.series)} />
        <Signal label="Episodes" value={String(summary.episodes)} />
        <Signal label="Anime" value={String(summary.animeItems)} />
        <Signal label="Files" value={String(summary.files)} />
        <Signal label="Unmapped" value={String(summary.unmappedFiles)} />
        <Signal label="Server Size" value={formatBytes(summary.serverReportedBytes)} />
        <Signal label="Local Proof" value={formatBytes(summary.locallyVerifiedBytes)} />
      </div>
      {diagnostics.warnings.length > 0 && (
        <ul className="diagnostics-warnings">
          {diagnostics.warnings.slice(0, 3).map((warning) => <li key={warning}>{warning}</li>)}
        </ul>
      )}
      {diagnostics.topRecommendations.length > 0 && (
        <div className="diagnostics-top">
          <span>Top suggestion</span>
          <strong>{formatBytes(diagnostics.topRecommendations[0].spaceSavedBytes)}</strong>
          <span>{formatVerification(diagnostics.topRecommendations[0].verification)}</span>
        </div>
      )}
    </div>
  );
}

function PathMappingWorkbench({
  integrations,
  mappings,
  unmappedItems,
  busy,
  onSave,
  onVerify,
  onDelete,
}: {
  integrations: Integration[];
  mappings: PathMapping[];
  unmappedItems: MediaServerItem[];
  busy: boolean;
  onSave: (mapping: Partial<PathMapping> & Pick<PathMapping, 'serverPathPrefix' | 'localPathPrefix'>) => void;
  onVerify: (id: string) => void;
  onDelete: (id: string) => void;
}) {
  const [serverId, setServerID] = useState(integrations[0]?.id || 'jellyfin');
  const [serverPathPrefix, setServerPathPrefix] = useState('/mnt/media');
  const [localPathPrefix, setLocalPathPrefix] = useState('/media');

  useEffect(() => {
    if (!integrations.some((integration) => integration.id === serverId)) {
      setServerID(integrations[0]?.id || 'jellyfin');
    }
  }, [integrations, serverId]);

  function submit(event: React.FormEvent) {
    event.preventDefault();
    onSave({ serverId, serverPathPrefix, localPathPrefix });
  }

  return (
    <section className="mapping-workbench">
      <div className="panel-heading">
        <div>
          <h2>Path Mapping</h2>
          <p>{unmappedItems.length} unmapped server items require path proof</p>
        </div>
        <span className={unmappedItems.length > 0 ? 'status-pill warning' : 'status-pill'}>{unmappedItems.length > 0 ? 'Review' : 'Verified'}</span>
      </div>
      <form className="mapping-form" onSubmit={submit}>
        <label>
          Server
          <select value={serverId} onChange={(event) => setServerID(event.target.value)}>
            {integrations.map((integration) => <option value={integration.id} key={integration.id}>{integration.name}</option>)}
          </select>
        </label>
        <label>
          Server path
          <input value={serverPathPrefix} onChange={(event) => setServerPathPrefix(event.target.value)} />
        </label>
        <label>
          Mediarr path
          <input value={localPathPrefix} onChange={(event) => setLocalPathPrefix(event.target.value)} />
        </label>
        <button className="primary-button" type="submit" disabled={busy || !serverPathPrefix.trim() || !localPathPrefix.trim()}>
          <MapIcon size={16} />
          Save mapping
        </button>
      </form>
      <div className="mapping-grid">
        <div className="mapping-list">
          {mappings.map((mapping) => (
            <article className="mapping-row" key={mapping.id}>
              <div>
                <strong>{mapping.serverId || 'All servers'}</strong>
                <code>{mapping.serverPathPrefix}</code>
                <code>{mapping.localPathPrefix}</code>
              </div>
              <div className="button-row">
                <button className="secondary-button" disabled={busy} onClick={() => onVerify(mapping.id)}>
                  <SearchCheck size={16} />
                  Verify
                </button>
                <button className="secondary-button" disabled={busy} onClick={() => onDelete(mapping.id)}>
                  <Trash2 size={16} />
                  Remove
                </button>
              </div>
            </article>
          ))}
          {mappings.length === 0 && <EmptyState icon={<MapIcon />} text="No path mappings configured." />}
        </div>
        <div className="unmapped-list">
          {unmappedItems.slice(0, 8).map((item) => (
            <div className="unmapped-row" key={`${item.serverId}-${item.externalId}`}>
              <span>{item.serverId}</span>
              <strong>{item.title}</strong>
              <code>{item.path || 'No path reported'}</code>
            </div>
          ))}
          {unmappedItems.length === 0 && <EmptyState icon={<ShieldCheck />} text="No unmapped server items." />}
        </div>
      </div>
    </section>
  );
}

function Signal({ label, value }: { label: string; value: string }) {
  return (
    <div className="signal">
      <span>{label}</span>
      <strong>{value}</strong>
    </div>
  );
}

function JobTelemetry({
  jobs,
  jobDetails,
  onCancel,
  onRetry,
}: {
  jobs: Job[];
  jobDetails: Record<string, JobDetail>;
  onCancel: (id: string) => void;
  onRetry: (id: string) => void;
}) {
  const active = jobs.filter(isActiveJob);
  const recent = jobs.filter((job) => !isActiveJob(job)).slice(0, 3);
  if (active.length === 0 && recent.length === 0) {
    return null;
  }
  return (
    <section className="job-telemetry">
      {active.map((job) => (
        <ProgressPanel key={job.id} job={job} events={jobDetails[job.id]?.events || []} onCancel={onCancel} onRetry={onRetry} />
      ))}
      {active.length === 0 && recent.map((job) => (
        <ProgressPanel key={job.id} job={job} events={jobDetails[job.id]?.events || []} compact onCancel={onCancel} onRetry={onRetry} />
      ))}
    </section>
  );
}

function ProgressPanel({
  job,
  events,
  compact = false,
  onCancel,
  onRetry,
}: {
  job: Job;
  events: JobEvent[];
  compact?: boolean;
  onCancel?: (id: string) => void;
  onRetry?: (id: string) => void;
}) {
  const percent = job.total > 0 ? Math.min(100, Math.round((job.processed / job.total) * 100)) : 0;
  const visibleEvents = events.slice(0, compact ? 3 : 5);
  return (
    <article className={compact ? 'progress-panel compact' : 'progress-panel'}>
      <div className="progress-header">
        <div>
          <span className="status-pill">{job.status}</span>
          <h2>{jobKindLabel(job)}</h2>
          <p>{job.message || job.phase}</p>
        </div>
        <div className="progress-actions">
          <strong>{job.total > 0 ? `${job.processed} / ${job.total}` : job.currentLabel || job.phase}</strong>
          {isActiveJob(job) && onCancel && (
            <button className="secondary-button compact-action" onClick={() => onCancel(job.id)}>Cancel</button>
          )}
          {!isActiveJob(job) && job.status !== 'completed' && onRetry && (
            <button className="secondary-button compact-action" onClick={() => onRetry(job.id)}>Retry</button>
          )}
        </div>
      </div>
      <div className="progress-track" aria-label={`${jobKindLabel(job)} progress`}>
        <span style={{ width: job.total > 0 ? `${percent}%` : isActiveJob(job) ? '36%' : '100%' }} />
      </div>
      <div className="progress-meta">
        <span>{job.currentLabel || 'Preparing'}</span>
        <span>{job.itemsImported ? `${job.itemsImported} imported` : job.rollupsImported ? `${job.rollupsImported} activity rows` : formatElapsed(job.startedAt, job.completedAt)}</span>
        {job.unmappedItems > 0 && <span>{job.unmappedItems} unmapped</span>}
      </div>
      {visibleEvents.length > 0 && (
        <div className="event-feed">
          {visibleEvents.map((event) => (
            <div key={event.id} className="event-row">
              <span>{event.phase}</span>
              <strong>{event.currentLabel || event.message}</strong>
            </div>
          ))}
        </div>
      )}
      {job.error && <div className="notice error">{job.error}</div>}
    </article>
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

function SettingsView({
  onBackup,
  onSupportBundle,
  onRestore,
  supportBundles,
  notice,
  busy,
}: {
  onBackup: () => void;
  onSupportBundle: () => void;
  onRestore: (path: string, dryRun: boolean) => void;
  supportBundles: SupportBundle[];
  notice: string | null;
  busy: boolean;
}) {
  const [restorePath, setRestorePath] = useState('');
  return (
    <section className="settings-layout">
      <div className="panel form-panel">
        <div className="panel-heading">
          <h2>Backups</h2>
          <span>/config/backups</span>
        </div>
        <button className="primary-button" onClick={onBackup} disabled={busy}>
          <Archive size={18} />
          Create backup
        </button>
        <label>
          Restore archive
          <input value={restorePath} onChange={(event) => setRestorePath(event.target.value)} placeholder="/config/backups/mediarr-...zip" />
        </label>
        <div className="button-row">
          <button className="secondary-button" type="button" onClick={() => onRestore(restorePath, true)} disabled={busy || restorePath.trim() === ''}>
            <SearchCheck size={16} />
            Inspect
          </button>
          <button className="secondary-button" type="button" onClick={() => onRestore(restorePath, false)} disabled={busy || restorePath.trim() === ''}>
            <RotateCcw size={16} />
            Restore
          </button>
        </div>
        {notice && <div className="notice success">{notice}</div>}
      </div>
      <div className="panel form-panel">
        <div className="panel-heading">
          <h2>Support Bundle</h2>
          <span>/config/support</span>
        </div>
        <p className="muted-copy">Export redacted settings, ingestion diagnostics, path mappings, jobs, recommendations, and safety proof for support without bundling media files or raw database contents.</p>
        <button className="secondary-button" type="button" onClick={onSupportBundle} disabled={busy}>
          <ShieldCheck size={16} />
          Create support bundle
        </button>
        <div className="compact-list">
          {supportBundles.length === 0 && <p className="muted-copy">No support bundles have been created yet.</p>}
          {supportBundles.slice(0, 5).map((bundle) => (
            <div className="compact-row bundle-row" key={bundle.name}>
              <div>
                <strong>{bundle.name}</strong>
                <span>{formatBytes(bundle.sizeBytes)} • {formatDateTime(bundle.createdAt)}</span>
              </div>
              <a className="secondary-button compact-action" href={api.supportBundleDownloadUrl(bundle.name)} download>
                <Download size={16} />
                Download
              </a>
            </div>
          ))}
        </div>
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

function formatDateTime(value: string): string {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return 'Unknown';
  }
  return date.toLocaleDateString(undefined, { year: 'numeric', month: 'short', day: 'numeric' });
}

function formatVerification(value?: string): string {
  switch (value) {
    case 'local_verified':
      return 'Local';
    case 'path_mapped':
      return 'Mapped';
    case 'server_reported':
      return 'Server';
    default:
      return 'Unknown';
  }
}

function formatRecommendationState(value?: string): string {
  switch (value) {
    case 'accepted_for_manual_action':
      return 'Manual';
    case 'reviewing':
      return 'Reviewing';
    case 'protected':
      return 'Protected';
    case 'ignored':
      return 'Ignored';
    default:
      return 'New';
  }
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

function isActiveJob(job: Job): boolean {
  return job.status === 'queued' || job.status === 'running';
}

function jobKindLabel(job: Job): string {
  switch (job.kind) {
    case 'filesystem_scan':
      return 'Filesystem Scan';
    case 'jellyfin_sync':
      return 'Jellyfin Sync';
    case 'plex_sync':
      return 'Plex Sync';
    case 'emby_sync':
      return 'Emby Sync';
    default:
      return job.kind.replaceAll('_', ' ');
  }
}

function clampAutoSyncInterval(minutes: number): number {
  if (!Number.isFinite(minutes) || minutes <= 0) {
    return 360;
  }
  if (minutes < 15) {
    return 15;
  }
  if (minutes > 10080) {
    return 10080;
  }
  return minutes;
}

function formatMinutes(minutes: number): string {
  if (minutes % 1440 === 0) {
    const days = minutes / 1440;
    return `${days}d`;
  }
  if (minutes % 60 === 0) {
    const hours = minutes / 60;
    return `${hours}h`;
  }
  return `${minutes}m`;
}

function nextAutoSyncAt(completedAt: string | undefined, intervalMinutes: number, enabled: boolean): string {
  if (!enabled) {
    return 'Disabled';
  }
  if (!completedAt) {
    return 'Queued after save';
  }
  const completed = new Date(completedAt);
  if (Number.isNaN(completed.getTime())) {
    return 'Pending';
  }
  const next = new Date(completed.getTime() + clampAutoSyncInterval(intervalMinutes) * 60_000);
  return formatDateTime(next.toISOString());
}

function syncJobToJob(job: IntegrationSyncJob | null): Job | undefined {
  if (!job) {
    return undefined;
  }
  return {
    id: job.id,
    kind: `${job.serverId}_sync`,
    targetId: job.serverId,
    status: job.status,
    phase: job.phase || job.status,
    message: job.message || job.status,
    currentLabel: job.currentLabel,
    processed: job.processed || 0,
    total: job.total || 0,
    itemsImported: job.itemsImported,
    rollupsImported: job.rollupsImported,
    unmappedItems: job.unmappedItems,
    error: job.error,
    startedAt: job.startedAt,
    updatedAt: job.completedAt || job.startedAt,
    completedAt: job.completedAt,
  };
}

function formatElapsed(startedAt: string, completedAt?: string): string {
  const start = new Date(startedAt);
  const end = completedAt ? new Date(completedAt) : new Date();
  if (Number.isNaN(start.getTime()) || Number.isNaN(end.getTime())) {
    return 'Timing unavailable';
  }
  const seconds = Math.max(0, Math.round((end.getTime() - start.getTime()) / 1000));
  if (seconds < 60) {
    return `${seconds}s`;
  }
  const minutes = Math.floor(seconds / 60);
  return `${minutes}m ${seconds % 60}s`;
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
