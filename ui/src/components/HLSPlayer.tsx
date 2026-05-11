import { useEffect, useRef } from 'react';
import Hls from 'hls.js';

interface Props {
  src: string;
}

export default function HLSPlayer({ src }: Props) {
  const audioRef = useRef<HTMLAudioElement>(null);

  useEffect(() => {
    const audio = audioRef.current;
    if (!audio) return;

    if (Hls.isSupported()) {
      const hls = new Hls({
        // Start playback close to the live edge, not the oldest buffered segment.
        liveSyncDurationCount: 1,
        // Resync if playback falls more than 3 segments behind the live edge.
        liveMaxLatencyDurationCount: 3,
        // Discard played segments immediately so the user cannot rewind.
        liveBackBufferLength: 0,
      });
      hls.loadSource(src);
      hls.attachMedia(audio as unknown as HTMLMediaElement);

      // Recover from fatal errors automatically so a brief gap (e.g. during
      // an audio source switch) doesn't leave the player permanently stalled.
      hls.on(Hls.Events.ERROR, (_event, data) => {
        if (!data.fatal) return;
        if (data.type === Hls.ErrorTypes.NETWORK_ERROR) {
          hls.startLoad();
        } else if (data.type === Hls.ErrorTypes.MEDIA_ERROR) {
          hls.recoverMediaError();
        } else {
          hls.destroy();
        }
      });

      return () => hls.destroy();
    } else if (audio.canPlayType('application/vnd.apple.mpegurl')) {
      audio.src = src;
    }
  }, [src]);

  return <audio ref={audioRef} controls autoPlay style={{ width: '100%', marginTop: '0.5rem' }} />;
}
