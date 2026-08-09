import { useEffect, useState } from "react";
import { apiBase } from "../services/api";

export type RealtimeEvent = {
  id: string;
  type: string;
  resourceType: string;
  resourceId: string;
  publishedAt: string;
};

const eventTypes = [
  "branding.updated",
  "matter.created",
  "document.uploaded",
  "document.deleted",
  "document.restored",
  "deadline.created",
  "deadline.updated",
  "task.created",
  "task.updated",
];

export function useRealtime() {
  const [connected, setConnected] = useState(false);
  const [latest, setLatest] = useState<RealtimeEvent>();
  useEffect(() => {
    const source = new EventSource(`${apiBase}/api/v1/stream`, {
      withCredentials: true,
    });
    const receive = (message: MessageEvent<string>) => {
      try {
        const event = JSON.parse(message.data) as RealtimeEvent;
        setLatest(event);
        window.dispatchEvent(
          new CustomEvent("lawoffice:realtime", { detail: event }),
        );
      } catch {
        // Invalid server events are ignored; operational data is reloaded via API.
      }
    };
    source.onopen = () => setConnected(true);
    source.onerror = () => setConnected(false);
    eventTypes.forEach((type) => source.addEventListener(type, receive));
    return () => {
      eventTypes.forEach((type) => source.removeEventListener(type, receive));
      source.close();
    };
  }, []);
  useEffect(() => {
    if (!latest) return;
    const timer = window.setTimeout(() => setLatest(undefined), 5000);
    return () => window.clearTimeout(timer);
  }, [latest]);
  return { connected, latest };
}
