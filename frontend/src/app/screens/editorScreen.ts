import type { Screen } from "../screen";

export function createEditorScreen(onExit: () => void): Screen {
    let el: HTMLElement | null = null;

    const onKey = (e: KeyboardEvent) => {
        if (e.key === "Escape") onExit();
    };

    return {
        mount(root) {
            el = document.createElement("div");
            el.className = "screen editor-screen";
            el.innerHTML = `
                <div class="placeholder">
                    <p>Editor — coming soon</p>
                    <button class="menu-btn" data-action="back">Back</button>
                </div>
            `;
            el.querySelector("[data-action=back]")!.addEventListener("click", onExit);
            window.addEventListener("keydown", onKey);
            root.appendChild(el);
        },

        unmount() {
            window.removeEventListener("keydown", onKey);
            el?.remove();
            el = null;
        },
    };
}
