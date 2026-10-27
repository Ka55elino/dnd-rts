import { Application } from "@wailsio/runtime";
import type { Screen } from "../screen";

export interface MainMenuActions {
    onGame(): void;
    onEditor(): void;
    onOptions(): void;
    onBenchmark(): void;
}

export function createMainMenu(actions: MainMenuActions): Screen {
    let el: HTMLElement | null = null;

    return {
        mount(root) {
            el = document.createElement("div");
            el.className = "screen main-menu";
            el.innerHTML = `
                <h1 class="main-menu__title">dnd-rts</h1>
                <nav class="main-menu__items">
                    <button class="menu-btn" data-action="game">Game</button>
                    <button class="menu-btn" data-action="editor">Editor</button>
                    <button class="menu-btn" data-action="options">Options</button>
                    <button class="menu-btn menu-btn--dev" data-action="benchmark">Benchmark</button>
                    <button class="menu-btn" data-action="exit">Exit</button>
                </nav>
            `;

            el.addEventListener("click", (e) => {
                const action = (e.target as HTMLElement).closest<HTMLElement>("[data-action]")?.dataset.action;
                switch (action) {
                    case "game":
                        actions.onGame();
                        break;
                    case "editor":
                        actions.onEditor();
                        break;
                    case "options":
                        actions.onOptions();
                        break;
                    case "benchmark":
                        actions.onBenchmark();
                        break;
                    case "exit":
                        void Application.Quit();
                        break;
                }
            });

            root.appendChild(el);
        },

        unmount() {
            el?.remove();
            el = null;
        },
    };
}
