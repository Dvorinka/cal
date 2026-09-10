// Web push subscribe/unsubscribe against our /api/push endpoints.
// Returns true when a server-side subscription exists after the call.

import type { CalApi } from "@cal/api-client";

function b64encode(buf: ArrayBuffer): string {
  return btoa(String.fromCharCode(...new Uint8Array(buf)));
}

export async function enablePush(api: CalApi): Promise<boolean> {
  if (!("serviceWorker" in navigator) || typeof PushManager === "undefined") return false;
  const reg = await navigator.serviceWorker.ready;
  const vapidKey = await api.pushVapid();
  const existing = await reg.pushManager.getSubscription();
  const sub =
    existing ??
    (await reg.pushManager.subscribe({
      userVisibleOnly: true,
      applicationServerKey: urlBase64ToUint8Array(vapidKey),
    }));
  await api.pushSubscribe(sub.toJSON() as PushSubscriptionJSON);
  return true;
}

export async function disablePush(api: CalApi): Promise<void> {
  if (!("serviceWorker" in navigator)) return;
  const reg = await navigator.serviceWorker.ready;
  const sub = await reg.pushManager.getSubscription();
  if (sub) {
    await api.pushUnsubscribe(sub.endpoint).catch(() => {});
    await sub.unsubscribe();
  }
}

export async function pushEnabled(): Promise<boolean> {
  if (!("serviceWorker" in navigator)) return false;
  const reg = await navigator.serviceWorker.ready;
  return (await reg.pushManager.getSubscription()) !== null;
}

function urlBase64ToUint8Array(base64: string): BufferSource {
  const padding = "=".repeat((4 - (base64.length % 4)) % 4);
  const raw = atob((base64 + padding).replace(/-/g, "+").replace(/_/g, "/"));
  return Uint8Array.from(raw, (c) => c.charCodeAt(0)) as BufferSource;
}
