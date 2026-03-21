import {
    detectLang, loadLocale, setStrings,
    applyI18n, buildLangSelector, whenReady,
} from "/js/components/i18n.js";

await whenReady(() => {
    applyI18n({});

    buildLangSelector(async lang => {
        localStorage.setItem("fs_lang", lang);
        const data = await loadLocale(lang);
        setStrings(data);
        applyI18n({});
    });

    const lang = detectLang();
    if (lang !== "en") {
        loadLocale(lang).then(data => {
            if (Object.keys(data).length) {
                setStrings(data);
                applyI18n({});
            }
        });
    }
});