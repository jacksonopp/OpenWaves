import { Link } from '@tanstack/react-router';
import styles from './TopBar.module.css';

export default function TopBar() {
  return (
    <header className={styles.topBar}>
      <div className={styles.left}>
        <span className={styles.logo}>((·))</span>
        <span className={styles.brandName}>OpenWaves Protocol</span>
        <div className={styles.divider} />
        <nav className={styles.tabs}>
          <Link to="/" className={styles.tabClient}>
            Client
          </Link>
          <span className={styles.tabAdmin}>Admin</span>
        </nav>
      </div>
      <div className={styles.right}>
        <span className={styles.fedText}>Federated via ActivityPub</span>
      </div>
    </header>
  );
}
