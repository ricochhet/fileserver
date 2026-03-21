const chatRoute = window.FS_CHAT_ROUTE ?? "/chat";

export const API = {
    me: `${chatRoute}/api/me`,
    channels: `${chatRoute}/api/channels`,
    join: `${chatRoute}/api/channels/join`,
    leave: `${chatRoute}/api/channels/leave`,
    messages: `${chatRoute}/api/messages`,
    post: `${chatRoute}/api/messages`,
    events: `${chatRoute}/api/events`,
};

export const state = {
    /** @type {{ username: string, displayName: string } | null} */
    me: null,
    /** @type {Array<{ code: string, name: string }>} */
    channels: [],
    /** @type {string | null} */
    active: null,
    /** @type {Record<string, Array>} message history per channel code */
    messages: {},
    /** @type {Record<string, number>} unread counts per channel code */
    unread: {},
    /** @type {EventSource | null} */
    sse: null,
};

const $ = id => document.getElementById(id);

export const els = {
    channelList: $("slv-chat-channels"),
    identity: $("slv-chat-identity"),
    emptyState: $("slv-chat-empty"),
    chatView: $("slv-chat-view"),
    channelName: $("slv-chat-channel-name"),
    channelCode: $("slv-chat-channel-code"),
    log: $("slv-chat-log"),
    input: $("slv-chat-input"),
    sendBtn: $("slv-chat-send"),
    leaveBtn: $("slv-leave-btn"),
    joinBtn: $("slv-join-btn"),
    joinEmptyBtn: $("slv-join-empty-btn"),
    joinModal: $("slv-join-modal"),
    joinBackdrop: $("slv-join-backdrop"),
    joinCode: $("slv-join-code"),
    joinName: $("slv-join-name"),
    joinError: $("slv-join-error"),
    joinSubmit: $("slv-join-submit"),
    joinCancel: $("slv-join-cancel"),
    connStatus: $("slv-conn-status"),
};

/** Fetches a JSON endpoint, throwing on non-2xx responses. */
export async function apiFetch(url, options = {}) {
    const res = await fetch(url, options);
    if (res.status === 204) return null;
    if (!res.ok) {
        const text = await res.text().catch(() => "");
        throw new Error(text || res.statusText);
    }
    return res.json();
}

/** Escapes a string for safe HTML insertion. */
export function escHtml(s) {
    return String(s)
        .replaceAll("&", "&amp;")
        .replaceAll("<", "&lt;")
        .replaceAll(">", "&gt;")
        .replaceAll('"', "&quot;");
}