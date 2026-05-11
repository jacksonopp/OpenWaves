import { useState } from 'react';
import { useQuery, useQueryClient, useMutation } from '@tanstack/react-query';
import { useAuth } from '../../context/AuthContext';
import StreamCard from '../../components/admin/StreamCard';
import CreateChannelModal from '../../components/admin/CreateChannelModal';
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

  const deleteChannelMutation = useMutation({
    mutationFn: (username: string) => client!.deleteChannel(username),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['stations'] }),
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
          Create New Channel
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
          {sorted.map(s => (
            <StreamCard
              key={s.username}
              station={s}
              client={client}
              onMutate={handleMutate}
              onDelete={!s.isStatic ? () => {
                if (window.confirm(`Delete channel "${s.username}"? This cannot be undone.`)) {
                  deleteChannelMutation.mutate(s.username);
                }
              } : undefined}
            />
          ))}
        </div>
      )}

      {showModal && (
        <CreateChannelModal
          client={client}
          onClose={() => setShowModal(false)}
          onSuccess={() => { queryClient.invalidateQueries({ queryKey: ['stations'] }); setShowModal(false); }}
        />
      )}
    </div>
  );
}
