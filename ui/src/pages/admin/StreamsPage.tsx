import { useState } from 'react';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import { useAuth } from '../../context/AuthContext';
import StreamCard from '../../components/admin/StreamCard';
import StartStreamModal from '../../components/admin/StartStreamModal';
import styles from './StreamsPage.module.css';

function PlusIcon() {
  return (
    <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round">
      <line x1="12" y1="5" x2="12" y2="19" />
      <line x1="5" y1="12" x2="19" y2="12" />
    </svg>
  );
}

export default function StreamsPage() {
  const { client } = useAuth();
  const queryClient = useQueryClient();
  const [showModal, setShowModal] = useState(false);

  const { data: stations = [], error } = useQuery({
    queryKey: ['stations'],
    queryFn: () => client!.listStations(),
    refetchInterval: 3000,
    enabled: !!client,
  });

  const sorted = [...stations].sort((a, b) => {
    if (a.isLive === b.isLive) return 0;
    return a.isLive ? -1 : 1;
  });

  function handleMutate() {
    queryClient.invalidateQueries({ queryKey: ['stations'] });
  }

  if (!client) return null;

  return (
    <div className={styles.page}>
      <div className={styles.header}>
        <div className={styles.headingGroup}>
          <h1 className={styles.heading}>Active Streams</h1>
          <p className={styles.subtitle}>Manage live broadcasts</p>
        </div>
        <button className={styles.btnNewStream} onClick={() => setShowModal(true)}>
          <PlusIcon />
          Start New Stream
        </button>
      </div>

      {error && (
        <p className={styles.errorMsg}>
          {error instanceof Error ? error.message : 'Failed to load stations.'}
        </p>
      )}

      {sorted.length === 0 && !error ? (
        <p className={styles.emptyState}>No stations configured.</p>
      ) : (
        <div className={styles.stationList}>
          {sorted.map(station => (
            <StreamCard
              key={station.username}
              station={station}
              client={client}
              onMutate={handleMutate}
            />
          ))}
        </div>
      )}

      {showModal && (
        <StartStreamModal
          stations={stations}
          client={client}
          onClose={() => setShowModal(false)}
          onSuccess={handleMutate}
        />
      )}
    </div>
  );
}
