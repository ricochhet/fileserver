import { t } from "/js/components/i18n.js";
import { state, els, apiFetch, API } from "/js/chat/state.js";
import { renderMessages, scrollToBottom } from "/js/chat/messages.js";

/** Re-renders the channel list sidebar with unread badges and active state. */
export function renderSidebar() {
	els.channelList.innerHTML = "";

	if (state.channels.length === 0) {
		const p = document.createElement("p");
		p.className = "slv-chat-no-channels";
		p.textContent = t("chat_no_channels");
		els.channelList.appendChild(p);
		return;
	}

	for (const ch of state.channels) {
		const item = document.createElement("button");
		item.className =
			"slv-chat-channel-item" +
			(ch.code === state.active ? " active" : "");
		item.dataset.code = ch.code;

		const name = document.createElement("span");
		name.className = "slv-chat-channel-item-name";
		name.textContent = ch.name;
		item.appendChild(name);

		const unread = state.unread[ch.code];
		if (unread) {
			const badge = document.createElement("span");
			badge.className = "slv-chat-unread-badge";
			badge.textContent = unread > 99 ? "99+" : String(unread);
			item.appendChild(badge);
		}

		item.addEventListener("click", () => selectChannel(ch.code));
		els.channelList.appendChild(item);
	}
}

/** Switches the active channel, loads its history if needed, and updates the view. */
export async function selectChannel(code) {
	state.active = code;
	delete state.unread[code];
	renderSidebar();

	const ch = state.channels.find((c) => c.code === code);
	if (!ch) return;

	els.channelName.textContent = ch.name;
	els.channelCode.textContent = code;

	showChatView();

	if (!state.messages[code]) {
		state.messages[code] = [];
		try {
			state.messages[code] =
				(await apiFetch(
					`${API.messages}?channel=${encodeURIComponent(code)}`,
				)) ?? [];
		} catch {
			// Non-fatal; messages will arrive via SSE going forward.
		}
	}

	renderMessages(state.messages[code]);
	scrollToBottom(true);
	els.input.focus();
}

/** Shows the empty state and hides the chat view. */
export function showEmpty() {
	els.emptyState.classList.remove("slv-hidden");
	els.chatView.classList.add("slv-hidden");
}

/** Hides the empty state and shows the chat view. */
export function showChatView() {
	els.emptyState.classList.add("slv-hidden");
	els.chatView.classList.remove("slv-hidden");
}
