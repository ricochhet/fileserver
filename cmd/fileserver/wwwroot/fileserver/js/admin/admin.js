import { t, applyI18n } from "/js/components/i18n.js";
import { els, queryEls } from "/js/admin/state.js";
import { buildPanelHTML } from "/js/admin/panel.js";
import { initUsers, loadUsers } from "/js/admin/users.js";
import { initChannels, loadChannels } from "/js/admin/channels.js";
import { initUpload } from "/js/admin/upload.js";

/** Opens the admin panel and loads the initial user list. */
function openPanel() {
	els.panel.classList.remove("slv-hidden");
	document.body.style.overflow = "hidden";
	loadUsers();
}

/** Closes the admin panel and restores normal page scrolling. */
function closePanel() {
	els.panel.classList.add("slv-hidden");
	document.body.style.overflow = "";
}

/** Activates the named tab pane and loads its data if needed. */
function switchTab(name) {
	for (const tab of els.tabs) {
		const active = tab.dataset.tab === name;
		tab.classList.toggle("active", active);
		tab.setAttribute("aria-selected", String(active));
	}
	els.paneUsers.classList.toggle("slv-hidden", name !== "users");
	els.paneChannels.classList.toggle("slv-hidden", name !== "channels");
	els.paneUpload.classList.toggle("slv-hidden", name !== "upload");
	if (name === "channels") loadChannels();
}

/** Bootstraps the admin panel for admin users. No-op for non-admins. */
export function initAdmin(me) {
	if (!me?.isAdmin) return;

	const tmp = document.createElement("div");
	tmp.innerHTML = buildPanelHTML();
	document.body.appendChild(tmp.firstElementChild);
	queryEls();

	applyI18n({});

	const bar = document.getElementById("slv-user-bar");
	if (bar) {
		const adminBtn = document.createElement("button");
		adminBtn.id = "slv-admin-open";
		adminBtn.className = "slv-btn slv-admin-open-btn";
		adminBtn.textContent = t("admin_open_btn");
		adminBtn.addEventListener("click", openPanel);
		bar.appendChild(adminBtn);
	}

	els.closeBtn.addEventListener("click", closePanel);
	els.backdrop.addEventListener("click", closePanel);
	document.addEventListener("keydown", (e) => {
		if (e.key === "Escape" && !els.panel.classList.contains("slv-hidden"))
			closePanel();
	});

	for (const tab of els.tabs) {
		tab.addEventListener("click", () => switchTab(tab.dataset.tab));
	}

	initUsers();
	initChannels();
	initUpload();
}
