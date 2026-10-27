import "./ui/ui.css";
import "./ui/hud.css";
import { ScreenManager } from "./app/screen";
import { createMainMenu } from "./app/screens/mainMenu";
import { createGameScreen } from "./app/screens/gameScreen";
import { createEditorScreen } from "./app/screens/editorScreen";
import { createOptionsScreen } from "./app/screens/optionsScreen";
import { createBenchmarkScreen } from "./app/screens/benchmarkScreen";

// Esc belongs to the game. Without preventDefault the webview passes the unhandled
// key up to the native window, and macOS uses it to leave fullscreen.
window.addEventListener(
    "keydown",
    (e) => {
        if (e.key === "Escape") e.preventDefault();
    },
    { capture: true },
);

// Right click is a game command. Wails always shows the webview menu (Reload / Inspect)
// in dev builds, so block it ourselves. Hold Alt/Option to get it back for debugging.
window.addEventListener("contextmenu", (e) => {
    if (!e.altKey) e.preventDefault();
});

const screens = new ScreenManager(document.getElementById("app")!);

const showMenu = (): void =>
    screens.show(() =>
        createMainMenu({
            onGame: () => screens.show(() => createGameScreen(showMenu)),
            onEditor: () => screens.show(() => createEditorScreen(showMenu)),
            onOptions: () => screens.show(() => createOptionsScreen(showMenu)),
            onBenchmark: () => screens.show(() => createBenchmarkScreen(showMenu)),
        }),
    );

showMenu();
