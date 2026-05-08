import styles from './ModerationPage.module.css';

export default function ModerationPage() {
  return (
    <div className={styles.page}>
      <div className={styles.header}>
        <h1 className={styles.heading}>Moderation</h1>
        <p className={styles.subtitle}>Manage content and users</p>
      </div>

      <div className={styles.emptyCard}>
        <span className={styles.emptyIcon}>
          <svg width="40" height="40" viewBox="0 0 24 24" fill="none" stroke="#6366f1" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round">
            <path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z" />
          </svg>
        </span>
        <h2 className={styles.emptyHeading}>No moderation items</h2>
        <p className={styles.emptyText}>Moderation tools coming soon.</p>
      </div>
    </div>
  );
}
