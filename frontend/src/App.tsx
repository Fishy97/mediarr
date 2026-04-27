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
  Moon,
  Pencil,
  PlayCircle,
  RefreshCw,
  RotateCcw,
  Save,
  SearchCheck,
  Server,
  Settings,
  ShieldCheck,
  SlidersHorizontal,
  Sun,
  Trash2,
  UserRound,
} from 'lucide-react';
import { api, getAuthToken } from './lib/api';
import { applyAppearanceSettings, nextThemePreference, resolveTheme } from './lib/appearance';
import { formatBytes, formatConfidence, formatVerification, storageCertaintyDefinition, storageCertaintyDescription, storageCertaintyForVerification } from './lib/format';
import { groupAffectedPaths } from './lib/pathGroups';
import type {
  AIStatus,
  ActivityRollup,
  AppearanceSettings,
  AuthUser,
  Backup,
  Campaign,
  CampaignResult,
  CampaignRun,
  CampaignTemplate,
  CampaignRule,
  CampaignRuleField,
  CampaignRuleOperator,
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
  ProtectionRequest,
  PublicationInput,
  PublicationPlan,
  Recommendation,
  RecommendationEvidence,
  RequestSignal,
  RequestSource,
  RequestSourceInput,
  ScanResult,
  StorageLedger,
  StewardshipNotification,
  SupportBundle,
  WhatIfSimulation,
} from './types';

type View = 'dashboard' | 'libraries' | 'catalog' | 'recommendations' | 'campaigns' | 'integrations' | 'settings';

const integrationItemSampleLimit = 100;
const unmappedItemSampleLimit = 50;
const activityRollupSampleLimit = 250;
const defaultAppearance: AppearanceSettings = { theme: 'system', customCss: '' };

