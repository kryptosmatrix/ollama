export type SpeakState = "idle" | "generating" | "playing" | "paused";

export type AudioHandle = {
  play: () => Promise<void>;
  pause: () => void;
  currentTime: number;
  src: string;
  onended: ((this: unknown, ev: Event) => unknown) | null;
  onerror: ((this: unknown, ev: Event) => unknown) | null;
};

export type SpeakChunk = {
  blob: Blob;
  contentType: string;
  chunkIndex: number;
  chunkCount: number;
  fingerprint: string;
};

export type SpeakTransport = {
  speak: (
    text: string,
    chunkIndex: number,
    signal: AbortSignal,
  ) => Promise<SpeakChunk>;
  commit: (fingerprint: string) => Promise<void>;
};

const AUDIO_MIME = new Set([
  "audio/mpeg",
  "audio/mp3",
  "application/octet-stream",
]);

export function isPlayableSpeechMIME(contentType: string): boolean {
  const mime = contentType.split(";")[0].trim().toLowerCase();
  return AUDIO_MIME.has(mime);
}

// Pause must not rewind. A handler that assigns currentTime = 0 or calls load()
// is stop labelled as pause.
export function pausePlayback(audio: AudioHandle): number {
  audio.pause();
  return audio.currentTime;
}

export async function unlockAudioContext(): Promise<void> {
  const Ctx =
    typeof window === "undefined"
      ? undefined
      : window.AudioContext ||
        (window as unknown as { webkitAudioContext?: typeof AudioContext })
          .webkitAudioContext;
  if (!Ctx) {
    return;
  }
  const ctx = new Ctx();
  if (ctx.state === "suspended") {
    await ctx.resume();
  }
  const buffer = ctx.createBuffer(1, 1, 22050);
  const src = ctx.createBufferSource();
  src.buffer = buffer;
  src.connect(ctx.destination);
  src.start(0);
}

type Listener = (state: SpeakState, error: string | null) => void;

class SpeakSession {
  state: SpeakState = "idle";
  error: string | null = null;
  private listeners = new Set<Listener>();
  private audio: AudioHandle | null = null;
  private objectURL: string | null = null;
  private abort: AbortController | null = null;
  private generation = 0;
  private makeAudio: () => AudioHandle;
  private transport: SpeakTransport | null = null;
  private playWait: {
    resolve: () => void;
    reject: (err: Error) => void;
  } | null = null;

  constructor(makeAudio: () => AudioHandle) {
    this.makeAudio = makeAudio;
  }

  setTransport(transport: SpeakTransport) {
    this.transport = transport;
  }

  setAudioFactory(makeAudio: () => AudioHandle) {
    this.makeAudio = makeAudio;
  }

  subscribe(listener: Listener): () => void {
    this.listeners.add(listener);
    listener(this.state, this.error);
    return () => {
      this.listeners.delete(listener);
    };
  }

  async toggle(text: string) {
    if (this.state === "playing" && this.audio) {
      pausePlayback(this.audio);
      this.setState("paused", null);
      return;
    }
    if (this.state === "paused" && this.audio) {
      try {
        await this.audio.play();
        this.setState("playing", null);
      } catch (err) {
        this.setState("idle", playError(err));
      }
      return;
    }
    if (this.state === "generating") {
      this.cancel();
      return;
    }
    await this.start(text);
  }

  cancel() {
    this.generation += 1;
    this.abort?.abort();
    this.abort = nilAbort();
    this.stopAudio();
    this.setState("idle", null);
  }

  private async start(text: string) {
    this.generation += 1;
    const gen = this.generation;
    this.abort?.abort();
    this.abort = new AbortController();
    this.stopAudio();
    this.setState("generating", null);
    try {
      await unlockAudioContext();
      if (gen !== this.generation) {
        return;
      }
      const transport = this.transport;
      if (!transport) {
        throw new Error("Speech is not wired.");
      }
      let index = 0;
      let count = 1;
      while (index < count) {
        if (gen !== this.generation) {
          return;
        }
        const chunk = await transport.speak(text, index, this.abort.signal);
        if (gen !== this.generation) {
          return;
        }
        if (!isPlayableSpeechMIME(chunk.contentType)) {
          throw new Error("ElevenLabs returned data that was not audio.");
        }
        count = chunk.chunkCount;
        await this.playBlob(chunk.blob, gen);
        if (gen !== this.generation) {
          return;
        }
        try {
          await transport.commit(chunk.fingerprint);
        } catch {
          // A missed commit costs a cache write, not playback.
        }
        index += 1;
      }
      if (gen === this.generation) {
        this.setState("idle", null);
      }
    } catch (err) {
      if (isAbort(err) || gen !== this.generation) {
        if (gen === this.generation) {
          this.setState("idle", null);
        }
        return;
      }
      this.setState("idle", playError(err));
    }
  }

  private playBlob(blob: Blob, gen: number): Promise<void> {
    return new Promise((resolve, reject) => {
      this.stopAudio();
      const url = URL.createObjectURL(blob);
      this.objectURL = url;
      const audio = this.makeAudio();
      this.audio = audio;
      this.playWait = {
        resolve: () => resolve(),
        reject,
      };
      audio.src = url;
      audio.onended = () => {
        this.playWait = null;
        resolve();
      };
      audio.onerror = () => {
        this.playWait = null;
        reject(new Error("The audio could not be played."));
      };
      this.setState("playing", null);
      audio.play().catch((err) => {
        this.playWait = null;
        reject(err instanceof Error ? err : new Error(playError(err)));
      });
      if (gen !== this.generation) {
        this.playWait = null;
        resolve();
      }
    });
  }

  private stopAudio() {
    if (this.playWait) {
      this.playWait.resolve();
      this.playWait = null;
    }
    if (this.audio) {
      this.audio.pause();
      this.audio.onended = null;
      this.audio.onerror = null;
      this.audio.src = "";
      this.audio = null;
    }
    if (this.objectURL) {
      URL.revokeObjectURL(this.objectURL);
      this.objectURL = null;
    }
  }

  private setState(state: SpeakState, error: string | null) {
    this.state = state;
    this.error = error;
    for (const listener of this.listeners) {
      listener(state, error);
    }
  }
}

function nilAbort(): AbortController | null {
  return null;
}

function isAbort(err: unknown): boolean {
  return (
    (err instanceof DOMException && err.name === "AbortError") ||
    (err instanceof Error && err.name === "AbortError")
  );
}

function playError(err: unknown): string {
  if (err instanceof Error && err.message) {
    return err.message;
  }
  return "Speech could not be played.";
}

function browserAudio(): AudioHandle {
  return new Audio() as unknown as AudioHandle;
}

export const speakSession = new SpeakSession(browserAudio);

export function pauseDoesNotRewind(audio: AudioHandle): boolean {
  const before = audio.currentTime;
  pausePlayback(audio);
  return audio.currentTime === before;
}
