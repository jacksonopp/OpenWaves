import { useState } from 'react';
import { useMutation } from '@tanstack/react-query';
import { type AdminClient } from '../../api/client';
import styles from './CreateChannelModal.module.css';

interface Props {
  client: AdminClient;
  onClose: () => void;
  onSuccess: () => void;
}

export default function CreateChannelModal({ client, onClose, onSuccess }: Props) {
  const [username, setUsername] = useState('');
  const [name, setName] = useState('');
  const [summary, setSummary] = useState('');
  const [relayPolicy, setRelayPolicy] = useState('open');
  const [territory, setTerritory] = useState('');
  const [usernameError, setUsernameError] = useState('');

  const mutation = useMutation({
    mutationFn: async () => {
      const territories = territory.trim()
        ? territory.split(',').map(t => t.trim()).filter(Boolean)
        : undefined;
      await client.createChannel({
        username,
        name: name || undefined,
        summary: summary || undefined,
        relay_policy: relayPolicy || undefined,
        license_territory: territories,
      });
    },
    onSuccess: () => {
      onSuccess();
      onClose();
    },
  });

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (!/^[a-z0-9-]+$/.test(username)) {
      setUsernameError('Username must contain only lowercase letters, digits, and hyphens.');
      return;
    }
    setUsernameError('');
    mutation.mutate();
  }

  return (
    <div className={styles.overlay} onClick={e => { if (e.target === e.currentTarget) onClose(); }}>
      <div className={styles.modal} role="dialog" aria-modal="true" aria-labelledby="modal-title">
        <h2 id="modal-title" className={styles.modalTitle}>Create New Channel</h2>

        <form className={styles.form} onSubmit={handleSubmit}>
          <div className={styles.field}>
            <label className={styles.label} htmlFor="channel-username">Username <span className={styles.required}>*</span></label>
            <input
              id="channel-username"
              className={styles.input}
              type="text"
              placeholder="my-station"
              value={username}
              onChange={e => { setUsername(e.target.value); setUsernameError(''); }}
              required
            />
            {usernameError && <p className={styles.fieldError}>{usernameError}</p>}
          </div>

          <div className={styles.field}>
            <label className={styles.label} htmlFor="channel-name">Display Name</label>
            <input
              id="channel-name"
              className={styles.input}
              type="text"
              placeholder="My Station"
              value={name}
              onChange={e => setName(e.target.value)}
            />
          </div>

          <div className={styles.field}>
            <label className={styles.label} htmlFor="channel-summary">Summary</label>
            <input
              id="channel-summary"
              className={styles.input}
              type="text"
              placeholder="A short description of this channel"
              value={summary}
              onChange={e => setSummary(e.target.value)}
            />
          </div>

          <div className={styles.field}>
            <label className={styles.label} htmlFor="channel-relay-policy">Relay Policy</label>
            <select
              id="channel-relay-policy"
              className={styles.select}
              value={relayPolicy}
              onChange={e => setRelayPolicy(e.target.value)}
            >
              <option value="open">Open</option>
              <option value="closed">Closed</option>
              <option value="allowlist">Allowlist</option>
            </select>
          </div>

          <div className={styles.field}>
            <label className={styles.label} htmlFor="channel-territory">Territory</label>
            <input
              id="channel-territory"
              className={styles.input}
              type="text"
              placeholder="US, GB, * for worldwide"
              value={territory}
              onChange={e => setTerritory(e.target.value)}
            />
          </div>

          {mutation.isError && (
            <p className={styles.errorMsg}>{(mutation.error as Error).message}</p>
          )}

          <div className={styles.actions}>
            <button type="button" className={styles.btnCancel} onClick={onClose}>
              Cancel
            </button>
            <button
              type="submit"
              className={styles.btnSubmit}
              disabled={mutation.isPending || !username}
            >
              {mutation.isPending ? 'Creating…' : 'Create Channel'}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}
