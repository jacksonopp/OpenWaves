import { useEffect, useRef, useState } from 'react';
import { useAuth } from '../context/AuthContext';

const MAX_LINES = 200;

export default function LogFeed() {
  const { adminKey } = useAuth();
  const [lines, setLines] = useState<string[]>([]);
  const bottomRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!adminKey) return;

    const controller = new AbortController();

    (async () => {
      try {
        const resp = await fetch('/admin/logs', {
          headers: { 'Authorization': `Bearer ${adminKey}` },
          signal: controller.signal,
        });
        if (!resp.ok || !resp.body) return;
        const reader = resp.body.getReader();
        const decoder = new TextDecoder();
        let buf = '';
        while (true) {
          const { value, done } = await reader.read();
          if (done) break;
          buf += decoder.decode(value, { stream: true });
          const events = buf.split('\n\n');
          buf = events.pop() ?? '';
          for (const event of events) {
            const line = event.replace(/^data: /, '');
            if (line.trim()) {
              setLines(prev => [...prev.slice(-MAX_LINES + 1), line]);
            }
          }
        }
      } catch {
        // disconnected
      }
    })();

    return () => controller.abort();
  }, [adminKey]);

  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [lines]);

  return (
    <div style={{ background: '#0a0a0a', border: '1px solid #333', borderRadius: '6px', padding: '0.75rem', fontFamily: 'monospace', fontSize: '0.75rem', color: '#22c55e', height: '200px', overflowY: 'auto' }}>
      <div style={{ fontWeight: 'bold', color: '#888', marginBottom: '0.5rem' }}>Server Logs</div>
      {lines.length === 0 && <div style={{ color: '#555' }}>Waiting for logs…</div>}
      {lines.map((l, i) => <div key={i}>{l}</div>)}
      <div ref={bottomRef} />
    </div>
  );
}
