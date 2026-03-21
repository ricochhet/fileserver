const LOCALE_BASE = "/js/locales/";

let fallback = {};
let strings = {};

const ready = fetch(`${LOCALE_BASE}en.json`)
    .then(r => r.ok ? r.json() : {})
    .catch(() => ({}))
    .then(data => {
        fallback = data;
        strings = { ...data };
    });

/** Returns the translated string for key, substituting any vars. */
export function t(key, vars) {
    let s = strings[key] ?? fallback[key] ?? key;
    if (vars) {
        for (const [k, v] of Object.entries(vars)) {
            s = s.replaceAll(`{${k}}`, v);
        }
    }
    return s;
}

/** Returns the active locale code from localStorage or navigator. */
export function detectLang() {
    const saved = localStorage.getItem("fs_lang");
    const locales = window.FS_LOCALES ?? { en: "English" };
    if (saved && locales[saved]) return saved;
    const nav = (navigator.language ?? "en").toLowerCase().split("-")[0];
    return locales[nav] ? nav : "en";
}

/** Fetches and returns locale data for the given language code. */
export async function loadLocale(lang) {
    if (lang === "en") return fallback;
    try {
        const r = await fetch(`${LOCALE_BASE}${lang}.json`);
        if (!r.ok) throw new Error("not found");
        return await r.json();
    } catch {
        return {};
    }
}

/** Merges data into the active string table, falling back to en for missing keys. */
export function setStrings(data) {
    strings = Object.keys(data).length ? { ...fallback, ...data } : { ...fallback };
}

/** Calls fn after the default locale has loaded. */
export async function whenReady(fn) {
    await ready;
    fn();
}

/** Applies i18n strings to all data-i18n elements in the document. */
export function applyI18n(cfg) {
    for (const el of document.querySelectorAll("[data-i18n]")) {
        el.textContent = t(el.dataset.i18n);
    }
    for (const el of document.querySelectorAll("[data-i18n-placeholder]")) {
        el.placeholder = t(el.dataset.i18nPlaceholder);
    }
    for (const el of document.querySelectorAll("[data-i18n-title]")) {
        el.title = t(el.dataset.i18nTitle);
    }
    for (const th of document.querySelectorAll("th[data-sort]")) {
        th.title = t("sort_click_title");
    }

    const meta = document.getElementById("slv-dir-meta");
    if (meta) {
        const n = cfg.fileCount;
        if (n > 0) {
            let label = t(n === 1 ? "files_count_one" : "files_count_many", { n });
            if (cfg.totalSize) label += `, ${cfg.totalSize}`;
            meta.textContent = label;
        } else {
            meta.textContent = "";
        }
    }
}

/** Populates the language selector and calls onchange when the user picks a language. */
export function buildLangSelector(onchange) {
    const sel = document.getElementById("slv-lang-select");
    if (!sel) return;

    const locales = window.FS_LOCALES ?? { en: "English" };
    const current = detectLang();
    sel.innerHTML = "";

    for (const [code, label] of Object.entries(locales)) {
        const opt = document.createElement("option");
        opt.value = code;
        opt.textContent = label;
        opt.selected = code === current;
        sel.appendChild(opt);
    }

    sel.addEventListener("change", () => onchange(sel.value));
}