export function App() {
  const [view, setView] = useState<View>('dashboard');
  const [libraries, setLibraries] = useState<MediaLibrary[]>([]);
  const [scans, setScans] = useState<ScanResult[]>([]);
  const [catalog, setCatalog] = useState<CatalogItem[]>([]);
  const [recommendations, setRecommendations] = useState<Recommendation[]>([]);
  const [campaigns, setCampaigns] = useState<Campaign[]>([]);
  const [campaignTemplates, setCampaignTemplates] = useState<CampaignTemplate[]>([]);
  const [campaignResults, setCampaignResults] = useState<Record<string, CampaignResult>>({});
  const [campaignWhatIf, setCampaignWhatIf] = useState<Record<string, WhatIfSimulation>>({});
  const [publicationPreviews, setPublicationPreviews] = useState<Record<string, PublicationPlan>>({});
  const [campaignRuns, setCampaignRuns] = useState<Record<string, CampaignRun[]>>({});
  const [providers, setProviders] = useState<ProviderHealth[]>([]);
  const [providerSettings, setProviderSettings] = useState<ProviderSetting[]>([]);
  const [integrationSettings, setIntegrationSettings] = useState<IntegrationSetting[]>([]);
  const [integrations, setIntegrations] = useState<Integration[]>([]);
  const [syncJobs, setSyncJobs] = useState<Record<string, IntegrationSyncJob | null>>({});
  const [integrationDiagnostics, setIntegrationDiagnostics] = useState<Record<string, IntegrationDiagnostics | null>>({});
  const [integrationItems, setIntegrationItems] = useState<MediaServerItem[]>([]);
  const [activityRollups, setActivityRollups] = useState<ActivityRollup[]>([]);
  const [requestSources, setRequestSources] = useState<RequestSource[]>([]);
  const [requestSignals, setRequestSignals] = useState<RequestSignal[]>([]);
  const [storageLedger, setStorageLedger] = useState<StorageLedger | null>(null);
  const [notifications, setNotifications] = useState<StewardshipNotification[]>([]);
  const [protectionRequests, setProtectionRequests] = useState<ProtectionRequest[]>([]);
  const [pathMappings, setPathMappings] = useState<PathMapping[]>([]);
  const [unmappedItems, setUnmappedItems] = useState<MediaServerItem[]>([]);
  const [jobs, setJobs] = useState<Job[]>([]);
  const [jobDetails, setJobDetails] = useState<Record<string, JobDetail>>({});
  const [recommendationEvidence, setRecommendationEvidence] = useState<Record<string, RecommendationEvidence>>({});
  const [aiStatus, setAIStatus] = useState<AIStatus | null>(null);
  const [status, setStatus] = useState('Loading');
  const [error, setError] = useState<string | null>(null);
  const [backupNotice, setBackupNotice] = useState<string | null>(null);
  const [backups, setBackups] = useState<Backup[]>([]);
  const [supportBundles, setSupportBundles] = useState<SupportBundle[]>([]);
  const [appearance, setAppearance] = useState<AppearanceSettings>(defaultAppearance);
  const [appearanceSaving, setAppearanceSaving] = useState(false);
  const [busy, setBusy] = useState(false);
  const [authChecked, setAuthChecked] = useState(false);
  const [setupRequired, setSetupRequired] = useState(false);
  const [user, setUser] = useState<AuthUser | null>(null);
  const activeJobIds = useRef<Set<string>>(new Set());

  async function refresh() {
    try {
      const [health, libs, catalogRows, scanRows, recs, campaignRows, templateRows, providerRows, providerSettingRows, integrationSettingRows, integrationRows, appearanceSettings, ai, backupRows, bundleRows, ledger, sourceRows, signalRows, notificationRows, protectionRows] = await Promise.all([
        api.health(),
        api.libraries(),
        api.catalog(),
        api.scans(),
        api.recommendations(),
        api.campaigns(),
        api.campaignTemplates(),
        api.providers(),
        api.providerSettings(),
        api.integrationSettings(),
        api.integrations(),
        api.appearance(),
        api.aiStatus(),
        api.backups(),
        api.supportBundles(),
        api.storageLedger(),
        api.requestSources(),
        api.requestSignals(),
        api.notifications(),
        api.protectionRequests('pending'),
      ]);
      setStatus(health.status);
      setLibraries(libs);
      setCatalog(catalogRows);
      setScans(scanRows);
      setRecommendations(recs);
      setCampaigns(campaignRows);
      setCampaignTemplates(templateRows);
      setProviders(providerRows);
      setProviderSettings(providerSettingRows);
      setIntegrationSettings(integrationSettingRows);
      setIntegrations(integrationRows);
      setAppearance(appearanceSettings);
      setAIStatus(ai);
      setBackups(backupRows);
      setSupportBundles(bundleRows);
      setStorageLedger(ledger);
      setRequestSources(sourceRows);
      setRequestSignals(signalRows);
      setNotifications(notificationRows);
      setProtectionRequests(protectionRows);
      await refreshIntegrationActivity(integrationRows);
      await refreshCampaignRuns(campaignRows);
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
      api.activityRollups(undefined, activityRollupSampleLimit).catch(() => [] as ActivityRollup[]),
      api.pathMappings().catch(() => [] as PathMapping[]),
      api.unmappedPathItems(undefined, unmappedItemSampleLimit).catch(() => [] as MediaServerItem[]),
      Promise.all(mediaServers.map((integration) => api.integrationItems(integration.id, false, integrationItemSampleLimit).catch(() => [] as MediaServerItem[]))),
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

  async function refreshCampaignRuns(campaignRows = campaigns) {
    const rows = await Promise.all(campaignRows.map(async (campaign) => {
      const runs = await api.campaignRuns(campaign.id).catch(() => [] as CampaignRun[]);
      return [campaign.id, runs] as const;
    }));
    setCampaignRuns(Object.fromEntries(rows));
  }

  useEffect(() => {
    void bootstrap();
  }, []);

  useEffect(() => {
    const query = window.matchMedia('(prefers-color-scheme: light)');
    const sync = () => applyAppearanceSettings(document, appearance, query.matches);
    sync();
    if (appearance.theme !== 'system') {
      return undefined;
    }
    if (typeof query.addEventListener === 'function') {
      query.addEventListener('change', sync);
      return () => query.removeEventListener('change', sync);
    }
    query.addListener(sync);
    return () => query.removeListener(sync);
  }, [appearance]);

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
      setBackups(await api.backups());
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

  async function restoreBackup(name: string, dryRun: boolean) {
    if (!dryRun && !window.confirm(`Restore backup ${name}? This creates a pre-restore backup first, then replaces files under /config.`)) {
      return;
    }
    setBusy(true);
    try {
      const result = await api.restoreBackup(name, dryRun);
      if (dryRun) {
        setBackupNotice(`Archive contains ${result.entries?.length ?? 0} entries.`);
      } else {
        setBackupNotice(`Restored ${result.restored?.length ?? 0} entries. Pre-restore backup: ${result.preRestoreBackup}`);
        setBackups(await api.backups());
      }
      setError(null);
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : 'Backup restore failed');
    } finally {
      setBusy(false);
    }
  }

  async function updateAppearance(nextAppearance: AppearanceSettings) {
    setBusy(true);
    try {
      const updated = await api.updateAppearance(nextAppearance);
      setAppearance(updated);
      setBackupNotice('Appearance settings saved');
      setError(null);
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : 'Unable to update appearance settings');
    } finally {
      setBusy(false);
    }
  }

  async function toggleAppearanceTheme() {
    const prefersLight = window.matchMedia('(prefers-color-scheme: light)').matches;
    const nextTheme = nextThemePreference(appearance.theme, prefersLight);
    setAppearanceSaving(true);
    try {
      const updated = await api.updateAppearance({ ...appearance, theme: nextTheme });
      setAppearance(updated);
      setError(null);
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : 'Unable to update theme');
    } finally {
      setAppearanceSaving(false);
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

  async function updateRequestSource(id: string, source: RequestSourceInput) {
    setBusy(true);
    try {
      await api.updateRequestSource(id, source);
      setRequestSources(await api.requestSources());
      setError(null);
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : 'Unable to update request source');
    } finally {
      setBusy(false);
    }
  }

  async function syncRequestSource(id: string) {
    setBusy(true);
    try {
      await api.syncRequestSource(id);
      const [sources, signals, ledger, noticeRows] = await Promise.all([api.requestSources(), api.requestSignals(), api.storageLedger(), api.notifications()]);
      setRequestSources(sources);
      setRequestSignals(signals);
      setStorageLedger(ledger);
      setNotifications(noticeRows);
      setError(null);
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : 'Unable to sync request source');
    } finally {
      setBusy(false);
    }
  }

  async function syncTautulli() {
    setBusy(true);
    try {
      const job = await api.syncTautulli();
      setJobs((current) => [job, ...current.filter((row) => row.id !== job.id)]);
      await refreshJobs();
      setError(null);
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : 'Unable to sync Tautulli');
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

  async function createCampaignFromTemplate(id: string) {
    setBusy(true);
    try {
      const created = await api.createCampaignFromTemplate(id);
      const rows = await api.campaigns();
      setCampaigns(rows);
      setCampaignResults((current) => ({ ...current, [created.id]: current[created.id] ?? emptyCampaignResult(created.id) }));
      await refreshCampaignRuns(rows);
      setError(null);
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : 'Unable to create campaign template');
    } finally {
      setBusy(false);
    }
  }

  async function saveCampaign(campaign: Campaign) {
    setBusy(true);
    try {
      const saved = campaigns.some((row) => row.id === campaign.id)
        ? await api.updateCampaign(campaign.id, campaign)
        : await api.createCampaign(campaign);
      const rows = await api.campaigns();
      setCampaigns(rows);
      await refreshCampaignRuns(rows);
      setCampaignResults((current) => ({ ...current, [saved.id]: current[saved.id] ?? emptyCampaignResult(saved.id) }));
      setError(null);
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : 'Unable to save campaign');
    } finally {
      setBusy(false);
    }
  }

  async function simulateCampaign(id: string) {
    setBusy(true);
    try {
      const result = await api.simulateCampaign(id);
      setCampaignResults((current) => ({ ...current, [id]: result }));
      setError(null);
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : 'Unable to simulate campaign');
    } finally {
      setBusy(false);
    }
  }

  async function whatIfCampaign(id: string) {
    setBusy(true);
    try {
      const result = await api.whatIfCampaign(id);
      setCampaignWhatIf((current) => ({ ...current, [id]: result }));
      setError(null);
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : 'Unable to run what-if simulation');
    } finally {
      setBusy(false);
    }
  }

  async function previewCampaignPublication(id: string, input: PublicationInput) {
    setBusy(true);
    try {
      const preview = await api.publishCampaignPreview(id, input);
      setPublicationPreviews((current) => ({ ...current, [id]: preview }));
      setError(null);
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : 'Unable to preview collection publication');
    } finally {
      setBusy(false);
    }
  }

  async function runCampaign(id: string) {
    setBusy(true);
    try {
      const response = await api.runCampaign(id);
      const [recs, rows] = await Promise.all([api.recommendations(), api.campaigns()]);
      setRecommendations(recs);
      setCampaigns(rows);
      setCampaignResults((current) => ({ ...current, [id]: response.result }));
      await refreshCampaignRuns(rows);
      setError(null);
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : 'Unable to run campaign');
    } finally {
      setBusy(false);
    }
  }

  async function markNotificationRead(id: string) {
    try {
      const read = await api.markNotificationRead(id);
      setNotifications((current) => current.filter((notification) => notification.id !== read.id));
      setError(null);
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : 'Unable to mark notification read');
    }
  }

  async function createProtectionRequest(request: Omit<ProtectionRequest, 'id' | 'status' | 'createdAt' | 'decidedAt'>) {
    setBusy(true);
    try {
      await api.createProtectionRequest(request);
      setProtectionRequests(await api.protectionRequests('pending'));
      setError(null);
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : 'Unable to create protection request');
    } finally {
      setBusy(false);
    }
  }

  async function decideProtectionRequest(id: string, approve: boolean) {
    setBusy(true);
    try {
      const decisionBy = user?.email || 'admin';
      if (approve) {
        await api.approveProtectionRequest(id, decisionBy, 'Protected from stewardship review');
      } else {
        await api.declineProtectionRequest(id, decisionBy, 'Protection request declined');
      }
      const [requests, recs, ledger] = await Promise.all([api.protectionRequests('pending'), api.recommendations(), api.storageLedger()]);
      setProtectionRequests(requests);
      setRecommendations(recs);
      setStorageLedger(ledger);
      setError(null);
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : 'Unable to decide protection request');
    } finally {
      setBusy(false);
    }
  }

  async function deleteCampaign(id: string) {
    setBusy(true);
    try {
      await api.deleteCampaign(id);
      const rows = await api.campaigns();
      setCampaigns(rows);
      setCampaignResults((current) => {
        const next = { ...current };
        delete next[id];
        return next;
      });
      setCampaignRuns((current) => {
        const next = { ...current };
        delete next[id];
        return next;
      });
      setError(null);
    } catch (caught) {
      setError(caught instanceof Error ? caught.message : 'Unable to delete campaign');
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
  const resolvedTheme = resolveTheme(appearance.theme, window.matchMedia('(prefers-color-scheme: light)').matches);

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
          <div className="brand-identity">
            <div className="brand-mark"><SearchCheck size={20} /></div>
            <div>
              <strong>Mediarr</strong>
              <span>Library control plane</span>
            </div>
          </div>
          <ThemeToggle resolvedTheme={resolvedTheme} busy={appearanceSaving} onToggle={() => void toggleAppearanceTheme()} />
        </div>
        <nav className="nav">
          <NavButton icon={<Gauge />} label="Dashboard" active={view === 'dashboard'} onClick={() => setView('dashboard')} />
          <NavButton icon={<FolderOpen />} label="Libraries" active={view === 'libraries'} onClick={() => setView('libraries')} />
          <NavButton icon={<Library />} label="Catalog" active={view === 'catalog'} onClick={() => setView('catalog')} />
          <NavButton icon={<Trash2 />} label="Review Queue" active={view === 'recommendations'} onClick={() => setView('recommendations')} />
          <NavButton icon={<SlidersHorizontal />} label="Campaigns" active={view === 'campaigns'} onClick={() => setView('campaigns')} />
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
            activityRollupTotal={diagnosticActivityTotal(integrationDiagnostics, activityRollups.length)}
            storageLedger={storageLedger}
            notifications={notifications}
            protectionRequests={protectionRequests}
            jobs={jobs}
            jobDetails={jobDetails}
            onReadNotification={(id) => void markNotificationRead(id)}
            onDecideProtection={(id, approve) => void decideProtectionRequest(id, approve)}
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
        {view === 'campaigns' && (
          <CampaignsView
            campaigns={campaigns}
            templates={campaignTemplates}
            results={campaignResults}
            whatIf={campaignWhatIf}
            publicationPreviews={publicationPreviews}
            runs={campaignRuns}
            busy={busy}
            onTemplateCreate={(id) => void createCampaignFromTemplate(id)}
            onSave={(campaign) => void saveCampaign(campaign)}
            onSimulate={(id) => void simulateCampaign(id)}
            onWhatIf={(id) => void whatIfCampaign(id)}
            onPublishPreview={(id, input) => void previewCampaignPublication(id, input)}
            onRun={(id) => void runCampaign(id)}
            onDelete={(id) => void deleteCampaign(id)}
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
            requestSources={requestSources}
            requestSignals={requestSignals}
            pathMappings={pathMappings}
            unmappedItems={unmappedItems}
            jobs={jobs}
            jobDetails={jobDetails}
            onProviderUpdate={(provider, setting) => void updateProviderSetting(provider, setting)}
            onIntegrationUpdate={(integration, setting) => void updateIntegrationSetting(integration, setting)}
            onIntegrationRefresh={(id) => void refreshIntegration(id)}
            onIntegrationSync={(id) => void syncIntegration(id)}
            onRequestSourceUpdate={(id, source) => void updateRequestSource(id, source)}
            onRequestSourceSync={(id) => void syncRequestSource(id)}
            onTautulliSync={() => void syncTautulli()}
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
            appearance={appearance}
            onAppearanceSave={(nextAppearance) => void updateAppearance(nextAppearance)}
            onBackup={() => void createBackup()}
            onSupportBundle={() => void createSupportBundle()}
            onRestore={(name, dryRun) => void restoreBackup(name, dryRun)}
            backups={backups}
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
  const [email, setEmail] = useState('');
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
  activityRollupTotal,
  storageLedger,
  notifications,
  protectionRequests,
  jobs,
  jobDetails,
  onReadNotification,
  onDecideProtection,
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
  activityRollupTotal: number;
  storageLedger: StorageLedger | null;
  notifications: StewardshipNotification[];
  protectionRequests: ProtectionRequest[];
  jobs: Job[];
  jobDetails: Record<string, JobDetail>;
  onReadNotification: (id: string) => void;
  onDecideProtection: (id: string, approve: boolean) => void;
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
        <Stat icon={<Server />} label="Activity Items" value={String(activityRollupTotal)} />
        <Stat icon={<Trash2 />} label="Cold Suggestions" value={String(activityRecommendations.length)} />
        <Stat icon={<PlayCircle />} label="Never Watched" value={String(neverWatched)} />
        <Stat icon={<ShieldCheck />} label="Verified Savings" value={formatBytes(verifiedSavings)} />
        <Stat icon={<Bot />} label="AI Mode" value="Advisory" />
      </div>
      {storageLedger && <StorageLedgerPanel ledger={storageLedger} />}
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
      <div className="split">
        <NotificationsPanel notifications={notifications} onRead={onReadNotification} />
        <ProtectionQueuePanel requests={protectionRequests} onDecide={onDecideProtection} />
      </div>
    </section>
  );
}

function StorageLedgerPanel({ ledger }: { ledger: StorageLedger }) {
  return (
    <section className="panel ledger-panel">
      <div className="panel-heading">
        <div>
          <h2>Storage Ledger</h2>
          <span>verified savings are separated from estimates</span>
        </div>
        <span className="status-pill">suggest-only</span>
      </div>
      <div className="ledger-grid">
        <Signal label="Local proof" value={formatBytes(ledger.locallyVerifiedBytes)} />
        <Signal label="Mapped estimate" value={formatBytes(ledger.mappedEstimateBytes)} />
        <Signal label="Server reported" value={formatBytes(ledger.serverReportedBytes)} />
        <Signal label="Blocked unmapped" value={formatBytes(ledger.blockedUnmappedBytes)} />
        <Signal label="Protected" value={formatBytes(ledger.protectedBytes)} />
        <Signal label="Accepted manual" value={formatBytes(ledger.acceptedManualBytes)} />
        <Signal label="Requested media" value={formatBytes(ledger.requestedMediaBytes)} />
        <Signal label="Total verified" value={formatBytes(ledger.totalVerifiedBytes)} />
      </div>
    </section>
  );
}

function NotificationsPanel({ notifications, onRead }: { notifications: StewardshipNotification[]; onRead: (id: string) => void }) {
  return (
    <section className="panel">
      <div className="panel-heading">
        <h2>Notifications</h2>
        <span>{notifications.length} unread</span>
      </div>
      <div className="compact-list">
        {notifications.slice(0, 6).map((notification) => (
          <div className="compact-row" key={notification.id}>
            <div>
              <strong>{notification.title}</strong>
              <span>{notification.body || notification.eventType || notification.level}</span>
            </div>
            <button className="secondary-button compact-action" type="button" onClick={() => onRead(notification.id)}>
              <Check size={16} />
              Read
            </button>
          </div>
        ))}
        {notifications.length === 0 && <EmptyState icon={<Check />} text="No unread stewardship notifications." />}
      </div>
    </section>
  );
}

function ProtectionQueuePanel({ requests, onDecide }: { requests: ProtectionRequest[]; onDecide: (id: string, approve: boolean) => void }) {
  return (
    <section className="panel">
      <div className="panel-heading">
        <h2>Protection Requests</h2>
        <span>{requests.length} pending</span>
      </div>
      <div className="compact-list">
        {requests.slice(0, 6).map((request) => (
          <div className="compact-row" key={request.id || `${request.title}-${request.requestedBy}`}>
            <div>
              <strong>{request.title}</strong>
              <span>{request.reason || `Requested by ${request.requestedBy}`}</span>
            </div>
            {request.id && (
              <div className="button-row">
                <button className="secondary-button compact-action" type="button" onClick={() => onDecide(request.id!, false)}>
                  Decline
                </button>
                <button className="secondary-button compact-action" type="button" onClick={() => onDecide(request.id!, true)}>
                  <ShieldCheck size={16} />
                  Protect
                </button>
              </div>
            )}
          </div>
        ))}
        {requests.length === 0 && <EmptyState icon={<ShieldCheck />} text="No pending protection requests." />}
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
  const [expandedPaths, setExpandedPaths] = useState<Record<string, boolean>>({});

  return (
    <section className="queue">
      {recommendations.map((rec) => {
        const proof = evidence[rec.id];
        const estimatedSavingsBytes = recommendationEstimatedSavings(rec);
        const verifiedSavingsBytes = recommendationVerifiedSavings(rec);
        const storageCertainty = rec.evidence?.storageCertainty || storageCertaintyForVerification(rec.verification);
        const subjectKind = rec.evidence?.subjectKind;
        const itemCount = recommendationItemCount(rec);
        const pathGroups = groupAffectedPaths(rec.affectedPaths, subjectKind);
        const isExpanded = Boolean(expandedPaths[rec.id]);
        const previewPaths = rec.affectedPaths.slice(0, 3);
        const hiddenPathCount = Math.max(rec.affectedPaths.length - previewPaths.length, 0);
        return (
          <article className="recommendation-card" key={rec.id}>
            <div className="rec-main">
              <div className="rec-icon"><ShieldCheck size={22} /></div>
              <div>
                <div className="rec-title-row">
                  <h2>{rec.title}</h2>
                  <div className="rec-badges">
                    <span className={`confidence-pill ${confidenceTone(rec.confidence)}`}>{formatConfidence(rec.confidence)} confidence</span>
                    <span className="action-pill">{formatRecommendationAction(rec.action)}</span>
                    <span className="status-pill">{formatRecommendationState(rec.state)}</span>
                  </div>
                </div>
                <p>{rec.explanation}</p>
                <div className="affected-media">
                  <div className="affected-media-header">
                    <span>{formatAffectedSummary(itemCount, pathGroups.length, subjectKind)}</span>
                    {rec.affectedPaths.length > 0 && (
                      <button
                        className="secondary-button compact-button"
                        type="button"
                        aria-expanded={isExpanded}
                        onClick={() => setExpandedPaths((current) => ({ ...current, [rec.id]: !isExpanded }))}
                      >
                        <FolderOpen size={15} />
                        {isExpanded ? 'Hide affected files' : `Show ${rec.affectedPaths.length} affected files`}
                      </button>
                    )}
                  </div>
                  {isExpanded ? (
                    <div className="path-groups">
                      {pathGroups.map((group) => (
                        <div className="path-group" key={group.label}>
                          <div className="path-group-header">
                            <strong>{group.label}</strong>
                            <span>{formatFileCount(group.count)}</span>
                          </div>
                          <div className="path-list">
                            {group.paths.map((path) => <code key={path}>{path}</code>)}
                          </div>
                        </div>
                      ))}
                    </div>
                  ) : (
                    <div className="path-list path-list-preview">
                      {previewPaths.map((path) => <code key={path}>{path}</code>)}
                      {hiddenPathCount > 0 && <span className="path-overflow">+{hiddenPathCount} more affected files hidden</span>}
                    </div>
                  )}
                </div>
                <div className="rec-evidence">
                  <Signal label="Source" value={rec.serverId || 'Local scan'} />
                  <Signal label="Scope" value={formatFileCount(itemCount)} />
                  <Signal label="Groups" value={formatGroupCount(pathGroups.length, subjectKind)} />
                  <Signal label="Last Played" value={rec.lastPlayedAt ? formatDateTime(rec.lastPlayedAt) : rec.serverId ? 'Never' : 'N/A'} />
                  <Signal label="Plays" value={String(rec.playCount ?? 0)} />
                  <Signal label="Users" value={String(rec.uniqueUsers ?? 0)} />
                  <Signal label="Evidence" value={formatVerification(rec.verification)} />
                  <Signal label="Confidence" value={formatConfidence(rec.confidence)} />
                  <Signal label="Estimated savings" value={formatBytes(estimatedSavingsBytes)} />
                  <Signal label="Verified savings" value={formatBytes(verifiedSavingsBytes)} />
                </div>
                <div className="proof-summary">
                  <ProofRow label="Why suggested" value={recommendationWhySuggested(rec)} />
                  <ProofRow label="Confidence basis" value={rec.evidence?.confidenceBasis || recommendationConfidenceBasis(rec)} />
                  <ProofRow label="Storage certainty" value={storageCertaintyDefinition(rec.verification || rec.evidence?.storageBasis || storageCertainty)} />
                  <ProofRow label="Activity proof" value={recommendationActivityProof(rec)} />
                  <ProofRow label="Safety" value="Mediarr will not delete this. Accepting marks it for manual action only." />
                </div>
                <div className={storageCertainty === 'verified' ? 'notice success' : 'notice warning'}>
                  {storageCertaintyDescription(storageCertainty)}
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
              <span>{formatBytes(estimatedSavingsBytes)}</span>
              <small>{formatConfidence(rec.confidence)} • {formatVerification(rec.verification)} • {rec.source}</small>
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
  const storageDefinition = storageCertaintyDefinition(evidence.storage.verification || evidence.storage.basis || evidence.storage.certainty);
  return (
    <div className="evidence-panel">
      <div className="signal-grid">
        <Signal label="Storage" value={formatVerification(evidence.storage.verification)} />
        <Signal label="Risk" value={evidence.storage.risk} />
        <Signal label="Estimated" value={formatBytes(evidence.storage.estimatedSavingsBytes || evidence.storage.spaceSavedBytes)} />
        <Signal label="Verified" value={formatBytes(evidence.storage.verifiedSavingsBytes || 0)} />
        <Signal label="Rule source" value={evidence.source.rule.replace('rule:', '')} />
        <Signal label="Threshold" value={thresholdProofLabel(evidence.raw)} />
        <Signal label="Last played" value={evidence.activity.lastPlayedAt ? formatDateTime(evidence.activity.lastPlayedAt) : evidence.activity.serverId ? 'Never watched by imported users' : 'N/A'} />
        <Signal label="Watched users" value={String(evidence.activity.uniqueUsers)} />
        <Signal label="Favorites/protection" value={evidence.activity.favoriteCount > 0 ? `${evidence.activity.favoriteCount} signal(s)` : 'None imported'} />
      </div>
      <ProofRow label="Storage certainty" value={storageDefinition} />
      {evidence.raw.confidenceBasis && <ProofRow label="Confidence basis" value={evidence.raw.confidenceBasis} />}
      <ProofRow label="Safety" value="Mediarr will not delete this. Accepting marks it for manual action only." />
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

function CampaignsView({
  campaigns,
  templates,
  results,
  whatIf,
  publicationPreviews,
  runs,
  busy,
  onTemplateCreate,
  onSave,
  onSimulate,
  onWhatIf,
  onPublishPreview,
  onRun,
  onDelete,
}: {
  campaigns: Campaign[];
  templates: CampaignTemplate[];
  results: Record<string, CampaignResult>;
  whatIf: Record<string, WhatIfSimulation>;
  publicationPreviews: Record<string, PublicationPlan>;
  runs: Record<string, CampaignRun[]>;
  busy: boolean;
  onTemplateCreate: (id: string) => void;
  onSave: (campaign: Campaign) => void;
  onSimulate: (id: string) => void;
  onWhatIf: (id: string) => void;
  onPublishPreview: (id: string, input: PublicationInput) => void;
  onRun: (id: string) => void;
  onDelete: (id: string) => void;
}) {
  const safeCampaigns = campaigns || [];
  const [selectedID, setSelectedID] = useState(safeCampaigns[0]?.id ?? '__new__');
  const [draft, setDraft] = useState<Campaign>(() => newCampaignDraft());
  const selectedCampaign = safeCampaigns.find((campaign) => campaign.id === selectedID);
  const activeID = selectedCampaign?.id ?? draft.id;
  const result = results[activeID];
  const simulation = whatIf[activeID];
  const publicationPreview = publicationPreviews[activeID];
  const campaignRuns = runs[activeID] || [];
  const [publicationServer, setPublicationServer] = useState('jellyfin');
  const [collectionTitle, setCollectionTitle] = useState('Leaving Soon');
  const [minimumVerification, setMinimumVerification] = useState('local_verified');

  useEffect(() => {
    if (selectedCampaign) {
      setDraft(cloneCampaign(selectedCampaign));
      return;
    }
    if (selectedID !== '__new__') {
      setSelectedID(safeCampaigns[0]?.id ?? '__new__');
      return;
    }
    if (!draft.id) {
      setDraft(newCampaignDraft());
    }
  }, [safeCampaigns, selectedCampaign, selectedID, draft.id]);

  function updateRule(index: number, updates: Partial<CampaignRule>) {
    setDraft((current) => ({
      ...current,
      rules: (current.rules || []).map((rule, ruleIndex) => ruleIndex === index ? { ...rule, ...updates } : rule),
    }));
  }

  function updateRuleValue(index: number, raw: string) {
    const rule = draft.rules[index];
    if (!rule) {
      return;
    }
    if (rule.operator === 'in' || rule.operator === 'not_in') {
      updateRule(index, { value: '', values: raw.split(',').map((value) => value.trim()).filter(Boolean) });
      return;
    }
    updateRule(index, { value: raw, values: [] });
  }

  function save(event: React.FormEvent) {
    event.preventDefault();
    onSave(normalizeCampaignDraft(draft));
  }

  return (
    <section className="campaign-layout">
      <div className="campaign-rail">
        <div className="panel-heading">
          <div>
            <h2>Stewardship Campaigns</h2>
              <span>{safeCampaigns.length} saved</span>
          </div>
          <button
            className="secondary-button compact-action"
            type="button"
            onClick={() => {
              const next = newCampaignDraft();
              setSelectedID('__new__');
              setDraft(next);
            }}
          >
            New
          </button>
        </div>
        <div className="campaign-list">
          {safeCampaigns.map((campaign) => (
            <button
              className={campaign.id === selectedID ? 'campaign-list-item active' : 'campaign-list-item'}
              type="button"
              key={campaign.id}
              onClick={() => setSelectedID(campaign.id)}
            >
              <strong>{campaign.name}</strong>
              <span>{campaign.enabled ? 'Enabled' : 'Paused'} • {campaign.rules.length} rules</span>
              <small>{campaign.lastRunAt ? `Last run ${formatDateTime(campaign.lastRunAt)}` : 'No runs yet'}</small>
            </button>
          ))}
          {safeCampaigns.length === 0 && <EmptyState icon={<SlidersHorizontal />} text="No campaigns saved." />}
        </div>
        <TemplateGallery templates={templates} busy={busy} onCreate={onTemplateCreate} />
      </div>
      <form className="campaign-editor" onSubmit={(event) => void save(event)}>
        <div className="panel-heading">
          <div>
            <h2>{selectedCampaign ? 'Edit Campaign' : 'New Campaign'}</h2>
            <span>Suggest-only • evidence-backed • media-server activity</span>
          </div>
          <div className="button-row">
            {selectedCampaign && (
              <button className="secondary-button" type="button" disabled={busy} onClick={() => onDelete(selectedCampaign.id)}>
                <Trash2 size={16} />
                Delete
              </button>
            )}
            <button className="primary-button" type="submit" disabled={busy || !draft.name.trim()}>
              <Save size={16} />
              Save
            </button>
          </div>
        </div>
        <div className="campaign-form-grid">
          <label>
            Name
            <input value={draft.name} onChange={(event) => setDraft((current) => ({ ...current, name: event.target.value }))} required />
          </label>
          <label>
            ID
            <input value={draft.id} onChange={(event) => setDraft((current) => ({ ...current, id: slugCampaignID(event.target.value) }))} required />
          </label>
          <label className="wide-field">
            Description
            <input value={draft.description || ''} onChange={(event) => setDraft((current) => ({ ...current, description: event.target.value }))} />
          </label>
          <label>
            Target
            <select value={(draft.targetKinds || [])[0] || ''} onChange={(event) => setDraft((current) => ({ ...current, targetKinds: event.target.value ? [event.target.value] : [] }))}>
              <option value="">All media</option>
              <option value="movie">Movies</option>
              <option value="series">Series</option>
              <option value="anime">Anime</option>
            </select>
          </label>
          <label>
            Minimum confidence
            <input value={String(Math.round((draft.minimumConfidence || 0) * 100))} onChange={(event) => setDraft((current) => ({ ...current, minimumConfidence: clampPercentInput(event.target.value) / 100 }))} type="number" min={0} max={100} />
          </label>
          <label>
            Minimum storage GB
            <input value={String(Math.round((draft.minimumStorageBytes || 0) / 1_000_000_000))} onChange={(event) => setDraft((current) => ({ ...current, minimumStorageBytes: Math.max(0, Number.parseInt(event.target.value, 10) || 0) * 1_000_000_000 }))} type="number" min={0} />
          </label>
          <label className="checkbox-row">
            <input checked={draft.enabled} onChange={(event) => setDraft((current) => ({ ...current, enabled: event.target.checked }))} type="checkbox" />
            Enabled
          </label>
          <label className="checkbox-row">
            <input checked={draft.requireAllRules} onChange={(event) => setDraft((current) => ({ ...current, requireAllRules: event.target.checked }))} type="checkbox" />
            Require all rules
          </label>
        </div>
        <div className="rule-builder">
          <div className="panel-heading">
            <h2>Rules</h2>
            <button className="secondary-button compact-action" type="button" onClick={() => setDraft((current) => ({ ...current, rules: [...(current.rules || []), defaultCampaignRule()] }))}>
              Add rule
            </button>
          </div>
          {(draft.rules || []).map((rule, index) => (
            <div className="rule-row" key={`${index}-${rule.field}-${rule.operator}`}>
              <select value={rule.field} onChange={(event) => updateRule(index, { field: event.target.value as CampaignRuleField })}>
                {campaignRuleFields.map((field) => <option key={field.value} value={field.value}>{field.label}</option>)}
              </select>
              <select value={rule.operator} onChange={(event) => updateRule(index, { operator: event.target.value as CampaignRuleOperator })}>
                {campaignRuleOperators.map((operator) => <option key={operator.value} value={operator.value}>{operator.label}</option>)}
              </select>
              <input
                value={rule.operator === 'in' || rule.operator === 'not_in' ? (rule.values || []).join(', ') : rule.value || ''}
                disabled={rule.operator === 'is_empty' || rule.operator === 'is_not_empty'}
                onChange={(event) => updateRuleValue(index, event.target.value)}
              />
              <button className="icon-button" type="button" title="Remove rule" aria-label="Remove rule" onClick={() => setDraft((current) => ({ ...current, rules: (current.rules || []).filter((_, ruleIndex) => ruleIndex !== index) }))}>
                <Trash2 size={15} />
              </button>
            </div>
          ))}
        </div>
        <div className="button-row campaign-run-row">
          <button className="secondary-button" type="button" disabled={busy || !selectedCampaign} onClick={() => selectedCampaign && onSimulate(selectedCampaign.id)}>
            <SearchCheck size={16} />
            Simulate
          </button>
          <button className="secondary-button" type="button" disabled={busy || !selectedCampaign} onClick={() => selectedCampaign && onWhatIf(selectedCampaign.id)}>
            <Database size={16} />
            What-if
          </button>
          <button className="primary-button" type="button" disabled={busy || !selectedCampaign} onClick={() => selectedCampaign && onRun(selectedCampaign.id)}>
            <PlayCircle size={16} />
            Run campaign
          </button>
        </div>
        <div className="publication-controls">
          <label>
            Collection
            <input value={collectionTitle} onChange={(event) => setCollectionTitle(event.target.value)} />
          </label>
          <label>
            Server
            <select value={publicationServer} onChange={(event) => setPublicationServer(event.target.value)}>
              <option value="jellyfin">Jellyfin</option>
              <option value="plex">Plex</option>
            </select>
          </label>
          <label>
            Minimum proof
            <select value={minimumVerification} onChange={(event) => setMinimumVerification(event.target.value)}>
              <option value="local_verified">Local verified</option>
              <option value="path_mapped">Path mapped</option>
              <option value="server_reported">Server reported</option>
            </select>
          </label>
          <button
            className="secondary-button"
            type="button"
            disabled={busy || !selectedCampaign}
            onClick={() => selectedCampaign && onPublishPreview(selectedCampaign.id, { serverId: publicationServer, collectionTitle, minimumVerification })}
          >
            <Archive size={16} />
            Preview collection
          </button>
        </div>
      </form>
      <WhatIfPanel simulation={simulation} />
      <CampaignResultPanel result={result} />
      <PublicationPreviewPanel preview={publicationPreview} />
      <CampaignRunsPanel runs={campaignRuns} />
    </section>
  );
}

function TemplateGallery({ templates, busy, onCreate }: { templates: CampaignTemplate[]; busy: boolean; onCreate: (id: string) => void }) {
  return (
    <section className="template-gallery">
      <div className="panel-heading">
        <h2>Templates</h2>
        <span>{templates.length} built in</span>
      </div>
      {templates.slice(0, 6).map((template) => (
        <button className="template-card" type="button" key={template.id} disabled={busy} onClick={() => onCreate(template.id)}>
          <strong>{template.name}</strong>
          <span>{template.description}</span>
          <small>{template.campaign.targetKinds.join(', ') || 'all'} • {template.campaign.rules.length} rules</small>
        </button>
      ))}
      {templates.length === 0 && <EmptyState icon={<Archive />} text="No campaign templates loaded." />}
    </section>
  );
}

function WhatIfPanel({ simulation }: { simulation?: WhatIfSimulation }) {
  return (
    <section className="panel campaign-results">
      <div className="panel-heading">
        <h2>What-if</h2>
        <span>{simulation ? `${simulation.matched} matched` : 'Pending'}</span>
      </div>
      {simulation ? (
        <div className="signal-grid">
          <Signal label="Estimated" value={formatBytes(simulation.estimatedBytes)} />
          <Signal label="Verified" value={formatBytes(simulation.verifiedBytes)} />
          <Signal label="Suppressed" value={String(simulation.suppressed)} />
          <Signal label="Unmapped" value={String(simulation.blockedUnmapped)} />
          <Signal label="Requests" value={String(simulation.requestConflicts)} />
          <Signal label="Protections" value={String(simulation.protectionConflicts)} />
        </div>
      ) : (
        <EmptyState icon={<Database />} text="No what-if simulation loaded." />
      )}
    </section>
  );
}

function PublicationPreviewPanel({ preview }: { preview?: PublicationPlan }) {
  return (
    <section className="panel campaign-results">
      <div className="panel-heading">
        <h2>Leaving Soon Preview</h2>
        <span>{preview ? preview.status : 'Pending'}</span>
      </div>
      {preview ? (
        <>
          <div className="signal-grid">
            <Signal label="Publishable" value={String(preview.publishableItems)} />
            <Signal label="Blocked" value={String(preview.blockedItems)} />
            <Signal label="Publishable bytes" value={formatBytes(preview.publishableEstimatedBytes)} />
            <Signal label="Blocked bytes" value={formatBytes(preview.blockedEstimatedBytes)} />
          </div>
          <div className="campaign-result-list">
            {preview.items.slice(0, 8).map((item) => (
              <article className={item.publishable ? 'campaign-result-item' : 'campaign-result-item suppressed'} key={`${item.title}-${item.externalItemId || item.blockedReason}`}>
                <div>
                  <strong>{item.title}</strong>
                  <span>{formatVerification(item.verification)} • {item.externalItemId || 'no external id'}</span>
                </div>
                <div className="campaign-result-metrics">
                  <span>{formatBytes(item.estimatedBytes)}</span>
                  <small>{item.publishable ? 'ready for collection' : item.blockedReason}</small>
                </div>
              </article>
            ))}
          </div>
        </>
      ) : (
        <EmptyState icon={<Archive />} text="No collection preview loaded." />
      )}
    </section>
  );
}

function CampaignResultPanel({ result }: { result?: CampaignResult }) {
  if (!result) {
    return (
      <section className="panel campaign-results">
        <div className="panel-heading">
          <h2>Simulation</h2>
          <span>Pending</span>
        </div>
        <EmptyState icon={<SearchCheck />} text="No simulation loaded." />
      </section>
    );
  }
  return (
    <section className="panel campaign-results">
      <div className="panel-heading">
        <h2>Simulation</h2>
        <span>{result.enabled ? 'Enabled' : 'Paused'}</span>
      </div>
      <div className="signal-grid">
        <Signal label="Matched" value={String(result.matched)} />
        <Signal label="Suppressed" value={String(result.suppressed)} />
        <Signal label="Estimated" value={formatBytes(result.totalEstimatedSavingsBytes)} />
        <Signal label="Verified" value={formatBytes(result.totalVerifiedSavingsBytes)} />
        <Signal label="Confidence avg" value={formatConfidence(result.confidenceAverage)} />
        <Signal label="Confidence range" value={`${formatConfidence(result.confidenceMin)}-${formatConfidence(result.confidenceMax)}`} />
      </div>
      <div className="campaign-result-list">
        {result.items.slice(0, 12).map((item) => (
          <article className={item.suppressed ? 'campaign-result-item suppressed' : 'campaign-result-item'} key={item.candidate.key}>
            <div>
              <strong>{item.candidate.title}</strong>
              <span>{item.candidate.kind} • {formatVerification(item.candidate.verification)} • {formatConfidence(item.candidate.confidence)}</span>
            </div>
            <div className="campaign-result-metrics">
              <span>{formatBytes(item.candidate.estimatedSavingsBytes)}</span>
              <small>{item.suppressed ? item.suppressionReasons.join(', ') : item.matchedRules.map((rule) => formatCampaignRule(rule.rule)).join('; ')}</small>
            </div>
          </article>
        ))}
        {result.items.length === 0 && <EmptyState icon={<ShieldCheck />} text="No media matched this campaign." />}
      </div>
    </section>
  );
}

function CampaignRunsPanel({ runs }: { runs: CampaignRun[] }) {
  return (
    <section className="panel campaign-runs">
      <div className="panel-heading">
        <h2>Run History</h2>
        <span>{runs.length} runs</span>
      </div>
      <div className="compact-list">
        {runs.slice(0, 8).map((run) => (
          <div className="compact-row" key={run.id}>
            <div>
              <strong>{run.status}</strong>
              <span>{formatDateTime(run.startedAt)} • {run.matched} matched • {run.suppressed} suppressed</span>
            </div>
            <strong>{formatBytes(run.estimatedSavingsBytes)}</strong>
          </div>
        ))}
        {runs.length === 0 && <EmptyState icon={<Archive />} text="No campaign runs recorded." />}
      </div>
    </section>
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
  requestSources,
  requestSignals,
  pathMappings,
  unmappedItems,
  jobs,
  jobDetails,
  onProviderUpdate,
  onIntegrationUpdate,
  onIntegrationRefresh,
  onIntegrationSync,
  onRequestSourceUpdate,
  onRequestSourceSync,
  onTautulliSync,
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
  requestSources: RequestSource[];
  requestSignals: RequestSignal[];
  pathMappings: PathMapping[];
  unmappedItems: MediaServerItem[];
  jobs: Job[];
  jobDetails: Record<string, JobDetail>;
  onProviderUpdate: (provider: string, setting: ProviderSettingInput) => void;
  onIntegrationUpdate: (integration: string, setting: IntegrationSettingInput) => void;
  onIntegrationRefresh: (id: string) => void;
  onIntegrationSync: (id: string) => void;
  onRequestSourceUpdate: (id: string, source: RequestSourceInput) => void;
  onRequestSourceSync: (id: string) => void;
  onTautulliSync: () => void;
  onPathMappingSave: (mapping: Partial<PathMapping> & Pick<PathMapping, 'serverPathPrefix' | 'localPathPrefix'>) => void;
  onPathMappingVerify: (id: string) => void;
  onPathMappingDelete: (id: string) => void;
  onCancelJob: (id: string) => void;
  onRetryJob: (id: string) => void;
  busy: boolean;
}) {
  const mediaServers = integrations.filter((integration) => integration.kind === 'media_server');
  const activityTargets = integrations.filter((integration) => integration.kind === 'activity_analytics');
  const activeSyncJobs = jobs.filter((job) => isActiveJob(job) && job.kind.endsWith('_sync'));
  return (
    <section className="view-grid">
      <div className="integration-grid">
        {mediaServers.map((integration) => {
          const activeJob = activeSyncJobs.find((job) => job.targetId === integration.id);
          const diagnostics = integrationDiagnostics[integration.id] ?? null;
          return (
            <MediaServerCard
              key={integration.id}
              integration={integration}
              setting={integrationSettings.find((setting) => setting.integration === integration.id)}
              job={syncJobs[integration.id] ?? null}
              diagnostics={diagnostics}
              importedItems={diagnostics ? diagnosticImportedItems(diagnostics) : integrationItems.filter((item) => item.serverId === integration.id).length}
              activityCount={diagnostics ? diagnostics.summary.activityRollups : activityRollups.filter((rollup) => rollup.serverId === integration.id).length}
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
      <StewardshipSignalsPanel
        requestSources={requestSources}
        requestSignals={requestSignals}
        activityTargets={activityTargets}
        settings={integrationSettings}
        activeJobs={activeSyncJobs}
        busy={busy}
        onRequestSourceUpdate={onRequestSourceUpdate}
        onRequestSourceSync={onRequestSourceSync}
        onIntegrationUpdate={onIntegrationUpdate}
        onTautulliSync={onTautulliSync}
      />
      <PathMappingWorkbench
        integrations={mediaServers}
        mappings={pathMappings}
        unmappedItems={unmappedItems}
        unmappedTotal={diagnosticUnmappedTotal(mediaServers, integrationDiagnostics, unmappedItems.length)}
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

function StewardshipSignalsPanel({
  requestSources,
  requestSignals,
  activityTargets,
  settings,
  activeJobs,
  busy,
  onRequestSourceUpdate,
  onRequestSourceSync,
  onIntegrationUpdate,
  onTautulliSync,
}: {
  requestSources: RequestSource[];
  requestSignals: RequestSignal[];
  activityTargets: Integration[];
  settings: IntegrationSetting[];
  activeJobs: Job[];
  busy: boolean;
  onRequestSourceUpdate: (id: string, source: RequestSourceInput) => void;
  onRequestSourceSync: (id: string) => void;
  onIntegrationUpdate: (integration: string, setting: IntegrationSettingInput) => void;
  onTautulliSync: () => void;
}) {
  const seerr = requestSources.find((source) => source.id === 'seerr');
  const tautulli = activityTargets.find((target) => target.id === 'tautulli');
  const tautulliSetting = settings.find((setting) => setting.integration === 'tautulli');
  const tautulliJob = activeJobs.find((job) => job.kind === 'tautulli_sync');
  const [seerrURL, setSeerrURL] = useState(seerr?.baseUrl || '');
  const [seerrKey, setSeerrKey] = useState('');
  const [tautulliURL, setTautulliURL] = useState(tautulliSetting?.baseUrl || '');
  const [tautulliKey, setTautulliKey] = useState('');

  useEffect(() => {
    setSeerrURL(seerr?.baseUrl || '');
    setSeerrKey('');
  }, [seerr?.baseUrl, seerr?.apiKeyConfigured]);

  useEffect(() => {
    setTautulliURL(tautulliSetting?.baseUrl || '');
    setTautulliKey('');
  }, [tautulliSetting?.baseUrl, tautulliSetting?.apiKeyConfigured]);

  return (
    <section className="signals-panel">
      <div className="panel-heading">
        <div>
          <h2>Stewardship Signals</h2>
          <p>request intent and external activity enrich suggestions without deleting media</p>
        </div>
        <span className="status-pill">{requestSignals.length} requests</span>
      </div>
      <div className="signals-columns">
        <form className="signal-source-card" onSubmit={(event) => {
          event.preventDefault();
          onRequestSourceUpdate('seerr', { kind: 'seerr', name: 'Seerr', baseUrl: seerrURL, apiKey: seerrKey || undefined, enabled: true });
        }}>
          <div className="panel-heading">
            <h2>Seerr</h2>
            <span>{seerr?.apiKeyConfigured ? `key ...${seerr.apiKeyLast4 || ''}` : 'not connected'}</span>
          </div>
          <label>
            URL
            <input value={seerrURL} onChange={(event) => setSeerrURL(event.target.value)} placeholder="http://jellyseerr:5055" />
          </label>
          <label>
            API key
            <input value={seerrKey} onChange={(event) => setSeerrKey(event.target.value)} type="password" placeholder={seerr?.apiKeyConfigured ? `Configured ...${seerr.apiKeyLast4 || ''}` : 'Paste key'} />
          </label>
          <div className="button-row">
            <button className="secondary-button" type="submit" disabled={busy || !seerrURL.trim()}>
              <Save size={16} />
              Save
            </button>
            <button className="primary-button" type="button" disabled={busy || !seerr?.apiKeyConfigured} onClick={() => onRequestSourceSync('seerr')}>
              <Database size={16} />
              Sync requests
            </button>
          </div>
        </form>
        <form className="signal-source-card" onSubmit={(event) => {
          event.preventDefault();
          onIntegrationUpdate('tautulli', { baseUrl: tautulliURL, apiKey: tautulliKey || undefined, autoSyncEnabled: false });
        }}>
          <div className="panel-heading">
            <h2>{tautulli?.name || 'Tautulli'}</h2>
            <span>{tautulli?.status || 'not_configured'}</span>
          </div>
          <label>
            URL
            <input value={tautulliURL} onChange={(event) => setTautulliURL(event.target.value)} placeholder="http://tautulli:8181" />
          </label>
          <label>
            API key
            <input value={tautulliKey} onChange={(event) => setTautulliKey(event.target.value)} type="password" placeholder={tautulliSetting?.apiKeyConfigured ? `Configured ...${tautulliSetting.apiKeyLast4 || ''}` : 'Paste key'} />
          </label>
          <div className="button-row">
            <button className="secondary-button" type="submit" disabled={busy || !tautulliURL.trim()}>
              <Save size={16} />
              Save
            </button>
            <button className="primary-button" type="button" disabled={busy || !tautulliSetting?.apiKeyConfigured} onClick={onTautulliSync}>
              <Activity size={16} />
              {tautulliJob ? 'Syncing' : 'Sync activity'}
            </button>
          </div>
        </form>
      </div>
      <div className="request-signal-list">
        {requestSignals.slice(0, 8).map((signal) => (
          <div className="compact-row" key={`${signal.sourceId}-${signal.externalRequestId}`}>
            <div>
              <strong>{signal.title || signal.externalMediaId || signal.externalRequestId}</strong>
              <span>{signal.mediaType} • {signal.status} • {signal.availability} • {signal.requestedBy || 'unknown requester'}</span>
            </div>
            <span className="status-pill">{signal.providerIds.tmdb || signal.providerIds.tvdb || signal.providerIds.imdb || 'no id'}</span>
          </div>
        ))}
        {requestSignals.length === 0 && <EmptyState icon={<Archive />} text="No request signals imported." />}
      </div>
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
          <span>{formatConfidence(diagnostics.topRecommendations[0].confidence)}</span>
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
  unmappedTotal,
  busy,
  onSave,
  onVerify,
  onDelete,
}: {
  integrations: Integration[];
  mappings: PathMapping[];
  unmappedItems: MediaServerItem[];
  unmappedTotal: number;
  busy: boolean;
  onSave: (mapping: Partial<PathMapping> & Pick<PathMapping, 'serverPathPrefix' | 'localPathPrefix'>) => void;
  onVerify: (id: string) => void;
  onDelete: (id: string) => void;
}) {
  const suggestedServerPathPrefix = useMemo(() => suggestServerPathPrefix(unmappedItems), [unmappedItems]);
  const [serverId, setServerID] = useState(integrations[0]?.id || 'jellyfin');
  const [serverPathPrefix, setServerPathPrefix] = useState(() => suggestedServerPathPrefix || '/mnt/media');
  const [localPathPrefix, setLocalPathPrefix] = useState('/media');

  useEffect(() => {
    if (!integrations.some((integration) => integration.id === serverId)) {
      setServerID(integrations[0]?.id || 'jellyfin');
    }
  }, [integrations, serverId]);

  useEffect(() => {
    if (mappings.length > 0 || serverPathPrefix !== '/mnt/media') {
      return;
    }
    if (suggestedServerPathPrefix) {
      setServerPathPrefix(suggestedServerPathPrefix);
    }
  }, [mappings.length, serverPathPrefix, suggestedServerPathPrefix]);

  function submit(event: React.FormEvent) {
    event.preventDefault();
    onSave({ serverId, serverPathPrefix, localPathPrefix });
  }

  return (
    <section className="mapping-workbench">
      <div className="panel-heading">
        <div>
          <h2>Path Mapping</h2>
          <p>{unmappedTotal} unmapped server items require path proof</p>
        </div>
        <span className={unmappedTotal > 0 ? 'status-pill warning' : 'status-pill'}>{unmappedTotal > 0 ? 'Review' : 'Verified'}</span>
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
          {unmappedTotal > unmappedItems.length && (
            <div className="notice warning">Showing {unmappedItems.length} examples from {unmappedTotal} unmapped server items.</div>
          )}
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

function ProofRow({ label, value }: { label: string; value: string }) {
  return (
    <div className="proof-row">
      <span>{label}</span>
      <strong>{value}</strong>
    </div>
  );
}

function recommendationEstimatedSavings(rec: Recommendation): number {
  return parseEvidenceNumber(rec.evidence?.estimatedSavingsBytes, rec.spaceSavedBytes);
}

function recommendationVerifiedSavings(rec: Recommendation): number {
  const fallback = rec.verification === 'local_verified' ? rec.spaceSavedBytes : 0;
  return parseEvidenceNumber(rec.evidence?.verifiedSavingsBytes, fallback);
}

function recommendationWhySuggested(rec: Recommendation): string {
  const thresholdDays = rec.evidence?.thresholdDays;
  const action = formatRecommendationAction(rec.action).toLowerCase();
  if (thresholdDays) {
    return `${action} rule from ${rec.source}; threshold older than ${thresholdDays} days.`;
  }
  return `${action} rule from ${rec.source}.`;
}

function recommendationConfidenceBasis(rec: Recommendation): string {
  const parts = [formatVerification(rec.verification)];
  if (rec.evidence?.thresholdDays) {
    const days = rec.evidence.ageDays || rec.evidence.inactiveDays;
    parts.push(days ? `${days} days vs ${rec.evidence.thresholdDays} day threshold` : `${rec.evidence.thresholdDays} day threshold`);
  }
  if (rec.playCount && rec.playCount > 0) {
    parts.push(`${rec.playCount} prior plays`);
  }
  if (rec.uniqueUsers && rec.uniqueUsers > 0) {
    parts.push(`${rec.uniqueUsers} watched users`);
  }
  return parts.join('; ');
}

function recommendationActivityProof(rec: Recommendation): string {
  if (!rec.serverId) {
    return 'Local filesystem scan evidence; no media-server activity was imported for this recommendation.';
  }
  const users = rec.uniqueUsers ?? 0;
  const plays = rec.playCount ?? 0;
  const favorites = rec.favoriteCount ?? 0;
  const lastPlayed = rec.lastPlayedAt ? `last played ${formatDateTime(rec.lastPlayedAt)}` : 'never watched by imported users';
  const protectedText = favorites > 0 ? `${favorites} favorite/protected signal${favorites === 1 ? '' : 's'}` : 'no favorite/protected signals';
  return `${lastPlayed}; ${plays} total play${plays === 1 ? '' : 's'} across ${users} watched user${users === 1 ? '' : 's'}; ${protectedText}.`;
}

function thresholdProofLabel(raw: Record<string, string>): string {
  if (raw.thresholdDays) {
    return `Older than ${raw.thresholdDays} days`;
  }
  if (raw.ageDays) {
    return `${raw.ageDays} days old`;
  }
  if (raw.inactiveDays) {
    return `${raw.inactiveDays} days inactive`;
  }
  return 'Rule default';
}

function parseEvidenceNumber(value: string | undefined, fallback: number): number {
  if (!value) {
    return fallback;
  }
  const parsed = Number.parseInt(value, 10);
  return Number.isFinite(parsed) ? parsed : fallback;
}

function diagnosticImportedItems(diagnostics: IntegrationDiagnostics): number {
  const summary = diagnostics.summary;
  return summary.movies + summary.series + summary.episodes + summary.videos;
}

function diagnosticUnmappedTotal(integrations: Integration[], diagnostics: Record<string, IntegrationDiagnostics | null>, fallback: number): number {
  const totals = integrations
    .map((integration) => diagnostics[integration.id]?.summary.unmappedFiles)
    .filter((value): value is number => typeof value === 'number');
  if (totals.length === 0) {
    return fallback;
  }
  return totals.reduce((sum, value) => sum + value, 0);
}

function diagnosticActivityTotal(diagnostics: Record<string, IntegrationDiagnostics | null>, fallback: number): number {
  const totals = Object.values(diagnostics)
    .map((diagnosticsRow) => diagnosticsRow?.summary.activityRollups)
    .filter((value): value is number => typeof value === 'number');
  if (totals.length === 0) {
    return fallback;
  }
  return totals.reduce((sum, value) => sum + value, 0);
}

function suggestServerPathPrefix(items: MediaServerItem[]): string {
  const paths = items.map((item) => item.path || '').filter(Boolean);
  if (paths.length === 0) {
    return '';
  }
  const roots = paths.map((path) => mediaRootPrefix(path)).filter(Boolean);
  if (roots.length > 0 && roots.every((root) => root === roots[0])) {
    return roots[0];
  }
  return commonDirectoryPrefix(paths.slice(0, 25));
}

function mediaRootPrefix(path: string): string {
  const parts = path.split('/').filter(Boolean);
  const mediaMarkers = new Set(['media', 'movies', 'movie', 'tv-shows', 'tv', 'shows', 'series', 'anime']);
  const markerIndex = parts.findIndex((part, index) => index > 0 && mediaMarkers.has(part.toLowerCase()));
  if (markerIndex <= 0) {
    return '';
  }
  const marker = parts[markerIndex].toLowerCase();
  const end = marker === 'media' ? markerIndex + 1 : markerIndex;
  return '/' + parts.slice(0, end).join('/');
}

function commonDirectoryPrefix(paths: string[]): string {
  const split = paths
    .map((path) => path.split('/').filter(Boolean).slice(0, -1))
    .filter((parts) => parts.length > 0);
  if (split.length === 0) {
    return '';
  }
  const prefix: string[] = [];
  for (let index = 0; index < split[0].length; index += 1) {
    const part = split[0][index];
    if (!split.every((candidate) => candidate[index] === part)) {
      break;
    }
    prefix.push(part);
  }
  return prefix.length > 0 ? '/' + prefix.join('/') : '';
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
        <span>{job.currentLabel || formatJobPhase(job.phase)}</span>
        <span>{job.itemsImported ? `${job.itemsImported} imported` : job.rollupsImported ? `${job.rollupsImported} activity rows` : formatElapsed(job.startedAt, job.completedAt)}</span>
        {job.unmappedItems > 0 && <span>{job.unmappedItems} unmapped</span>}
      </div>
      {visibleEvents.length > 0 && (
        <div className="event-feed">
          {visibleEvents.map((event) => (
            <div key={event.id} className="event-row">
              <span>{formatJobPhase(event.phase)}</span>
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
  appearance,
  onAppearanceSave,
  onBackup,
  onSupportBundle,
  onRestore,
  backups,
  supportBundles,
  notice,
  busy,
}: {
  appearance: AppearanceSettings;
  onAppearanceSave: (appearance: AppearanceSettings) => void;
  onBackup: () => void;
  onSupportBundle: () => void;
  onRestore: (name: string, dryRun: boolean) => void;
  backups: Backup[];
  supportBundles: SupportBundle[];
  notice: string | null;
  busy: boolean;
}) {
  const [selectedBackup, setSelectedBackup] = useState('');
  const [customCss, setCustomCss] = useState(appearance.customCss);
  useEffect(() => {
    if (backups.length === 0) {
      setSelectedBackup('');
      return;
    }
    if (!backups.some((backup) => backup.name === selectedBackup)) {
      setSelectedBackup(backups[0].name);
    }
  }, [backups, selectedBackup]);
  useEffect(() => {
    setCustomCss(appearance.customCss);
  }, [appearance]);
  const activeBackup = backups.find((backup) => backup.name === selectedBackup) ?? backups[0];
  return (
    <section className="settings-layout">
      {notice && <div className="notice success settings-notice">{notice}</div>}
      <div className="panel form-panel appearance-panel">
        <div className="panel-heading">
          <h2>Custom CSS</h2>
          <span>Advanced</span>
        </div>
        <label>
          Custom CSS
          <textarea
            value={customCss}
            onChange={(event) => setCustomCss(event.target.value)}
            maxLength={20000}
            spellCheck={false}
            rows={8}
          />
        </label>
        <div className="button-row">
          <button
            className="primary-button"
            type="button"
            onClick={() => onAppearanceSave({ ...appearance, customCss })}
            disabled={busy}
          >
            <Save size={18} />
            Save CSS
          </button>
          <button
            className="secondary-button"
            type="button"
            onClick={() => {
              setCustomCss('');
              onAppearanceSave({ ...appearance, customCss: '' });
            }}
            disabled={busy}
          >
            <RotateCcw size={16} />
            Clear CSS
          </button>
        </div>
      </div>
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
          <select value={activeBackup?.name ?? ''} onChange={(event) => setSelectedBackup(event.target.value)} disabled={backups.length === 0}>
            {backups.length === 0 && <option value="">No backups available</option>}
            {backups.map((backup) => <option value={backup.name} key={backup.name}>{backup.name}</option>)}
          </select>
        </label>
        <div className="button-row">
          <button className="secondary-button" type="button" onClick={() => activeBackup && onRestore(activeBackup.name, true)} disabled={busy || !activeBackup}>
            <SearchCheck size={16} />
            Inspect
          </button>
          <button className="secondary-button" type="button" onClick={() => activeBackup && onRestore(activeBackup.name, false)} disabled={busy || !activeBackup}>
            <RotateCcw size={16} />
            Restore
          </button>
          {activeBackup && (
            <a className="secondary-button" href={api.backupDownloadUrl(activeBackup.name)} download>
              <Download size={16} />
              Download
            </a>
          )}
        </div>
        <div className="compact-list">
          {backups.length === 0 && <p className="muted-copy">No backups have been created yet.</p>}
          {backups.slice(0, 5).map((backup) => (
            <div className="compact-row bundle-row" key={backup.name}>
              <div>
                <strong>{backup.name}</strong>
                <span>{formatBytes(backup.sizeBytes)} • {formatDateTime(backup.createdAt)}</span>
              </div>
              <a className="secondary-button compact-action" href={api.backupDownloadUrl(backup.name)} download>
                <Download size={16} />
                Download
              </a>
            </div>
          ))}
        </div>
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

function ThemeToggle({ resolvedTheme, busy, onToggle }: { resolvedTheme: 'dark' | 'light'; busy: boolean; onToggle: () => void }) {
  const isLight = resolvedTheme === 'light';
  return (
    <button
      className={isLight ? 'theme-toggle light' : 'theme-toggle dark'}
      type="button"
      onClick={onToggle}
      disabled={busy}
      aria-label={`Switch to ${isLight ? 'dark' : 'light'} theme`}
      title={`Switch to ${isLight ? 'dark' : 'light'} theme`}
    >
      <Moon size={15} />
      <Sun size={15} />
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

const campaignRuleFields: Array<{ value: CampaignRuleField; label: string }> = [
  { value: 'kind', label: 'Kind' },
  { value: 'libraryName', label: 'Library' },
  { value: 'verification', label: 'Verification' },
  { value: 'estimatedSavingsBytes', label: 'Estimated bytes' },
  { value: 'verifiedSavingsBytes', label: 'Verified bytes' },
  { value: 'lastPlayedDays', label: 'Last played days' },
  { value: 'addedDays', label: 'Added days' },
  { value: 'playCount', label: 'Play count' },
  { value: 'uniqueUsers', label: 'Watched users' },
  { value: 'favoriteCount', label: 'Favorites' },
  { value: 'confidence', label: 'Confidence' },
];

const campaignRuleOperators: Array<{ value: CampaignRuleOperator; label: string }> = [
  { value: 'equals', label: 'Equals' },
  { value: 'not_equals', label: 'Not equals' },
  { value: 'in', label: 'In list' },
  { value: 'not_in', label: 'Not in list' },
  { value: 'greater_than', label: 'Greater than' },
  { value: 'greater_or_equal', label: 'At least' },
  { value: 'less_than', label: 'Less than' },
  { value: 'less_or_equal', label: 'At most' },
  { value: 'is_empty', label: 'Is empty' },
  { value: 'is_not_empty', label: 'Is not empty' },
];

function newCampaignDraft(): Campaign {
  const id = `campaign_${Date.now().toString(36)}`;
  return {
    id,
    name: 'Cold media review',
    description: 'Activity-backed review candidates with local storage proof.',
    enabled: true,
    targetKinds: ['movie'],
    targetLibraryNames: [],
    requireAllRules: true,
    minimumConfidence: 0.72,
    minimumStorageBytes: 10_000_000_000,
    rules: [
      { field: 'lastPlayedDays', operator: 'greater_or_equal', value: '365' },
      { field: 'estimatedSavingsBytes', operator: 'greater_or_equal', value: '10000000000' },
      { field: 'favoriteCount', operator: 'equals', value: '0' },
    ],
  };
}

function defaultCampaignRule(): CampaignRule {
  return { field: 'lastPlayedDays', operator: 'greater_or_equal', value: '365' };
}

function cloneCampaign(campaign: Campaign): Campaign {
  return {
    ...campaign,
    targetKinds: [...(campaign.targetKinds || [])],
    targetLibraryNames: [...(campaign.targetLibraryNames || [])],
    rules: (campaign.rules || []).map((rule) => ({
      ...rule,
      values: rule.values ? [...rule.values] : [],
    })),
  };
}

function normalizeCampaignDraft(campaign: Campaign): Campaign {
  const name = campaign.name.trim();
  return {
    ...campaign,
    id: slugCampaignID(campaign.id || name),
    name,
    description: campaign.description?.trim() || '',
    enabled: Boolean(campaign.enabled),
    targetKinds: (campaign.targetKinds || []).map((kind) => kind.trim()).filter(Boolean),
    targetLibraryNames: (campaign.targetLibraryNames || []).map((library) => library.trim()).filter(Boolean),
    requireAllRules: Boolean(campaign.requireAllRules),
    minimumConfidence: Math.min(1, Math.max(0, campaign.minimumConfidence || 0)),
    minimumStorageBytes: Math.max(0, campaign.minimumStorageBytes || 0),
    rules: (campaign.rules || []).map((rule) => ({
      field: rule.field,
      operator: rule.operator,
      value: rule.operator === 'in' || rule.operator === 'not_in' ? '' : (rule.value || '').trim(),
      values: rule.operator === 'in' || rule.operator === 'not_in' ? (rule.values || []).map((value) => value.trim()).filter(Boolean) : [],
    })),
  };
}

function emptyCampaignResult(id: string): CampaignResult {
  return {
    campaignId: id,
    enabled: true,
    matched: 0,
    suppressed: 0,
    totalEstimatedSavingsBytes: 0,
    totalVerifiedSavingsBytes: 0,
    confidenceMin: 0,
    confidenceAverage: 0,
    confidenceMax: 0,
    items: [],
  };
}

function slugCampaignID(value: string): string {
  const slug = value
    .toLowerCase()
    .trim()
    .replace(/[^a-z0-9]+/g, '_')
    .replace(/^_+|_+$/g, '');
  if (!slug) {
    return `campaign_${Date.now().toString(36)}`;
  }
  return slug.startsWith('campaign_') ? slug : `campaign_${slug}`;
}

function clampPercentInput(value: string): number {
  const parsed = Number.parseInt(value, 10);
  if (!Number.isFinite(parsed)) {
    return 0;
  }
  return Math.min(100, Math.max(0, parsed));
}

function formatCampaignRule(rule: CampaignRule): string {
  const field = campaignRuleFields.find((entry) => entry.value === rule.field)?.label || rule.field;
  const operator = campaignRuleOperators.find((entry) => entry.value === rule.operator)?.label || rule.operator;
  const value = rule.operator === 'in' || rule.operator === 'not_in' ? (rule.values || []).join(', ') : rule.value;
  if (rule.operator === 'is_empty' || rule.operator === 'is_not_empty') {
    return `${field} ${operator}`;
  }
  return `${field} ${operator} ${value || ''}`.trim();
}

function formatDateTime(value: string): string {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return 'Unknown';
  }
  return date.toLocaleDateString(undefined, { year: 'numeric', month: 'short', day: 'numeric' });
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

function formatRecommendationAction(value: Recommendation['action']): string {
  switch (value) {
    case 'review_abandoned_series':
      return 'Abandoned series';
    case 'review_inactive_series':
      return 'Inactive series';
    case 'review_inactive_movie':
      return 'Inactive movie';
    case 'review_never_watched_movie':
      return 'Never-watched movie';
    case 'review_duplicate':
      return 'Duplicate candidate';
    case 'review_unwatched_duplicate':
      return 'Unwatched duplicate';
    case 'review_oversized':
      return 'Oversized file';
    case 'review_missing_subtitles':
      return 'Missing subtitles';
    case 'review_campaign_match':
      return 'Campaign match';
    default:
      return 'Review item';
  }
}

function confidenceTone(value: number): string {
  if (value >= 0.85) {
    return 'strong';
  }
  if (value >= 0.72) {
    return 'moderate';
  }
  return 'caution';
}

function recommendationItemCount(rec: Recommendation): number {
  return parseEvidenceNumber(rec.evidence?.itemCount, rec.affectedPaths.length || 1);
}

function formatAffectedSummary(itemCount: number, groupCount: number, subjectKind?: string): string {
  const itemLabel = formatFileCount(itemCount);
  if (groupCount <= 1) {
    return itemLabel;
  }
  return `${itemLabel} across ${formatGroupCount(groupCount, subjectKind)}`;
}

function formatFileCount(count: number): string {
  return `${count} ${count === 1 ? 'file' : 'files'}`;
}

function formatGroupCount(count: number, subjectKind?: string): string {
  const groupLabel = subjectKind === 'movie' ? 'group' : 'season/group';
  return `${count} ${count === 1 ? groupLabel : `${groupLabel}s`}`;
}

function titleFor(view: View): string {
  return {
    dashboard: 'Dashboard',
    libraries: 'Libraries',
    catalog: 'Catalog',
    recommendations: 'Review Queue',
    campaigns: 'Campaigns',
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

function formatJobPhase(phase?: string): string {
  switch (phase) {
    case 'users':
      return 'Reading profiles';
    case 'inventory':
      return 'Importing media';
    case 'activity':
      return 'Importing watch activity';
    case 'recommendations':
      return 'Building review queue';
    case 'connecting':
      return 'Connecting';
    case 'complete':
      return 'Complete';
    case 'canceled':
      return 'Canceled';
    case 'failed':
      return 'Failed';
    case 'queued':
      return 'Queued';
    default:
      return phase ? phase.replaceAll('_', ' ') : 'Preparing';
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
