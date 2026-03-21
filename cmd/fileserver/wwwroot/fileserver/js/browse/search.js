import { t } from "/js/components/i18n.js";
import { escHtml } from "/js/browse/table.js";

const SK = location.pathname;

/** Parses a raw search string into a base query and optional ext: filter. */
export function parseQueryTags(raw) {
    const tokens = raw.trim().split(/\s+/);
    const rest = [];
    let ext = null;

    for (const tok of tokens) {
        const lower = tok.toLowerCase();
        let val = null;
        if (lower.startsWith("extension:")) val = lower.slice("extension:".length);
        else if (lower.startsWith("ext:")) val = lower.slice("ext:".length);

        if (val !== null && val !== "") {
            ext = val.startsWith(".") ? val : `.${val}`;
        } else {
            rest.push(tok);
        }
    }

    return { base: rest.join(" "), ext };
}

/** Renders or clears the active extension filter chip. */
export function updateTagHint(parsed) {
    const hint = document.getElementById("slv-tag-hint");
    const input = document.getElementById("slv-search-input");
    if (!hint) return;

    if (!parsed.ext) {
        hint.innerHTML = "";
        hint.style.display = "none";
        return;
    }

    hint.style.display = "flex";
    hint.innerHTML =
        `<span class="slv-tag-chip">${escHtml(t("ext_filter_label"))} <strong>${escHtml(parsed.ext)}</strong>`
        + `<button class="slv-tag-remove" title="${escHtml(t("ext_filter_remove_title"))}">\xd7</button></span>`;

    hint.querySelector(".slv-tag-remove").addEventListener("click", () => {
        input.value = input.value
            .replace(/\b(?:ext|extension):\S*/gi, "")
            .replace(/\s+/g, " ")
            .trim();
        input.dispatchEvent(new Event("input"));
        input.focus();
    });
}

/** Binds the search input, content toggle, and keyboard shortcuts for the given browse route. */
export function initSearch(route) {
    const input = document.getElementById("slv-search-input");
    const results = document.getElementById("slv-search-results");
    const toggle = document.getElementById("slv-content-toggle");

    let activeIdx = -1;
    let inContent = false;
    let timer = null;

    const closeResults = () => {
        results.classList.remove("open");
        results.innerHTML = "";
        activeIdx = -1;
    };

    const setActive = idx => {
        const items = results.querySelectorAll(".slv-search-item");
        items[activeIdx]?.classList.remove("slv-search-item-active");
        activeIdx = idx;
        const el = items[activeIdx];
        if (el) {
            el.classList.add("slv-search-item-active");
            el.scrollIntoView({ block: "nearest" });
        }
    };

    const doSearch = async raw => {
        activeIdx = -1;
        const parsed = parseQueryTags(raw);
        updateTagHint(parsed);
        if (!parsed.base && !parsed.ext) { closeResults(); return; }

        let url = `${route}?search=${encodeURIComponent(raw)}`;
        if (inContent) url += "&content=1";

        try {
            const r = await fetch(url);
            const items = await r.json();

            if (!items?.length) {
                const label = parsed.base || `*${parsed.ext ?? ""}`;
                results.innerHTML = `<div class="slv-search-empty">${escHtml(t("search_no_results", { q: label }))}</div>`;
            } else {
                results.innerHTML = items.map(item => {
                    const badge = item.matchType === "content"
                        ? `<span class="slv-match-badge">${escHtml(t("match_in_file"))}</span>`
                        : "";
                    const snippet = item.snippet
                        ? `<span class="slv-search-snippet">${escHtml(item.snippet)}</span>`
                        : "";
                    return `<a class="slv-search-item" href="${escHtml(item.highlightUrl)}">`
                        + `<div class="slv-search-item-top"><span class="slv-search-item-name">${escHtml(item.name)}</span>${badge}</div>`
                        + `<span class="slv-search-item-path">${escHtml(item.relPath)}</span>`
                        + snippet
                        + `</a>`;
                }).join("");
            }
            results.classList.add("open");
        } catch {
            closeResults();
        }
    };

    const savedQ = sessionStorage.getItem(`fs_q_${SK}`);
    if (savedQ) {
        input.value = savedQ;
        updateTagHint(parseQueryTags(savedQ));
    }

    input.addEventListener("input", () => {
        clearTimeout(timer);
        const q = input.value.trim();
        sessionStorage.setItem(`fs_q_${SK}`, q);
        if (!q) { closeResults(); updateTagHint({ ext: null }); return; }
        timer = setTimeout(() => doSearch(q), 220);
    });

    toggle.addEventListener("click", () => {
        inContent = !inContent;
        toggle.classList.toggle("active", inContent);
        const q = input.value.trim();
        if (q) doSearch(q);
    });

    input.addEventListener("keydown", e => {
        const items = results.querySelectorAll(".slv-search-item");
        switch (e.key) {
            case "Escape":
                closeResults();
                input.value = "";
                updateTagHint({ ext: null });
                sessionStorage.removeItem(`fs_q_${SK}`);
                break;
            case "ArrowDown":
                e.preventDefault();
                if (results.classList.contains("open"))
                    setActive(Math.min(activeIdx + 1, items.length - 1));
                break;
            case "ArrowUp":
                e.preventDefault();
                if (results.classList.contains("open"))
                    setActive(Math.max(activeIdx - 1, 0));
                break;
            case "Enter":
                if (activeIdx >= 0 && activeIdx < items.length) {
                    e.preventDefault();
                    items[activeIdx].click();
                }
                break;
        }
    });

    document.addEventListener("click", e => {
        if (![input, results, toggle, document.getElementById("slv-tag-hint")]
            .some(el => el?.contains(e.target))) {
            closeResults();
        }
    });

    document.addEventListener("keydown", e => {
        if (e.key === "/" &&
            document.activeElement !== input &&
            !["INPUT", "TEXTAREA", "SELECT"].includes(document.activeElement.tagName)) {
            e.preventDefault();
            input.focus();
        }
    });
}