import { useQuery } from '@tanstack/react-query';
import { useAuth } from '../../context/AuthContext';
import LogFeed from '../../components/LogFeed';
import styles from './OverviewPage.module.css';

export default function OverviewPage() {
  const { client } = useAuth();

  const { data: stations = [] } = useQuery({
    queryKey: ['stations'],
    queryFn: () => client!.listStations(),
    refetchInterval: 3000,
    enabled: !!client,
  });

  const totalStations = stations.length;
  const liveNow = stations.filter((s) => s.isLive).length;
  const activeRelays = stations.filter((s) => s.isRelaying).length;

  return (
    <div className={styles.page}>
      <div className={styles.header}>
        <h1 className={styles.heading}>Overview</h1>
        <p className={styles.subtitle}>Server status and activity</p>
      </div>

      <div className={styles.statsRow}>
        <div className={styles.statCard}>
          <span className={styles.statNumber}>{totalStations}</span>
          <span className={styles.statLabel}>Total Stations</span>
        </div>
        <div className={styles.statCard}>
          <span className={styles.statNumber}>{liveNow}</span>
          <span className={styles.statLabel}>Live Now</span>
        </div>
        <div className={styles.statCard}>
          <span className={styles.statNumber}>{activeRelays}</span>
          <span className={styles.statLabel}>Active Relays</span>
        </div>
      </div>

      <div className={styles.logsSection}>
        <h2 className={styles.sectionHeading}>Server Logs</h2>
        <LogFeed />
      </div>
    </div>
  );
}
