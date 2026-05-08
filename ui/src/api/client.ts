export interface StationStatus {
  username: string;
  isLive: boolean;
  segmentCount: number;
  listenerCount: number;
  isRelaying: boolean;
}

export class AdminClient {
  constructor(private adminKey: string, private baseURL = '') {}

  private headers(): HeadersInit {
    return { 'Authorization': `Bearer ${this.adminKey}`, 'Content-Type': 'application/json' };
  }

  async listStations(): Promise<StationStatus[]> {
    const r = await fetch(`${this.baseURL}/admin/stations`, { headers: this.headers() });
    if (!r.ok) throw new Error(`${r.status}`);
    return r.json();
  }

  async startStream(username: string): Promise<void> {
    const r = await fetch(`${this.baseURL}/admin/stations/${username}/stream/start`, { method: 'POST', headers: this.headers() });
    if (!r.ok) throw new Error(`${r.status}`);
  }

  async stopStream(username: string): Promise<void> {
    const r = await fetch(`${this.baseURL}/admin/stations/${username}/stream/stop`, { method: 'POST', headers: this.headers() });
    if (!r.ok) throw new Error(`${r.status}`);
  }

  async startRelay(username: string, sourceURL: string): Promise<void> {
    const r = await fetch(`${this.baseURL}/admin/stations/${username}/relay/start`, {
      method: 'POST', headers: this.headers(),
      body: JSON.stringify({ source_url: sourceURL }),
    });
    if (!r.ok) throw new Error(`${r.status}`);
  }

  async stopRelay(username: string): Promise<void> {
    const r = await fetch(`${this.baseURL}/admin/stations/${username}/relay/stop`, { method: 'POST', headers: this.headers() });
    if (!r.ok) throw new Error(`${r.status}`);
  }
}
