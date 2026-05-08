import { useState } from 'react';
import { useMutation } from '@tanstack/react-query';
import { type StationStatus, type AdminClient } from '../../api/client';
import HLSPlayer from '../HLSPlayer';
import styles from './StreamCard.module.css';

interface Props {
  station: StationStatus;
  client: AdminClient;
  onMutate: () => void;
}

function ListenerIcon() {
  return (
    <svg className={styles.listenerIcon} width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
      <path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2" />
      <circle cx="9" cy="7" r="4" />
      <path d="M23 21v-2a4 4 0 0 0-3-3.87" />
      <path d="M16 3.13a4 4 0 0 1 0 7.75" />
    </svg>
  );
}

function StopIcon() {
  return (
    <svg width="18" height="18" viewBox="0 0 24 24" fill="#ef4444">
      <rect x="4" y="4" width="16" height="16" rx="2" />
    </svg>
  );
}

function GearIcon() {
  return (
    <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
      <circle cx="12" cy="12" r="3" />
      <path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1-2.83 2.83l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-4 0v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83-2.83l.06-.06A1.65 1.65 0 0 0 4.68 15a1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1 0-4h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 2.83-2.83l.06.06A1.65 1.65 0 0 0 9 4.68a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 4 0v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 2.83l-.06.06A1.65 1.65 0 0 0 19.4 9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 0 4h-.09a1.65 1.65 0 0 0-1.51 1z" />
    </svg>
  );
}

function MonitorIcon() {
  return (
    <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
      <polygon points="5 3 19 12 5 21 5 3" />
    </svg>
  );
}

export default function StreamCard({ station, client, onMutate }: Props) {
  const [showMonitor, setShowMonitor] = useState(false);
  const [showSettings, setShowSettings] = useState(false);
  const [relayURL, setRelayURL] = useState('');
  const [audioFile, setAudioFile] = useState('');
  const [settingsMsg, setSettingsMsg] = useState<{ type: 'error' | 'success'; text: string } | null>(null);

  const stopMutation = useMutation({
    mutationFn: () => client.stopStream(station.username),
    onSuccess: onMutate,
  });

  const startRelayMutation = useMutation({
    mutationFn: () => client.startRelay(station.username, relayURL),
    onSuccess: () => { setSettingsMsg({ type: 'success', text: 'Relay started.' }); onMutate(); },
    onError: (e: Error) => setSettingsMsg({ type: 'error', text: e.message }),
  });

  const stopRelayMutation = useMutation({
    mutationFn: () => client.stopRelay(station.username),
    onSuccess: () => { setSettingsMsg({ type: 'success', text: 'Relay stopped.' }); onMutate(); },
    onError: (e: Error) => setSettingsMsg({ type: 'error', text: e.message }),
  });

  const startIngestMutation = useMutation({
    mutationFn: () => client.startIngest(station.username, audioFile || undefined),
    onSuccess: () => { setSettingsMsg({ type: 'success', text: 'Ingest started.' }); onMutate(); },
    onError: (e: Error) => setSettingsMsg({ type: 'error', text: e.message }),
  });

  const stopIngestMutation = useMutation({
    mutationFn: () => client.stopIngest(station.username),
    onSuccess: () => { setSettingsMsg({ type: 'success', text: 'Ingest stopped.' }); onMutate(); },
    onError: (e: Error) => setSettingsMsg({ type: 'error', text: e.message }),
  });

  const hlsSrc = `/stations/${station.username}/hls/stream.m3u8`;

  return (
    <div className={styles.card}>
      <div className={styles.cardMain}>
        {/* Left */}
        <div className={styles.cardLeft}>
          <div className={styles.titleRow}>
            <h3 className={styles.stationName}>{station.username}</h3>
            {station.isLive ? (
              <span className={styles.badgeLive}>LIVE</span>
            ) : (
              <span className={styles.badgeOffline}>OFFLINE</span>
            )}
          </div>
          <p className={styles.subtitle}>By {station.username}</p>
          <div className={styles.metaRow}>
            <span className={styles.listenerCount}>
              <ListenerIcon />
              {station.listenerCount}
            </span>
            {station.isRelaying && <span className={styles.badgeRelaying}>RELAYING</span>}
            {station.isIngesting && <span className={styles.badgeIngesting}>INGESTING</span>}
          </div>
        </div>

        {/* Right */}
        <div className={styles.cardRight}>
          <button
            className={`${styles.btnMonitor}${showMonitor ? ` ${styles.active}` : ''}`}
            onClick={() => setShowMonitor(v => !v)}
            title="Monitor stream"
          >
            <MonitorIcon />
            Monitor
          </button>

          {station.isLive && (
            <button
              className={styles.btnIconDanger}
              onClick={() => stopMutation.mutate()}
              disabled={stopMutation.isPending}
              title="Stop stream"
            >
              <StopIcon />
            </button>
          )}

          <button
            className={`${styles.btnIcon}${showSettings ? ` ${styles.active}` : ''}`}
            onClick={() => setShowSettings(v => !v)}
            title="Settings"
          >
            <GearIcon />
          </button>
        </div>
      </div>

      {/* Monitor section */}
      {showMonitor && (
        <div className={styles.sectionMonitor}>
          <HLSPlayer src={hlsSrc} />
        </div>
      )}

      {/* Settings section */}
      {showSettings && (
        <div className={styles.sectionSettings}>
          {/* Relay */}
          <div className={styles.settingsGroup}>
            <p className={styles.settingsGroupTitle}>Relay</p>
            {station.isRelaying ? (
              <button
                className={styles.btnDanger}
                onClick={() => stopRelayMutation.mutate()}
                disabled={stopRelayMutation.isPending}
              >
                Stop Relay
              </button>
            ) : (
              <div className={styles.inputRow}>
                <input
                  className={styles.input}
                  type="text"
                  placeholder="Source URL (e.g. https://source.example.com/stations/radio/hls/stream.m3u8)"
                  value={relayURL}
                  onChange={e => setRelayURL(e.target.value)}
                />
                <button
                  className={styles.btnPrimary}
                  onClick={() => startRelayMutation.mutate()}
                  disabled={startRelayMutation.isPending || !relayURL.trim()}
                >
                  Start Relay
                </button>
              </div>
            )}
          </div>

          {/* Ingest */}
          <div className={styles.settingsGroup}>
            <p className={styles.settingsGroupTitle}>Ingest</p>
            {station.isIngesting ? (
              <button
                className={styles.btnDanger}
                onClick={() => stopIngestMutation.mutate()}
                disabled={stopIngestMutation.isPending}
              >
                Stop Ingest
              </button>
            ) : (
              <div className={styles.inputRow}>
                <input
                  className={styles.input}
                  type="text"
                  placeholder="/path/to/audio.mp3 (optional, leave blank for test tone)"
                  value={audioFile}
                  onChange={e => setAudioFile(e.target.value)}
                />
                <button
                  className={styles.btnPrimary}
                  onClick={() => startIngestMutation.mutate()}
                  disabled={startIngestMutation.isPending}
                >
                  Start Ingest
                </button>
              </div>
            )}
          </div>

          {settingsMsg && (
            <p className={settingsMsg.type === 'error' ? styles.mutationError : styles.mutationSuccess}>
              {settingsMsg.text}
            </p>
          )}

          {stopMutation.isError && (
            <p className={styles.mutationError}>{(stopMutation.error as Error).message}</p>
          )}
        </div>
      )}
    </div>
  );
}
