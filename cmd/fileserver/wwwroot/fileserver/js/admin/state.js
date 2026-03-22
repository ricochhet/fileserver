// Shared DOM element references, populated by queryEls() in admin.js
// after the panel HTML has been injected into the document.
// All admin sub-modules import this object; because ES modules are singletons
// the same reference is shared across all importers.
export let els = {};

/** Queries and caches all admin panel DOM element references into the shared els object. */
export function queryEls() {
	els.panel = document.getElementById("slv-admin-panel");
	els.backdrop = document.getElementById("slv-admin-backdrop");
	els.closeBtn = document.getElementById("slv-admin-close");
	els.tabs = document.querySelectorAll(".slv-admin-tab");
	els.paneUsers = document.getElementById("slv-admin-pane-users");
	els.paneChannels = document.getElementById("slv-admin-pane-channels");
	els.paneUpload = document.getElementById("slv-admin-pane-upload");

	els.auUsername = document.getElementById("slv-au-username");
	els.auPassword = document.getElementById("slv-au-password");
	els.auDisplay = document.getElementById("slv-au-display");
	els.auAdmin = document.getElementById("slv-au-admin");
	els.auError = document.getElementById("slv-au-error");
	els.auOk = document.getElementById("slv-au-ok");
	els.auSubmit = document.getElementById("slv-au-submit");
	els.auRefresh = document.getElementById("slv-au-refresh");
	els.userList = document.getElementById("slv-admin-user-list");

	els.acCode = document.getElementById("slv-ac-code");
	els.acName = document.getElementById("slv-ac-name");
	els.acError = document.getElementById("slv-ac-error");
	els.acOk = document.getElementById("slv-ac-ok");
	els.acSubmit = document.getElementById("slv-ac-submit");
	els.acRefresh = document.getElementById("slv-ac-refresh");
	els.channelList = document.getElementById("slv-admin-channel-list");

	els.upPath = document.getElementById("slv-up-path");
	els.upDrop = document.getElementById("slv-up-drop");
	els.upInput = document.getElementById("slv-up-input");
	els.upFilelist = document.getElementById("slv-up-filelist");
	els.upError = document.getElementById("slv-up-error");
	els.upOk = document.getElementById("slv-up-ok");
	els.upProgress = document.getElementById("slv-up-progress");
	els.upBar = document.getElementById("slv-up-bar");
	els.upSubmit = document.getElementById("slv-up-submit");
	els.upClear = document.getElementById("slv-up-clear");
}

/** Fetches a JSON endpoint, throwing on non-2xx responses. */
export async function apiFetch(url, options = {}) {
	const res = await fetch(url, options);
	if (res.status === 204) return null;
	if (!res.ok) {
		let msg = res.statusText;
		try {
			const body = await res.json();
			msg = body.error ?? body.message ?? msg;
		} catch {
			try {
				msg = (await res.text()) || msg;
			} catch {
				/* noop */
			}
		}
		throw new Error(msg);
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

/** Formats a byte count as a human-readable string. */
export function formatBytes(n) {
	if (n < 1024) return `${n} B`;
	if (n < 1024 * 1024) return `${(n / 1024).toFixed(1)} KB`;
	return `${(n / 1024 / 1024).toFixed(1)} MB`;
}
