import { t } from "/js/components/i18n.js";
import { els, escHtml, formatBytes } from "/js/admin/state.js";

let selectedFiles = [];

/** Shows an ok or error message in the upload form, hiding the other. */
function setMsg(type, msg) {
	els.upOk.classList.add("slv-hidden");
	els.upError.classList.add("slv-hidden");
	if (type === "ok") {
		els.upOk.textContent = msg;
		els.upOk.classList.remove("slv-hidden");
	}
	if (type === "err") {
		els.upError.textContent = msg;
		els.upError.classList.remove("slv-hidden");
	}
}

/** Re-renders the selected files list with per-item remove buttons. */
function renderFileList() {
	if (selectedFiles.length === 0) {
		els.upFilelist.innerHTML = "";
		return;
	}

	els.upFilelist.innerHTML = selectedFiles
		.map(
			(f, i) =>
				`<div class="slv-admin-file-item">` +
				`<span class="slv-admin-file-name">${escHtml(f.name)}</span>` +
				`<span class="slv-admin-file-size slv-meta">${formatBytes(f.size)}</span>` +
				`<button class="slv-admin-file-remove slv-btn secondary" data-idx="${i}">✕</button>` +
				`</div>`,
		)
		.join("");

	for (const btn of els.upFilelist.querySelectorAll(
		".slv-admin-file-remove",
	)) {
		btn.addEventListener("click", () => {
			selectedFiles.splice(Number(btn.dataset.idx), 1);
			renderFileList();
		});
	}
}

/** Adds files from a FileList to selectedFiles, skipping duplicates by name and size. */
function addFiles(fileList) {
	for (const f of fileList) {
		if (
			!selectedFiles.some((x) => x.name === f.name && x.size === f.size)
		) {
			selectedFiles.push(f);
		}
	}
	renderFileList();
}

/** Resets the upload form, file list, and progress bar to their initial state. */
function clearUpload() {
	selectedFiles = [];
	els.upInput.value = "";
	renderFileList();
	setMsg(null, "");
	els.upProgress.classList.add("slv-hidden");
	els.upBar.style.width = "0%";
}

/** Uploads all selected files via XHR with progress tracking and shows the result. */
async function doUpload() {
	setMsg(null, "");
	if (selectedFiles.length === 0) {
		setMsg("err", t("admin_err_no_files"));
		return;
	}

	const form = new FormData();
	const subPath = els.upPath.value.trim();
	if (subPath) form.append("path", subPath);
	for (const f of selectedFiles) form.append("file", f);

	els.upSubmit.disabled = true;
	els.upProgress.classList.remove("slv-hidden");
	els.upBar.style.width = "0%";

	try {
		await new Promise((resolve, reject) => {
			const xhr = new XMLHttpRequest();
			xhr.open("POST", "/admin/upload");
			xhr.upload.addEventListener("progress", (e) => {
				if (e.lengthComputable) {
					els.upBar.style.width = `${Math.round((e.loaded / e.total) * 100)}%`;
				}
			});
			xhr.addEventListener("load", () => {
				if (xhr.status >= 200 && xhr.status < 300) {
					resolve(xhr.responseText);
				} else {
					let msg = xhr.statusText;
					try {
						msg = JSON.parse(xhr.responseText).error || msg;
					} catch {
						/* noop */
					}
					reject(new Error(msg));
				}
			});
			xhr.addEventListener("error", () =>
				reject(new Error(t("admin_err_upload_network"))),
			);
			xhr.send(form);
		});

		els.upBar.style.width = "100%";
		setMsg("ok", t("admin_upload_ok", { n: selectedFiles.length }));
		clearUpload();
	} catch (e) {
		setMsg("err", e.message);
	} finally {
		els.upSubmit.disabled = false;
	}
}

/** Binds all upload-tab event listeners. Call once after the panel is in the DOM. */
export function initUpload() {
	els.upInput.addEventListener("change", () => {
		addFiles(els.upInput.files);
		els.upInput.value = "";
	});

	els.upDrop.addEventListener("dragover", (e) => {
		e.preventDefault();
		els.upDrop.classList.add("drag-over");
	});
	els.upDrop.addEventListener("dragleave", () =>
		els.upDrop.classList.remove("drag-over"),
	);
	els.upDrop.addEventListener("drop", (e) => {
		e.preventDefault();
		els.upDrop.classList.remove("drag-over");
		addFiles(e.dataTransfer.files);
	});

	els.upSubmit.addEventListener("click", doUpload);
	els.upClear.addEventListener("click", clearUpload);
}
