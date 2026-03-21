const src = document.getElementById("slv-readme-src");
const render = document.getElementById("slv-readme-render");

if (src && render && typeof marked !== "undefined") {
    render.innerHTML = marked.parse(src.textContent);
}