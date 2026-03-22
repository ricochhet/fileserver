import { t } from "/js/components/i18n.js";
import { els, apiFetch, escHtml } from "/js/admin/state.js";

/** Shows an ok or error message in the channel form, hiding the other. */
function setMsg(type, msg) {
	els.acOk.classList.add("slv-hidden");
	els.acError.classList.add("slv-hidden");
	if (type === "ok") {
		els.acOk.textContent = msg;
		els.acOk.classList.remove("slv-hidden");
	}
	if (type === "err") {
		els.acError.textContent = msg;
		els.acError.classList.remove("slv-hidden");
	}
}

/** Fetches the channel list from the API and renders it into the panel. */
export async function loadChannels() {
	els.channelList.innerHTML = `<span class="slv-admin-loading">${escHtml(t("admin_loading"))}</span>`;
	try {
		const channels = await apiFetch("/admin/channels");
		renderChannelList(channels ?? []);
	} catch (e) {
		els.channelList.innerHTML = `<span class="slv-admin-load-error">${escHtml(e.message)}</span>`;
	}
}

/** Renders an array of channels as a table, or an empty state message. */
function renderChannelList(channels) {
	if (channels.length === 0) {
		els.channelList.innerHTML = `<span class="slv-admin-empty">${escHtml(t("admin_no_channels"))}</span>`;
		return;
	}

	const table = document.createElement("table");
	table.className = "slv-admin-utable";

	const thead = document.createElement("thead");
	thead.innerHTML =
		`<tr>` +
		`<th>${escHtml(t("admin_col_code"))}</th>` +
		`<th>${escHtml(t("admin_col_channel_name"))}</th>` +
		`<th></th>` +
		`</tr>`;
	table.appendChild(thead);

	const tbody = document.createElement("tbody");
	for (const ch of channels) {
		const tr = document.createElement("tr");
		tr.innerHTML =
			`<td class="slv-admin-utd-name">${escHtml(ch.code)}</td>` +
			`<td class="slv-admin-utd-dn">${escHtml(ch.name)}</td>` +
			`<td></td>`;

		const delBtn = document.createElement("button");
		delBtn.className = "slv-btn slv-admin-del-btn";
		delBtn.textContent = t("admin_delete_channel");
		delBtn.addEventListener("click", () => deleteChannel(ch.code, tr));
		tr.querySelector("td:last-child").appendChild(delBtn);

		tbody.appendChild(tr);
	}
	table.appendChild(tbody);

	els.channelList.innerHTML = "";
	els.channelList.appendChild(table);
}

/** Reads the create-channel form, validates it, and submits a POST request. */
async function createChannel() {
	setMsg(null, "");

	const code = els.acCode.value.trim();
	const name = els.acName.value.trim();

	if (!code) {
		setMsg("err", t("admin_err_code_required"));
		return;
	}

	els.acSubmit.disabled = true;
	try {
		await apiFetch("/admin/channels", {
			method: "POST",
			headers: { "Content-Type": "application/json" },
			body: JSON.stringify({ code, name }),
		});
		setMsg("ok", t("admin_channel_created", { c: code }));
		els.acCode.value = "";
		els.acName.value = "";
		await loadChannels();
	} catch (e) {
		setMsg("err", e.message);
	} finally {
		els.acSubmit.disabled = false;
	}
}

/** Sends a DELETE request for the given channel code and removes its row on success. */
async function deleteChannel(code, rowEl) {
	if (!confirm(t("admin_confirm_delete_channel", { c: code }))) return;

	rowEl.style.opacity = "0.4";
	try {
		await apiFetch(`/admin/channels/${encodeURIComponent(code)}`, {
			method: "DELETE",
		});
		rowEl.remove();
	} catch (e) {
		rowEl.style.opacity = "";
		alert(e.message);
	}
}

/** Binds all channel-tab event listeners. Call once after the panel is in the DOM. */
export function initChannels() {
	els.acSubmit.addEventListener("click", createChannel);
	els.acRefresh.addEventListener("click", loadChannels);
	[els.acCode, els.acName].forEach((inp) =>
		inp.addEventListener("keydown", (e) => {
			if (e.key === "Enter") createChannel();
		}),
	);
}
