import { useEffect, useState } from "react";
import { Switch } from "@/components/ui/switch";
import { Input } from "@/components/ui/input";
import { Field, Label, Description } from "@/components/ui/fieldset";
import { Button } from "@/components/ui/button";
import { SpeakerWaveIcon } from "@heroicons/react/20/solid";
import {
  clearTTSCache,
  deleteTTSKey,
  getTTSStatus,
  getTTSVoices,
  putTTSKey,
  updateTTSSettings,
  type TTSStatus,
  type TTSVoice,
} from "@/api";

export function voicesAfterKeyChange(
  hasAPIKey: boolean,
  voices: TTSVoice[],
): TTSVoice[] {
  return hasAPIKey ? voices : [];
}

export default function SpeechSettings() {
  const [status, setStatus] = useState<TTSStatus | null>(null);
  const [voices, setVoices] = useState<TTSVoice[]>([]);
  const [keyDraft, setKeyDraft] = useState("");
  const [message, setMessage] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  const load = async () => {
    const next = await getTTSStatus();
    setStatus(next);
    if (next.has_api_key) {
      try {
        setVoices(await getTTSVoices());
      } catch (err) {
        setVoices([]);
        setError(err instanceof Error ? err.message : "Could not list voices.");
      }
    } else {
      setVoices([]);
    }
  };

  useEffect(() => {
    load().catch((err) => {
      setError(err instanceof Error ? err.message : "Could not load speech settings.");
    });
  }, []);

  const shownVoices = voicesAfterKeyChange(!!status?.has_api_key, voices);

  const saveKey = async () => {
    setBusy(true);
    setError(null);
    try {
      await putTTSKey(keyDraft.trim());
      setKeyDraft("");
      setMessage("API key saved.");
      await load();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Could not save the key.");
    } finally {
      setBusy(false);
    }
  };

  const removeKey = async () => {
    setBusy(true);
    setError(null);
    try {
      await deleteTTSKey();
      setVoices([]);
      setMessage("API key removed.");
      await load();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Could not remove the key.");
    } finally {
      setBusy(false);
    }
  };

  const patch = async (next: Parameters<typeof updateTTSSettings>[0]) => {
    setBusy(true);
    setError(null);
    try {
      await updateTTSSettings(next);
      await load();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Could not save speech settings.");
    } finally {
      setBusy(false);
    }
  };

  if (!status) {
    return null;
  }

  return (
    <div className="overflow-hidden rounded-xl bg-white dark:bg-neutral-800">
      <div className="space-y-4 p-4">
        <Field>
          <div className="flex items-start space-x-3">
            <SpeakerWaveIcon className="mt-1 h-5 w-5 flex-shrink-0 text-black dark:text-neutral-100" />
            <div className="w-full space-y-3">
              <div>
                <Label>Speech</Label>
                <Description>
                  Read finished replies in your ElevenLabs voice. The key is kept
                  in {status.secret_store}.
                </Description>
              </div>
              <Input
                type="password"
                autoComplete="off"
                placeholder={
                  status.has_api_key ? "API key saved — paste to replace" : "ElevenLabs API key"
                }
                value={keyDraft}
                onChange={(e) => setKeyDraft(e.target.value)}
              />
              <div className="flex gap-2">
                <Button
                  type="button"
                  color="white"
                  disabled={busy || !keyDraft.trim()}
                  onClick={saveKey}
                >
                  Save key
                </Button>
                {status.has_api_key && (
                  <Button type="button" color="zinc" disabled={busy} onClick={removeKey}>
                    Remove key
                  </Button>
                )}
              </div>
              {status.has_api_key && (
                <>
                  <div>
                    <Label>Voice</Label>
                    <select
                      className="mt-1 w-full rounded-lg border border-neutral-200 bg-white px-3 py-2 text-sm dark:border-neutral-700 dark:bg-neutral-900 dark:text-white"
                      value={status.voice_id}
                      onChange={(e) => patch({ voice_id: e.target.value })}
                      disabled={busy || shownVoices.length === 0}
                    >
                      <option value="">Choose a voice</option>
                      {shownVoices.map((voice) => (
                        <option key={voice.voice_id} value={voice.voice_id}>
                          {voice.name}
                        </option>
                      ))}
                    </select>
                  </div>
                  <div>
                    <Label>Model</Label>
                    <select
                      className="mt-1 w-full rounded-lg border border-neutral-200 bg-white px-3 py-2 text-sm dark:border-neutral-700 dark:bg-neutral-900 dark:text-white"
                      value={status.model_id}
                      onChange={(e) => patch({ model_id: e.target.value })}
                      disabled={busy}
                    >
                      <option value="eleven_flash_v2_5">Flash v2.5 — fastest</option>
                      <option value="eleven_multilingual_v2">
                        Multilingual v2 — richer
                      </option>
                    </select>
                  </div>
                  <div>
                    <Label>Speed {status.speed.toFixed(2)}</Label>
                    <input
                      type="range"
                      min={0.7}
                      max={1.2}
                      step={0.01}
                      value={status.speed}
                      disabled={busy}
                      className="mt-1 w-full"
                      onChange={(e) => patch({ speed: Number(e.target.value) })}
                    />
                  </div>
                  <div className="flex items-start justify-between gap-4">
                    <div>
                      <Label>Private audio cache</Label>
                      <Description>
                        Off by default. When on, a successful play may save an
                        encrypted copy so the same reply is not billed twice.
                      </Description>
                    </div>
                    <Switch
                      checked={status.cache_enabled}
                      disabled={busy}
                      onChange={(checked) => patch({ cache_enabled: checked })}
                    />
                  </div>
                  <Button
                    type="button"
                    color="white"
                    disabled={busy}
                    onClick={async () => {
                      setBusy(true);
                      try {
                        await clearTTSCache();
                        setMessage("Speech cache cleared.");
                        await load();
                      } catch (err) {
                        setError(
                          err instanceof Error ? err.message : "Could not clear the cache.",
                        );
                      } finally {
                        setBusy(false);
                      }
                    }}
                  >
                    Clear speech cache
                  </Button>
                </>
              )}
              {message && (
                <Description className="text-green-600 dark:text-green-400">
                  {message}
                </Description>
              )}
              {error && (
                <Description className="text-red-600 dark:text-red-400">
                  {error}
                </Description>
              )}
            </div>
          </div>
        </Field>
      </div>
    </div>
  );
}
