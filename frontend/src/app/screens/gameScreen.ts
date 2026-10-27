import type { Screen } from "../screen";
import { Game } from "../../game/Game";

export function createGameScreen(onExit: () => void): Screen {
    let container: HTMLElement | null = null;
    let game: Game | null = null;

    const onKey = (e: KeyboardEvent) => {
        // Esc clears the selection first; with nothing selected it returns to the menu
        // (later: opens the in-game pause menu).
        if (e.key === "Escape" && !game?.handleEscape()) onExit();
    };

    return {
        mount(root) {
            container = document.createElement("div");
            container.className = "screen game-screen";
            root.appendChild(container);

            game = new Game(container);
            game.start();

            window.addEventListener("keydown", onKey);
        },

        unmount() {
            window.removeEventListener("keydown", onKey);
            game?.stop();
            game = null;
            container?.remove();
            container = null;
        },
    };
}
