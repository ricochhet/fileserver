import {
	whenReady,
	applyI18n,
	buildLangSelector,
	detectLang,
	loadLocale,
	setStrings,
} from "/js/components/i18n.js";
import { renderTable, initSortHeaders } from "/js/browse/table.js";
import { initSearch } from "/js/browse/search.js";
import { initPreview } from "/js/browse/preview.js";

const cfg = window.FS_CONFIG;
const hl = new URLSearchParams(location.search).get("highlight");
const showPreview = initPreview(cfg.imageExts, cfg.textExts);

const applyAll = () =>
	applyI18n({ fileCount: cfg.fileCount, totalSize: cfg.totalSize });

await whenReady(() => {
	applyAll();

	buildLangSelector(async (lang) => {
		localStorage.setItem("fs_lang", lang);
		const data = await loadLocale(lang);
		setStrings(data);
		applyAll();
	});

	const lang = detectLang();
	if (lang !== "en") {
		loadLocale(lang).then((data) => {
			if (Object.keys(data).length) {
				setStrings(data);
				applyAll();
			}
		});
	}

	renderTable(cfg.entries, cfg.imageExts, cfg.textExts, hl, showPreview);
	initSortHeaders(cfg.entries, cfg.imageExts, cfg.textExts, showPreview);
	initSearch(cfg.route);
});
