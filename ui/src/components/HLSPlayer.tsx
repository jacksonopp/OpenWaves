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
      const hls = new Hls();
      hls.loadSource(src);
      hls.attachMedia(audio as unknown as HTMLMediaElement);
      return () => hls.destroy();
    } else if (audio.canPlayType('application/vnd.apple.mpegurl')) {
      audio.src = src;
    }
  }, [src]);

  return <audio ref={audioRef} controls style={{ width: '100%', marginTop: '0.5rem' }} />;
}
