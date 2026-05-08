import { useState } from 'react';
import { useMutation } from '@tanstack/react-query';
import { type StationStatus, type AdminClient } from '../../api/client';
import styles from './StartStreamModal.module.css';

interface Props {
  stations: StationStatus[];
  client: AdminClient;
  onClose: () => void;
  onSuccess: () => void;
}

export default function StartStreamModal({ stations, client, onClose, onSuccess }: Props) {
  const [selectedUsername, setSelectedUsername] = useState(stations[0]?.username ?? '');
  const [audioFile, setAudioFile] = useState('');

  const mutation = useMutation({
    mutationFn: () => client.startIngest(selectedUsername, audioFile || undefined),
    onSuccess: () => {
      onSuccess();
      onClose();
    },
  });

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (!selectedUsername) return;
    mutation.mutate();
  }

  return (
    <div className={styles.overlay} onClick={e => { if (e.target === e.currentTarget) onClose(); }}>
      <div className={styles.modal} role="dialog" aria-modal="true" aria-labelledby="modal-title">
        <h2 id="modal-title" className={styles.modalTitle}>Start New Stream</h2>

        <form className={styles.form} onSubmit={handleSubmit}>
          <div className={styles.field}>
            <label className={styles.label} htmlFor="station-select">Station</label>
            <select
              id="station-select"
              className={styles.select}
              value={selectedUsername}
              onChange={e => setSelectedUsername(e.target.value)}
            >
              {stations.map(s => (
                <option key={s.username} value={s.username}>{s.username}</option>
              ))}
            </select>
          </div>

          <div className={styles.field}>
            <label className={styles.label} htmlFor="audio-file">Audio file path</label>
            <input
              id="audio-file"
              className={styles.input}
              type="text"
              placeholder="/path/to/audio.mp3 (optional, leave blank for test tone)"
              value={audioFile}
              onChange={e => setAudioFile(e.target.value)}
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
              disabled={mutation.isPending || !selectedUsername}
            >
              {mutation.isPending ? 'Starting…' : 'Start Ingest'}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}
