import { useEffect, useState, useCallback } from 'react';
import { useNavigate } from 'react-router-dom';
import { useAuth } from '../context/AuthContext';
import { type StationStatus } from '../api/client';
import StationCard from '../components/StationCard';
import LogFeed from '../components/LogFeed';

export default function Dashboard() {
  const { client, logout } = useAuth();
  const navigate = useNavigate();
  const [stations, setStations] = useState<StationStatus[]>([]);
  const [error, setError] = useState('');

  const refresh = useCallback(async () => {
    if (!client) { navigate('/admin/ui/login'); return; }
    try {
      const data = await client.listStations();
      setStations(data);
      setError('');
    } catch {
      setError('Failed to fetch stations. Check your admin key.');
    }
  }, [client, navigate]);

  useEffect(() => {
    refresh();
    const id = setInterval(refresh, 3000);
    return () => clearInterval(id);
  }, [refresh]);

  if (!client) return null;

  return (
    <div style={{ maxWidth: '900px', margin: '0 auto', padding: '1.5rem', background: '#0f0f0f', minHeight: '100vh' }}>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '1.5rem' }}>
        <h1 style={{ color: '#fff', margin: 0 }}>OpenWaves Admin</h1>
        <button onClick={() => { logout(); navigate('/admin/ui/login'); }} style={{ background: 'transparent', color: '#888', border: '1px solid #444', borderRadius: '4px', padding: '0.35rem 0.75rem', cursor: 'pointer' }}>
          Logout
        </button>
      </div>
      {error && <div style={{ color: '#f87171', marginBottom: '1rem' }}>{error}</div>}
      {stations.length === 0 && !error && <div style={{ color: '#555' }}>No stations configured.</div>}
      {stations.map(s => (
        <StationCard
          key={s.username}
          station={s}
          client={client}
          baseURL=""
          onRefresh={refresh}
        />
      ))}
      <LogFeed />
    </div>
  );
}
