import type { Screen } from "../screen";
import { Benchmark } from "../../game/bench/Benchmark";

/** Dev-only screen for the phase 0 rendering / transport benchmark. */
export function createBenchmarkScreen(onExit: () => void): Screen {
    let container: HTMLElement | null = null;
    let bench: Benchmark | null = null;

    const onKey = (e: KeyboardEvent) => {
        if (e.key === "Escape") onExit();
    };

    return {
        mount(root) {
            container = document.createElement("div");
            container.className = "screen bench-screen";
            root.appendChild(container);

            bench = new Benchmark(container);
            bench.start();
            window.addEventListener("keydown", onKey);
        },

        unmount() {
            window.removeEventListener("keydown", onKey);
            bench?.stop();
            bench = null;
            container?.remove();
            container = null;
        },
    };
}
