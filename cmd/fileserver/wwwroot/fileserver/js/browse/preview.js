import { t } from "/js/components/i18n.js";
import { escHtml } from "/js/browse/table.js";

/** Initialises the preview modal and returns a showPreview(entry) function. */
export function initPreview(imageExts, textExts) {
    const modal = document.getElementById("slv-preview-modal");
    const close = document.getElementById("slv-preview-close");
    const backdrop = document.getElementById("slv-preview-backdrop");
    const title = document.getElementById("slv-preview-title");
    const dl = document.getElementById("slv-preview-dl");
    const body = document.getElementById("slv-preview-body");

    const closePreview = () => {
        modal.classList.remove("open");
        body.innerHTML = "";
        document.body.style.overflow = "";
    };

    close.addEventListener("click", closePreview);
    backdrop.addEventListener("click", closePreview);
    document.addEventListener("keydown", e => {
        if (e.key === "Escape" && modal.classList.contains("open")) closePreview();
    });

    return async function showPreview(entry) {
        title.textContent = entry.name;
        dl.href = entry.downloadUrl;
        body.innerHTML = "";
        modal.classList.add("open");
        document.body.style.overflow = "hidden";

        if (imageExts[entry.ext]) {
            const img = Object.assign(document.createElement("img"), {
                src: entry.previewUrl,
                className: "slv-preview-img",
            });
            body.appendChild(img);
        } else if (entry.ext === ".pdf") {
            const iframe = Object.assign(document.createElement("iframe"), {
                src: entry.previewUrl,
                className: "slv-preview-iframe",
            });
            body.appendChild(iframe);
        } else if (textExts[entry.ext]) {
            body.innerHTML = `<div class="slv-preview-loading">${escHtml(t("preview_loading"))}</div>`;
            try {
                const r = await fetch(entry.previewUrl);
                const text = await r.text();
                body.innerHTML = "";
                const pre = Object.assign(document.createElement("pre"), {
                    className: "slv-preview-text",
                    textContent: text,
                });
                body.appendChild(pre);
            } catch {
                body.innerHTML = `<div class="slv-preview-error">${escHtml(t("preview_failed"))}</div>`;
            }
        }
    };
}