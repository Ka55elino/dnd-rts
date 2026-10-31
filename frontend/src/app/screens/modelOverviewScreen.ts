import type { Screen } from "../screen";
import { ModelOverview } from "../../game/overview/ModelOverview";

/** Look at any baked model and play its animations. */
export function createModelOverviewScreen(onExit: () => void): Screen {
    let container: HTMLElement | null = null;
    let overview: ModelOverview | null = null;

    const onKey = (e: KeyboardEvent) => {
        if (e.key === "Escape") onExit();
    };

    return {
        mount(root) {
            container = document.createElement("div");
            container.className = "screen bench-screen";
            root.appendChild(container);

            overview = new ModelOverview(container);
            overview.start();
            window.addEventListener("keydown", onKey);
        },

        unmount() {
            window.removeEventListener("keydown", onKey);
            overview?.stop();
            overview = null;
            container?.remove();
            container = null;
        },
    };
}
