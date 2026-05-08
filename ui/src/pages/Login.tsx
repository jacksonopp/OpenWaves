import { useState } from 'react';
import { useNavigate } from '@tanstack/react-router';
import { useAuth } from '../context/AuthContext';
import { AdminClient } from '../api/client';
import styles from './Login.module.css';

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
      navigate({ to: '/admin/ui/streams' });
    } catch {
      setError('Invalid admin key or server unreachable.');
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className={styles.page}>
      <div className={styles.brand}>
        <div className={styles.brandIcon}>((·))</div>
        <h1 className={styles.brandName}>OpenWaves</h1>
        <p className={styles.brandSubtitle}>Admin Panel</p>
      </div>

      <div className={styles.card}>
        <div className={styles.cardHeader}>
          <h2 className={styles.cardTitle}>Sign in</h2>
          <p className={styles.cardSubtitle}>Enter your admin key to continue</p>
        </div>

        <form onSubmit={handleSubmit} className={styles.form}>
          <input
            type="password"
            value={key}
            onChange={e => setKey(e.target.value)}
            placeholder="Admin key"
            className={styles.input}
          />
          {error && <p className={styles.error}>{error}</p>}
          <button type="submit" disabled={loading || !key} className={styles.button}>
            {loading ? 'Checking…' : 'Sign in'}
          </button>
          <p className={styles.note}>
            Admin key auth will be replaced with a proper session system in a future version.
          </p>
        </form>
      </div>
    </div>
  );
}
