import { t } from "/js/components/i18n.js";

const SK = location.pathname;

export let sortCol = sessionStorage.getItem(`fs_sc_${SK}`) ?? "name";
export let sortDir = sessionStorage.getItem(`fs_sd_${SK}`) ?? "asc";

/** Escapes a string for safe HTML insertion. */
export function escHtml(s) {
    return String(s)
        .replaceAll("&", "&amp;")
        .replaceAll("<", "&lt;")
        .replaceAll(">", "&gt;")
        .replaceAll('"', "&quot;");
}

export function setSortCol(col) { sortCol = col; }
export function setSortDir(dir) { sortDir = dir; }

/** Returns a sorted copy of entries, dirs first, respecting sortCol and sortDir. */
export function sortedEntries(entries) {
    return [...entries].sort((a, b) => {
        if (a.isDir !== b.isDir) return a.isDir ? -1 : 1;

        let va, vb;
        if (sortCol === "size") { va = a.sizeBytes; vb = b.sizeBytes; }
        else if (sortCol === "modified") { va = a.modUnix; vb = b.modUnix; }
        else { va = a.name.toLowerCase(); vb = b.name.toLowerCase(); }

        if (va < vb) return sortDir === "asc" ? -1 : 1;
        if (va > vb) return sortDir === "asc" ? 1 : -1;
        return 0;
    });
}

/** Updates sort arrow indicators on all sortable column headers. */
export function updateSortIndicators() {
    for (const th of document.querySelectorAll("th[data-sort]")) {
        th.classList.remove("slv-sort-asc", "slv-sort-desc");
        if (th.dataset.sort === sortCol) {
            th.classList.add(sortDir === "asc" ? "slv-sort-asc" : "slv-sort-desc");
        }
    }
}

function makeRow(e, imageExts, textExts, onPreview) {
    const previewable = imageExts[e.ext] || textExts[e.ext] || e.ext === ".pdf";

    const icon = Object.assign(document.createElement("span"), {
        className: "slv-icon",
        textContent: e.isDir ? "\uD83D\uDCC1" : "\uD83D\uDCC4",
    });

    const link = Object.assign(document.createElement("a"), { textContent: e.name });
    if (e.isDir) {
        link.href = e.browseUrl;
    } else {
        link.href = e.downloadUrl;
        if (previewable) {
            link.addEventListener("click", ev => { ev.preventDefault(); onPreview(e); });
        }
    }

    const nameDiv = Object.assign(document.createElement("div"), { className: "slv-name" });
    nameDiv.append(icon, link);

    const td1 = document.createElement("td");
    td1.appendChild(nameDiv);

    const td2 = Object.assign(document.createElement("td"), { className: "slv-meta", textContent: e.sizeStr });
    const td3 = Object.assign(document.createElement("td"), { className: "slv-meta", textContent: e.modStr });

    const actions = Object.assign(document.createElement("div"), { className: "slv-actions" });

    const dlBtn = Object.assign(document.createElement("a"), {
        className: "slv-btn",
        href: e.downloadUrl,
        textContent: e.isDir ? t("btn_zip") : t("btn_download"),
    });
    actions.appendChild(dlBtn);

    if (!e.isDir) {
        const infoBtn = Object.assign(document.createElement("a"), {
            className: "slv-btn secondary",
            href: e.infoUrl,
            textContent: t("btn_info"),
        });

        const copyBtn = Object.assign(document.createElement("button"), {
            className: "slv-btn secondary slv-copy-btn",
            textContent: t("btn_copy_url"),
        });
        copyBtn.addEventListener("click", async () => {
            const url = location.origin + e.downloadUrl;
            const reset = () => { copyBtn.textContent = t("btn_copy_url"); };
            try {
                await navigator.clipboard.writeText(url);
            } catch {
                // Fallback for browsers without the Clipboard API.
                const ta = Object.assign(document.createElement("textarea"), {
                    value: url,
                    style: "position:fixed;opacity:0",
                });
                document.body.append(ta);
                ta.select();
                document.execCommand("copy");
                ta.remove();
            }
            copyBtn.textContent = t("btn_copied");
            setTimeout(reset, 1500);
        });

        actions.append(infoBtn, copyBtn);
    }

    const td4 = document.createElement("td");
    td4.appendChild(actions);

    const tr = document.createElement("tr");
    tr.append(td1, td2, td3, td4);
    return tr;
}

/** Renders the directory entry table, optionally highlighting a named file. */
export function renderTable(entries, imageExts, textExts, hl, onPreview) {
    const tbody = document.getElementById("slv-tbody");
    const parentRow = tbody.querySelector(".slv-parent");
    tbody.innerHTML = "";
    if (parentRow) tbody.appendChild(parentRow);

    const sorted = sortedEntries(entries);

    for (const e of sorted) {
        const tr = makeRow(e, imageExts, textExts, onPreview);
        if (hl && e.name === hl) {
            tr.classList.add("slv-row-highlight");
            setTimeout(() => {
                tr.classList.remove("slv-row-highlight");
                tr.classList.add("slv-row-highlight-fade");
            }, 1500);
            setTimeout(() => tr.scrollIntoView({ behavior: "smooth", block: "center" }), 50);
        }
        tbody.appendChild(tr);
    }

    if (sorted.length === 0 && !parentRow) {
        const td = Object.assign(document.createElement("td"), {
            className: "slv-empty",
            textContent: t("empty_dir"),
        });
        td.setAttribute("colspan", "4");
        const row = document.createElement("tr");
        row.appendChild(td);
        tbody.appendChild(row);
    }

    updateSortIndicators();
}

/** Binds click handlers to all sortable column headers. */
export function initSortHeaders(entries, imageExts, textExts, onPreview) {
    for (const th of document.querySelectorAll("th[data-sort]")) {
        th.style.cursor = "pointer";
        th.addEventListener("click", () => {
            const col = th.dataset.sort;
            if (col === sortCol) {
                setSortDir(sortDir === "asc" ? "desc" : "asc");
            } else {
                setSortCol(col);
                setSortDir("asc");
            }
            sessionStorage.setItem(`fs_sc_${SK}`, sortCol);
            sessionStorage.setItem(`fs_sd_${SK}`, sortDir);
            renderTable(entries, imageExts, textExts, null, onPreview);
        });
    }
}