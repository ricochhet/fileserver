import { t } from "/js/components/i18n.js";
import { els, apiFetch, escHtml } from "/js/admin/state.js";

/** Shows an ok or error message in the user form, hiding the other. */
function setMsg(type, msg) {
	els.auOk.classList.add("slv-hidden");
	els.auError.classList.add("slv-hidden");
	if (type === "ok") {
		els.auOk.textContent = msg;
		els.auOk.classList.remove("slv-hidden");
	}
	if (type === "err") {
		els.auError.textContent = msg;
		els.auError.classList.remove("slv-hidden");
	}
}

/** Fetches the user list from the API and renders it into the panel. */
export async function loadUsers() {
	els.userList.innerHTML = `<span class="slv-admin-loading">${escHtml(t("admin_loading"))}</span>`;
	try {
		const users = await apiFetch("/admin/users");
		renderUserList(users ?? []);
	} catch (e) {
		els.userList.innerHTML = `<span class="slv-admin-load-error">${escHtml(e.message)}</span>`;
	}
}

/** Renders an array of users as a table, or an empty state message. */
function renderUserList(users) {
	if (users.length === 0) {
		els.userList.innerHTML = `<span class="slv-admin-empty">${escHtml(t("admin_no_users"))}</span>`;
		return;
	}

	const table = document.createElement("table");
	table.className = "slv-admin-utable";

	const thead = document.createElement("thead");
	thead.innerHTML =
		`<tr>` +
		`<th>${escHtml(t("admin_col_username"))}</th>` +
		`<th>${escHtml(t("admin_col_display"))}</th>` +
		`<th>${escHtml(t("admin_col_admin"))}</th>` +
		`<th></th>` +
		`</tr>`;
	table.appendChild(thead);

	const tbody = document.createElement("tbody");
	for (const u of users) {
		const tr = document.createElement("tr");

		const adminBadge = u.isAdmin
			? `<span class="slv-admin-badge slv-admin-badge-sm">${escHtml(t("admin_badge"))}</span>`
			: `<span class="slv-admin-no">—</span>`;

		tr.innerHTML =
			`<td class="slv-admin-utd-name">${escHtml(u.username)}</td>` +
			`<td class="slv-admin-utd-dn">${escHtml(u.displayName)}</td>` +
			`<td class="slv-admin-utd-role">${adminBadge}</td>` +
			`<td></td>`;

		const delBtn = document.createElement("button");
		delBtn.className = "slv-btn slv-admin-del-btn";
		delBtn.textContent = t("admin_delete_user");
		delBtn.addEventListener("click", () => deleteUser(u.username, tr));
		tr.querySelector("td:last-child").appendChild(delBtn);

		tbody.appendChild(tr);
	}
	table.appendChild(tbody);

	els.userList.innerHTML = "";
	els.userList.appendChild(table);
}

/** Reads the create-user form, validates it, and submits a POST request. */
async function createUser() {
	setMsg(null, "");

	const username = els.auUsername.value.trim();
	const password = els.auPassword.value.trim();
	const displayName = els.auDisplay.value.trim();
	const isAdmin = els.auAdmin.checked;

	if (!username) {
		setMsg("err", t("admin_err_username_required"));
		return;
	}
	if (!password) {
		setMsg("err", t("admin_err_password_required"));
		return;
	}

	els.auSubmit.disabled = true;
	try {
		await apiFetch("/admin/users", {
			method: "POST",
			headers: { "Content-Type": "application/json" },
			body: JSON.stringify({ username, password, displayName, isAdmin }),
		});
		setMsg("ok", t("admin_user_created", { u: username }));
		els.auUsername.value = "";
		els.auPassword.value = "";
		els.auDisplay.value = "";
		els.auAdmin.checked = false;
		await loadUsers();
	} catch (e) {
		setMsg("err", e.message);
	} finally {
		els.auSubmit.disabled = false;
	}
}

/** Sends a DELETE request for the given username and removes its row on success. */
async function deleteUser(username, rowEl) {
	if (!confirm(t("admin_confirm_delete", { u: username }))) return;

	rowEl.style.opacity = "0.4";
	try {
		await apiFetch(`/admin/users/${encodeURIComponent(username)}`, {
			method: "DELETE",
		});
		rowEl.remove();
	} catch (e) {
		rowEl.style.opacity = "";
		alert(e.message);
	}
}

/** Binds all user-tab event listeners. Call once after the panel is in the DOM. */
export function initUsers() {
	els.auSubmit.addEventListener("click", createUser);
	els.auRefresh.addEventListener("click", loadUsers);
	[els.auUsername, els.auPassword, els.auDisplay].forEach((inp) =>
		inp.addEventListener("keydown", (e) => {
			if (e.key === "Enter") createUser();
		}),
	);
}
