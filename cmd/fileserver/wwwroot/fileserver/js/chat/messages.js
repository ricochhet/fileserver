import { t } from "/js/components/i18n.js";
import { state, els } from "/js/chat/state.js";

const GROUP_MS = 2 * 60 * 1000; // messages from the same user within this window share a header

/** Renders the full message history into the log. */
export function renderMessages(msgs) {
    els.log.innerHTML = "";

    if (!msgs || msgs.length === 0) {
        const empty = document.createElement("div");
        empty.className = "slv-chat-log-empty";
        empty.textContent = t("chat_log_empty");
        els.log.appendChild(empty);
        return;
    }

    let lastDate = null;

    for (let i = 0; i < msgs.length; i++) {
        const msg = msgs[i];
        const prev = msgs[i - 1] ?? null;

        const date = formatDate(msg.timestamp);
        if (date !== lastDate) {
            els.log.appendChild(makeDivider(date));
            lastDate = date;
        }

        const grouped =
            prev !== null &&
            prev.username === msg.username &&
            new Date(msg.timestamp) - new Date(prev.timestamp) < GROUP_MS &&
            formatDate(prev.timestamp) === date;

        els.log.appendChild(makeMessageEl(msg, grouped));
    }
}

/** Appends a single message to the live log, respecting grouping with the last message. */
export function appendMessage(msg) {
    els.log.querySelector(".slv-chat-log-empty")?.remove();

    const msgs = state.messages[msg.channelCode] ?? [];
    const prev = msgs[msgs.length - 1] ?? null;

    const date = formatDate(msg.timestamp);
    const lastDivider = els.log.querySelector(".slv-chat-divider:last-of-type")?.dataset.date;
    if (date !== lastDivider) {
        els.log.appendChild(makeDivider(date));
    }

    const grouped =
        prev !== null &&
        prev.username === msg.username &&
        new Date(msg.timestamp) - new Date(prev.timestamp) < GROUP_MS &&
        formatDate(prev.timestamp) === date;

    els.log.appendChild(makeMessageEl(msg, grouped));
}

export function scrollToBottom(instant = false) {
    els.log.scrollTo({ top: els.log.scrollHeight, behavior: instant ? "instant" : "smooth" });
}

export function isNearBottom() {
    return els.log.scrollHeight - els.log.scrollTop - els.log.clientHeight < 120;
}

function makeDivider(label) {
    const el = document.createElement("div");
    el.className = "slv-chat-divider";
    el.dataset.date = label;
    el.textContent = label;
    return el;
}

function makeMessageEl(msg, grouped) {
    const isSelf = state.me && msg.username === state.me.username;

    const wrap = document.createElement("div");
    wrap.className = "slv-chat-msg" +
        (grouped ? " slv-chat-msg-grouped" : "") +
        (isSelf ? " slv-chat-msg-self" : "");

    if (!grouped) {
        const header = document.createElement("div");
        header.className = "slv-chat-msg-header";

        const name = document.createElement("span");
        name.className = "slv-chat-msg-name" + (isSelf ? " is-self" : "");
        name.textContent = msg.displayName;

        const time = document.createElement("span");
        time.className = "slv-chat-msg-time";
        time.textContent = formatTime(msg.timestamp);
        time.title = new Date(msg.timestamp).toLocaleString();

        header.appendChild(name);
        header.appendChild(time);
        wrap.appendChild(header);
    }

    const body = document.createElement("div");
    body.className = "slv-chat-msg-body";
    body.textContent = msg.body;
    wrap.appendChild(body);

    return wrap;
}

function formatDate(iso) {
    const d = new Date(iso);
    const today = new Date();
    const yesterday = new Date(today);
    yesterday.setDate(yesterday.getDate() - 1);

    if (d.toDateString() === today.toDateString()) return t("chat_today");
    if (d.toDateString() === yesterday.toDateString()) return t("chat_yesterday");

    return d.toLocaleDateString(undefined, { year: "numeric", month: "long", day: "numeric" });
}

function formatTime(iso) {
    return new Date(iso).toLocaleTimeString(undefined, { hour: "2-digit", minute: "2-digit" });
}