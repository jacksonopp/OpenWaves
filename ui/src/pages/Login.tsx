import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { useAuth } from '../context/AuthContext';
import { AdminClient } from '../api/client';

export default function Login() {
  const { login } = useAuth();
  const navigate = useNavigate();
  const [key, setKey] = useState('');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    setLoading(true);
    setError('');
    try {
      const client = new AdminClient(key);
      await client.listStations();
      login(key);
      navigate('/admin/ui/');
    } catch {
      setError('Invalid admin key or server unreachable.');
    } finally {
      setLoading(false);
    }
  }

  return (
    <div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: '100vh', background: '#0f0f0f' }}>
      <form onSubmit={handleSubmit} style={{ background: '#1a1a1a', padding: '2rem', borderRadius: '8px', minWidth: '300px' }}>
        <h2 style={{ color: '#fff', marginBottom: '1.5rem' }}>OpenWaves Admin</h2>
        <input
          type="password"
          value={key}
          onChange={e => setKey(e.target.value)}
          placeholder="Admin key"
          style={{ width: '100%', padding: '0.5rem', marginBottom: '1rem', boxSizing: 'border-box', background: '#2a2a2a', border: '1px solid #444', color: '#fff', borderRadius: '4px' }}
        />
        {error && <p style={{ color: '#f87171', marginBottom: '0.5rem' }}>{error}</p>}
        <button type="submit" disabled={loading || !key} style={{ width: '100%', padding: '0.5rem', background: '#6d28d9', color: '#fff', border: 'none', borderRadius: '4px', cursor: 'pointer' }}>
          {loading ? 'Checking…' : 'Login'}
        </button>
        <p style={{ color: '#888', fontSize: '0.75rem', marginTop: '1rem' }}>
          Note: Admin key auth will be replaced with a proper session system in a future version.
        </p>
      </form>
    </div>
  );
}
