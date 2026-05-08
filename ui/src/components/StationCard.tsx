import { useState, type CSSProperties } from 'react';
import { type StationStatus, type AdminClient } from '../api/client';
import HLSPlayer from './HLSPlayer';

interface Props {
  station: StationStatus;
  client: AdminClient;
  baseURL: string;
  onRefresh: () => void;
}

export default function StationCard({ station, client, baseURL, onRefresh }: Props) {
  const [relayInput, setRelayInput] = useState('');
  const [showRelayModal, setShowRelayModal] = useState(false);
  const [audioFileInput, setAudioFileInput] = useState('');
  const [showIngestModal, setShowIngestModal] = useState(false);
  const [busy, setBusy] = useState(false);
  const [msg, setMsg] = useState('');

  async function act(fn: () => Promise<void>, successMsg: string) {
    setBusy(true);
    setMsg('');
    try {
      await fn();
      setMsg(successMsg);
      onRefresh();
    } catch (e: unknown) {
      setMsg(`Error: ${e instanceof Error ? e.message : String(e)}`);
    } finally {
      setBusy(false);
    }
  }

  const hlsURL = `${baseURL}/stations/${station.username}/hls/stream.m3u8`;

  return (
    <div style={{ background: '#1a1a1a', border: '1px solid #333', borderRadius: '8px', padding: '1rem', marginBottom: '1rem' }}>
      <div style={{ display: 'flex', alignItems: 'center', gap: '0.75rem', marginBottom: '0.5rem' }}>
        <h3 style={{ color: '#fff', margin: 0 }}>{station.username}</h3>
        <span style={{ background: station.isLive ? '#16a34a' : '#374151', color: '#fff', padding: '2px 8px', borderRadius: '12px', fontSize: '0.75rem' }}>
          {station.isLive ? '● LIVE' : 'OFFLINE'}
        </span>
        {station.isRelaying && (
          <span style={{ background: '#1d4ed8', color: '#fff', padding: '2px 8px', borderRadius: '12px', fontSize: '0.75rem' }}>↔ RELAYING</span>
        )}
        {station.isIngesting && (
          <span style={{ background: '#b45309', color: '#fff', padding: '2px 8px', borderRadius: '12px', fontSize: '0.75rem' }}>⏺ INGESTING</span>
        )}
      </div>
      <div style={{ color: '#888', fontSize: '0.8rem', marginBottom: '0.75rem' }}>
        Listeners: {station.listenerCount} · Segments: {station.segmentCount}
      </div>

      {station.isLive && <HLSPlayer src={hlsURL} />}

      <div style={{ display: 'flex', flexWrap: 'wrap', gap: '0.5rem', marginTop: '0.75rem' }}>
        <button onClick={() => act(() => client.startStream(station.username), 'Stream started')} disabled={busy} style={btnStyle('#16a34a')}>Start Stream</button>
        <button onClick={() => act(() => client.stopStream(station.username), 'Stream stopped')} disabled={busy} style={btnStyle('#b91c1c')}>Stop Stream</button>
        {!station.isRelaying
          ? <button onClick={() => setShowRelayModal(true)} disabled={busy} style={btnStyle('#1d4ed8')}>Start Relay…</button>
          : <button onClick={() => act(() => client.stopRelay(station.username), 'Relay stopped')} disabled={busy} style={btnStyle('#6b7280')}>Stop Relay</button>
        }
        {!station.isIngesting
          ? <button onClick={() => setShowIngestModal(true)} disabled={busy} style={btnStyle('#b45309')}>Start Ingest…</button>
          : <button onClick={() => act(() => client.stopIngest(station.username), 'Ingest stopped')} disabled={busy} style={btnStyle('#6b7280')}>Stop Ingest</button>
        }
      </div>

      {showRelayModal && (
        <div style={{ marginTop: '0.75rem', display: 'flex', gap: '0.5rem' }}>
          <input
            value={relayInput}
            onChange={e => setRelayInput(e.target.value)}
            placeholder="Source URL (e.g. http://other-server/stations/morning-vibes)"
            style={{ flex: 1, padding: '0.4rem', background: '#2a2a2a', border: '1px solid #555', color: '#fff', borderRadius: '4px' }}
          />
          <button
            onClick={() => act(async () => {
              await client.startRelay(station.username, relayInput);
              setShowRelayModal(false);
              setRelayInput('');
            }, 'Relay started')}
            disabled={busy || !relayInput}
            style={btnStyle('#1d4ed8')}
          >Go</button>
          <button onClick={() => setShowRelayModal(false)} style={btnStyle('#374151')}>Cancel</button>
        </div>
      )}

      {showIngestModal && (
        <div style={{ marginTop: '0.75rem' }}>
          <div style={{ color: '#aaa', fontSize: '0.75rem', marginBottom: '0.4rem' }}>
            Audio file path (leave blank for test tone)
          </div>
          <div style={{ display: 'flex', gap: '0.5rem' }}>
            <input
              value={audioFileInput}
              onChange={e => setAudioFileInput(e.target.value)}
              placeholder="/path/to/audio.mp3  (optional)"
              style={{ flex: 1, padding: '0.4rem', background: '#2a2a2a', border: '1px solid #555', color: '#fff', borderRadius: '4px' }}
            />
            <button
              onClick={() => act(async () => {
                await client.startIngest(station.username, audioFileInput || undefined);
                setShowIngestModal(false);
                setAudioFileInput('');
              }, 'Ingest started')}
              disabled={busy}
              style={btnStyle('#b45309')}
            >Go</button>
            <button onClick={() => setShowIngestModal(false)} style={btnStyle('#374151')}>Cancel</button>
          </div>
        </div>
      )}

      {msg && <div style={{ marginTop: '0.5rem', color: msg.startsWith('Error') ? '#f87171' : '#4ade80', fontSize: '0.8rem' }}>{msg}</div>}
    </div>
  );
}

function btnStyle(bg: string): CSSProperties {
  return { background: bg, color: '#fff', border: 'none', borderRadius: '4px', padding: '0.4rem 0.75rem', cursor: 'pointer', fontSize: '0.8rem' };
}
