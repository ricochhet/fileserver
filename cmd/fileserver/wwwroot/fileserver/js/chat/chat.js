import {
	t,
	detectLang,
	loadLocale,
	setStrings,
	applyI18n,
	buildLangSelector,
	whenReady,
} from "/js/components/i18n.js";
import { state, els, apiFetch, escHtml, API } from "/js/chat/state.js";
import {
	appendMessage,
	renderMessages,
	scrollToBottom,
	isNearBottom,
} from "/js/chat/messages.js";
import { renderSidebar, selectChannel, showEmpty } from "/js/chat/channels.js";
import { openJoinModal, closeJoinModal, submitJoin } from "/js/chat/modal.js";

// Delay (ms) before the next reconnect attempt. Doubles on each failure up to
// MAX_RECONNECT_DELAY, then resets to MIN_RECONNECT_DELAY on a clean open.
const MIN_RECONNECT_DELAY = 1_000;
const MAX_RECONNECT_DELAY = 30_000;
let reconnectDelay = MIN_RECONNECT_DELAY;
let reconnectTimer = null;

/** Updates the connection status banner; hides it when the key is "connected". */
function setConnStatus(key) {
	if (!els.connStatus) return;
	if (key === "connected") {
		els.connStatus.textContent = "";
		els.connStatus.classList.add("slv-hidden");
	} else {
		els.connStatus.textContent = t(key);
		els.connStatus.classList.remove("slv-hidden");
	}
}

/** Re-fetches message history for a channel and updates the view if it is currently active. */
async function refreshChannel(code) {
	try {
		const msgs = await apiFetch(
			`${API.messages}?channel=${encodeURIComponent(code)}`,
		);
		state.messages[code] = msgs ?? [];
		if (code === state.active) {
			renderMessages(state.messages[code]);
			scrollToBottom(true);
		}
	} catch {
		// Non-fatal; existing messages remain displayed.
	}
}

/** Opens an SSE connection with exponential-backoff reconnect on failure. */
function connectSSE() {
	state.sse?.close();
	state.sse = null;

	const es = new EventSource(API.events);
	state.sse = es;

	es.onopen = () => {
		reconnectDelay = MIN_RECONNECT_DELAY;
		setConnStatus("connected");

		for (const code of Object.keys(state.messages)) {
			if (code !== state.active) {
				delete state.messages[code];
			}
		}
		if (state.active) {
			refreshChannel(state.active);
		}
	};

	es.onmessage = (e) => {
		let msg;
		try {
			msg = JSON.parse(e.data);
		} catch {
			return;
		}
		handleIncomingMessage(msg);
	};

	es.onerror = () => {
		es.close();
		state.sse = null;
		setConnStatus("chat_reconnecting");

		clearTimeout(reconnectTimer);
		reconnectTimer = setTimeout(() => {
			reconnectTimer = null;
			connectSSE();
		}, reconnectDelay);

		reconnectDelay = Math.min(reconnectDelay * 2, MAX_RECONNECT_DELAY);
	};
}

/** Fetches the current user and channel list, then starts the SSE connection. */
async function boot() {
	try {
		state.me = await apiFetch(API.me);
	} catch {
		return;
	}

	state.channels = (await apiFetch(API.channels).catch(() => [])) ?? [];

	renderIdentity();
	renderSidebar();

	if (state.channels.length > 0) {
		await selectChannel(state.channels[0].code);
	} else {
		showEmpty();
	}

	connectSSE();
	bindEvents();
}

/** Renders the signed-in user's display name into the identity element. */
function renderIdentity() {
	if (!state.me) return;
	els.identity.innerHTML =
		`<span class="slv-chat-identity-label">${escHtml(t("chat_signed_in_as"))}</span>` +
		`<span class="slv-chat-display-name">${escHtml(state.me.displayName)}</span>`;
}

/** Routes an incoming SSE message to the active log or increments the unread count. */
function handleIncomingMessage(msg) {
	const { channelCode: code } = msg;
	if (!state.channels.find((c) => c.code === code)) return;

	if (!state.messages[code]) state.messages[code] = [];
	state.messages[code].push(msg);

	if (code === state.active) {
		const near = isNearBottom();
		appendMessage(msg);
		if (near) scrollToBottom();
	} else {
		state.unread[code] = (state.unread[code] ?? 0) + 1;
		renderSidebar();
	}
}

/** Reads the input, sends a POST, and re-enables the form on completion. */
async function sendMessage() {
	const body = els.input.value.trim();
	if (!body || !state.active) return;

	els.input.value = "";
	els.input.disabled = true;
	els.sendBtn.disabled = true;

	try {
		await apiFetch(API.post, {
			method: "POST",
			headers: { "Content-Type": "application/json" },
			body: JSON.stringify({ channel: state.active, body }),
		});
	} catch (err) {
		els.input.value = body;
		console.error("send failed:", err);
	} finally {
		els.input.disabled = false;
		els.sendBtn.disabled = false;
		els.input.focus();
	}
}

/** Attaches all UI event listeners after the initial render. */
function bindEvents() {
	els.sendBtn.addEventListener("click", sendMessage);
	els.input.addEventListener("keydown", (e) => {
		if (e.key === "Enter" && !e.shiftKey) {
			e.preventDefault();
			sendMessage();
		}
	});

	els.leaveBtn.addEventListener("click", leaveCurrentChannel);

	const openJoin = () => openJoinModal();
	els.joinBtn.addEventListener("click", openJoin);
	els.joinEmptyBtn.addEventListener("click", openJoin);

	els.joinSubmit.addEventListener("click", submitJoin);
	els.joinCancel.addEventListener("click", closeJoinModal);
	els.joinBackdrop.addEventListener("click", closeJoinModal);
	els.joinCode.addEventListener("keydown", (e) => {
		if (e.key === "Enter") submitJoin();
		if (e.key === "Escape") closeJoinModal();
	});
	els.joinName.addEventListener("keydown", (e) => {
		if (e.key === "Enter") submitJoin();
		if (e.key === "Escape") closeJoinModal();
	});
	document.addEventListener("keydown", (e) => {
		if (
			e.key === "Escape" &&
			!els.joinModal.classList.contains("slv-hidden")
		)
			closeJoinModal();
	});
}

/** Leaves the active channel, removes it from local state, and switches to the next channel or empty state.
 *  Kept in chat.js rather than channels.js to avoid a circular import with bindEvents. */
async function leaveCurrentChannel() {
	const code = state.active;
	if (!code) return;

	try {
		await apiFetch(API.leave, {
			method: "POST",
			headers: { "Content-Type": "application/json" },
			body: JSON.stringify({ code }),
		});
	} catch {
		// Best-effort; remove locally regardless.
	}

	state.channels = state.channels.filter((c) => c.code !== code);
	delete state.messages[code];
	delete state.unread[code];

	state.active = state.channels[0]?.code ?? null;
	renderSidebar();

	if (state.active) {
		await selectChannel(state.active);
	} else {
		showEmpty();
	}
}

await whenReady(async () => {
	applyI18n({});

	buildLangSelector(async (lang) => {
		localStorage.setItem("fs_lang", lang);
		const data = await loadLocale(lang);
		setStrings(data);
		applyI18n({});
		renderIdentity();
	});

	const lang = detectLang();
	if (lang !== "en") {
		const data = await loadLocale(lang);
		if (Object.keys(data).length) {
			setStrings(data);
			applyI18n({});
		}
	}

	await boot();
});
