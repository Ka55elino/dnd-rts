import type { Screen } from "../screen";
import { createComingSoonScreen } from "./comingSoonScreen";

export function createEditorScreen(onExit: () => void): Screen {
    return createComingSoonScreen("Editor", onExit);
}
