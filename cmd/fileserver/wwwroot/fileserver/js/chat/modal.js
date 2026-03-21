import { t } from "/js/components/i18n.js";
import { state, els, apiFetch, API } from "/js/chat/state.js";
import { renderSidebar, selectChannel } from "/js/chat/channels.js";

export function openJoinModal() {
    els.joinCode.value = "";
    els.joinName.value = "";
    els.joinError.classList.add("slv-hidden");
    els.joinError.textContent = "";
    els.joinModal.classList.remove("slv-hidden");
    // Defer focus until after the browser paints the now-visible modal.
    requestAnimationFrame(() => els.joinCode.focus());
}

export function closeJoinModal() {
    els.joinModal.classList.add("slv-hidden");
}

export async function submitJoin() {
    const code = els.joinCode.value.trim();
    if (!code) {
        showJoinError(t("chat_join_code_required"));
        return;
    }

    els.joinSubmit.disabled = true;

    try {
        const ch = await apiFetch(API.join, {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({ code, name: els.joinName.value.trim() }),
        });

        if (!state.channels.find(c => c.code === ch.code)) {
            state.channels.push(ch);
        }

        closeJoinModal();
        renderSidebar();
        await selectChannel(ch.code);
    } catch (err) {
        showJoinError(err.message || t("chat_join_error_generic"));
    } finally {
        els.joinSubmit.disabled = false;
    }
}

function showJoinError(msg) {
    els.joinError.textContent = msg;
    els.joinError.classList.remove("slv-hidden");
}