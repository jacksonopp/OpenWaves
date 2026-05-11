import { useState, useEffect } from 'react';
import { useMutation } from '@tanstack/react-query';
import { type StationStatus, type AdminClient, type AudioInputType } from '../../api/client';
import HLSPlayer from '../HLSPlayer';
import styles from './StreamCard.module.css';

interface Props {
  station: StationStatus;
  client: AdminClient;
  onMutate: () => void;
  onDelete?: () => void;
}

function TrashIcon() {
  return (
    <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
      <polyline points="3 6 5 6 21 6" />
      <path d="M19 6l-1 14H6L5 6" />
      <path d="M10 11v6M14 11v6" />
      <path d="M9 6V4h6v2" />
    </svg>
  );
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

export default function StreamCard({ station, client, onMutate, onDelete }: Props) {
  const [showMonitor, setShowMonitor] = useState(false);
  const [showSettings, setShowSettings] = useState(false);
  const [relayURL, setRelayURL] = useState('');
  const [audioFile, setAudioFile] = useState(station.audioInput?.file ?? '');
  const [audioInputType, setAudioInputType] = useState<AudioInputType>(station.audioInput?.type ?? 'silence');
  const [settingsMsg, setSettingsMsg] = useState<{ type: 'error' | 'success'; text: string } | null>(null);

  useEffect(() => {
    if (!showSettings) return;
    setAudioInputType(station.audioInput?.type ?? 'silence');
    setAudioFile(station.audioInput?.file ?? '');
  }, [station.audioInput]);

  const stopMutation = useMutation({
    mutationFn: async () => {
      if (station.isIngesting) await client.stopIngest(station.username).catch(() => {});
      await client.stopStream(station.username);
    },
    onSuccess: onMutate,
  });

  const startStreamMutation = useMutation({
    mutationFn: () => client.startStream(station.username),
    onSuccess: () => { setSettingsMsg({ type: 'success', text: 'Stream ready.' }); onMutate(); },
    onError: (e: Error) => setSettingsMsg({ type: 'error', text: e.message }),
  });

  const startRelayMutation = useMutation({    mutationFn: () => client.startRelay(station.username, relayURL),
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

  const setAudioInputMutation = useMutation({
    mutationFn: () => client.setAudioInput(station.username, {
      type: audioInputType,
      file: audioInputType === 'file' ? audioFile : undefined,
    }),
    onSuccess: () => { setSettingsMsg({ type: 'success', text: 'Audio input updated.' }); onMutate(); },
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
          <span className={styles.audioInputLabel}>
            {station.audioInput?.type === 'file' ? `File: ${station.audioInput.file}` : station.audioInput?.type ?? 'silence'}
          </span>
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

          {!station.isLive && !station.isIngesting && (
            <button
              className={styles.btnPrimary}
              onClick={() => startStreamMutation.mutate()}
              disabled={startStreamMutation.isPending}
            >
              Start Stream
            </button>
          )}

          {(station.isLive || station.isIngesting) && (
            <button
              className={styles.btnIconDanger}
              onClick={() => stopMutation.mutate()}
              disabled={stopMutation.isPending}
              title="Stop stream"
            >
              <StopIcon />
            </button>
          )}

          {!station.isStatic && onDelete && (
            <button
              className={styles.btnIconDanger}
              onClick={onDelete}
              title="Delete channel"
            >
              <TrashIcon />
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
          {/* Stream */}
          <div className={styles.settingsGroup}>
            <p className={styles.settingsGroupTitle}>Stream</p>
            {station.isLive ? (
              <button
                className={styles.btnDanger}
                onClick={() => stopMutation.mutate()}
                disabled={stopMutation.isPending}
              >
                Stop Stream
              </button>
            ) : (
              <button
                className={styles.btnPrimary}
                onClick={() => startStreamMutation.mutate()}
                disabled={startStreamMutation.isPending}
              >
                Start Stream
              </button>
            )}
          </div>

          {/* Audio Input */}
          <div className={styles.settingsGroup}>
            <p className={styles.settingsGroupTitle}>Audio Input</p>
            <div className={styles.inputRow}>
              <select
                className={styles.input}
                value={audioInputType}
                onChange={e => setAudioInputType(e.target.value as AudioInputType)}
              >
                <option value="silence">Silence</option>
                <option value="test_tone">Test Tone (440 Hz)</option>
                <option value="file">Audio File</option>
              </select>
              {audioInputType === 'file' && (
                <input
                  className={styles.input}
                  type="text"
                  placeholder="/path/to/audio.mp3"
                  value={audioFile}
                  onChange={e => setAudioFile(e.target.value)}
                />
              )}
              <button
                className={styles.btnPrimary}
                onClick={() => setAudioInputMutation.mutate()}
                disabled={setAudioInputMutation.isPending}
              >
                Apply
              </button>
            </div>
          </div>

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
              <button
                className={styles.btnPrimary}
                onClick={() => startIngestMutation.mutate()}
                disabled={startIngestMutation.isPending}
              >
                Start Ingest
              </button>
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
