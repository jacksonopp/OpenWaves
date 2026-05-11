export type AudioInputType = 'silence' | 'test_tone' | 'file';

export interface AudioInput {
  type: AudioInputType;
  file?: string;
}

export interface StationStatus {
  username: string;
  isLive: boolean;
  segmentCount: number;
  listenerCount: number;
  isRelaying: boolean;
  isIngesting: boolean;
  audioInput: AudioInput;
  isStatic: boolean;
}

export class AdminClient {
  private adminKey: string;
  private baseURL: string;

  constructor(adminKey: string, baseURL = '') {
    this.adminKey = adminKey;
    this.baseURL = baseURL;
  }

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

  async startIngest(username: string, audioFile?: string): Promise<void> {
    const r = await fetch(`${this.baseURL}/admin/stations/${username}/ingest/start`, {
      method: 'POST', headers: this.headers(),
      body: JSON.stringify({ audio_file: audioFile ?? '' }),
    });
    if (!r.ok) throw new Error(await r.text());
  }

  async stopIngest(username: string): Promise<void> {
    const r = await fetch(`${this.baseURL}/admin/stations/${username}/ingest/stop`, { method: 'POST', headers: this.headers() });
    if (!r.ok) throw new Error(await r.text());
  }

  async createChannel(data: {
    username: string;
    name?: string;
    summary?: string;
    relay_policy?: string;
    license_territory?: string[];
  }): Promise<StationStatus> {
    const r = await fetch(`${this.baseURL}/admin/channels`, {
      method: 'POST', headers: this.headers(),
      body: JSON.stringify(data),
    });
    if (!r.ok) throw new Error(await r.text());
    return r.json();
  }

  async deleteChannel(username: string): Promise<void> {
    const r = await fetch(`${this.baseURL}/admin/channels/${username}`, {
      method: 'DELETE', headers: this.headers(),
    });
    if (!r.ok) throw new Error(await r.text());
  }

  async setAudioInput(username: string, input: AudioInput): Promise<void> {
    const r = await fetch(`${this.baseURL}/admin/stations/${username}/ingest/input`, {
      method: 'POST', headers: this.headers(),
      body: JSON.stringify(input),
    });
    if (!r.ok) throw new Error(await r.text());
  }
}
