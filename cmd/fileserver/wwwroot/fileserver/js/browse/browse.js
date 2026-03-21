import {
    t, detectLang, loadLocale, setStrings,
    applyI18n, buildLangSelector, whenReady,
} from "/js/components/i18n.js";
import { renderTable, initSortHeaders } from "/js/browse/table.js";
import { initSearch, parseQueryTags, updateTagHint } from "/js/browse/search.js";
import { initPreview } from "/js/browse/preview.js";
import "/js/userbar.js";

const CFG = window.FS_CONFIG ?? {};
const ROUTE = CFG.route ?? "";
const ENTRIES = CFG.entries ?? [];
const IMAGE_EXTS = CFG.imageExts ?? {};
const TEXT_EXTS = CFG.textExts ?? {};

const showPreview = initPreview(IMAGE_EXTS, TEXT_EXTS);
const render = hl => renderTable(ENTRIES, IMAGE_EXTS, TEXT_EXTS, hl, showPreview);

initSortHeaders(ENTRIES, IMAGE_EXTS, TEXT_EXTS, showPreview);
initSearch(ROUTE);

const params = new URLSearchParams(window.location.search);
const hl = params.get("highlight") ?? null;
const savedQ = sessionStorage.getItem(`fs_q_${location.pathname}`);

await whenReady(() => {
    render(hl);
    applyI18n(CFG);

    buildLangSelector(async lang => {
        localStorage.setItem("fs_lang", lang);
        const data = await loadLocale(lang);
        setStrings(data);
        applyI18n(CFG);
        render(null);
    });

    if (savedQ) updateTagHint(parseQueryTags(savedQ));

    const lang = detectLang();
    if (lang !== "en") {
        loadLocale(lang).then(data => {
            if (Object.keys(data).length) {
                setStrings(data);
                applyI18n(CFG);
                render(hl);
            }
        });
    }
});