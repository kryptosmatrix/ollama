import { PauseIcon, SpeakerWaveIcon } from "@heroicons/react/24/outline";
import React, { useEffect, useState } from "react";
import { useNavigate } from "@tanstack/react-router";
import {
  commitTTSCache,
  speakMessageChunk,
} from "@/api";
import {
  speakSession,
  type SpeakState,
} from "@/lib/speakSession";

speakSession.setTransport({
  speak: speakMessageChunk,
  commit: commitTTSCache,
});

const SpeakButton: React.FC<{
  content: string;
  className?: string;
}> = ({ content, className = "" }) => {
  const navigate = useNavigate();
  const [state, setState] = useState<SpeakState>(speakSession.state);
  const [error, setError] = useState<string | null>(speakSession.error);

  useEffect(() => {
    return speakSession.subscribe((next, nextError) => {
      setState(next);
      setError(nextError);
    });
  }, []);

  const handleClick = async () => {
    await speakSession.toggle(content);
    const message = speakSession.error || "";
    if (message.includes("API key") || message.includes("Choose an ElevenLabs voice")) {
      navigate({ to: "/settings" });
    }
  };

  const playing = state === "playing";
  const iconSize = "h-7 w-7";
  const title =
    state === "playing"
      ? "Pause"
      : state === "paused"
        ? "Resume"
        : state === "generating"
          ? "Cancel speech"
          : "Speak";

  return (
    <button
      type="button"
      className={`${iconSize} px-1 py-0.5 text-xs cursor-pointer rounded-lg hover:bg-neutral-100 dark:hover:bg-neutral-800 flex items-center justify-center z-10 text-neutral-500 dark:text-neutral-400 ${className}`}
      onClick={handleClick}
      title={error || title}
      aria-busy={state === "generating"}
    >
      {playing ? (
        <PauseIcon className={iconSize} />
      ) : (
        <SpeakerWaveIcon className={iconSize} />
      )}
    </button>
  );
};

export default SpeakButton;
