import styles from './FederationPage.module.css';

export default function FederationPage() {
  return (
    <div className={styles.page}>
      <div className={styles.header}>
        <h1 className={styles.heading}>Federation</h1>
        <p className={styles.subtitle}>Manage ActivityPub connections</p>
      </div>

      <div className={styles.emptyCard}>
        <span className={styles.emptyIcon}>
          <svg width="40" height="40" viewBox="0 0 24 24" fill="none" stroke="#6366f1" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round">
            <circle cx="12" cy="12" r="10" />
            <line x1="2" y1="12" x2="22" y2="12" />
            <path d="M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10 15.3 15.3 0 0 1 4-10z" />
          </svg>
        </span>
        <h2 className={styles.emptyHeading}>No federation connections</h2>
        <p className={styles.emptyText}>Federation management coming soon.</p>
      </div>
    </div>
  );
}
