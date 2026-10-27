import type { Screen } from "../screen";
import { DisplayMode, SettingsService } from "../../../bindings/dnd-rts/internal/services";

const DISPLAY_MODES: { value: DisplayMode; label: string }[] = [
    { value: DisplayMode.DisplayFullscreen, label: "Fullscreen" },
    { value: DisplayMode.DisplayWindowed, label: "Windowed" },
];

export function createOptionsScreen(onBack: () => void): Screen {
    let el: HTMLElement | null = null;

    const onKey = (e: KeyboardEvent) => {
        if (e.key === "Escape") onBack();
    };

    // Fullscreen can also change via F11 / macOS green button: re-read on resize.
    const onResize = () => void refresh();

    async function refresh(): Promise<void> {
        const settings = await SettingsService.Get();
        el?.querySelectorAll<HTMLButtonElement>("[data-display]").forEach((btn) => {
            btn.classList.toggle("is-active", btn.dataset.display === settings.displayMode);
        });
    }

    async function setDisplayMode(mode: DisplayMode): Promise<void> {
        try {
            await SettingsService.SetDisplayMode(mode);
        } catch (err) {
            console.error("Failed to set display mode:", err);
        }
        await refresh();
    }

    return {
        mount(root) {
            el = document.createElement("div");
            el.className = "screen options-screen";
            el.innerHTML = `
                <h2 class="options__title">Options</h2>
                <div class="options__panel">
                    <div class="options__row">
                        <span class="options__label">Display mode</span>
                        <div class="segmented">
                            ${DISPLAY_MODES.map(
                                (m) => `<button class="segmented__btn" data-display="${m.value}">${m.label}</button>`,
                            ).join("")}
                        </div>
                    </div>
                </div>
                <button class="menu-btn options__back" data-action="back">Back</button>
            `;

            el.addEventListener("click", (e) => {
                const target = e.target as HTMLElement;
                const display = target.closest<HTMLElement>("[data-display]")?.dataset.display;
                if (display) void setDisplayMode(display as DisplayMode);
                if (target.closest("[data-action=back]")) onBack();
            });

            window.addEventListener("keydown", onKey);
            window.addEventListener("resize", onResize);
            root.appendChild(el);
            void refresh();
        },

        unmount() {
            window.removeEventListener("keydown", onKey);
            window.removeEventListener("resize", onResize);
            el?.remove();
            el = null;
        },
    };
}
