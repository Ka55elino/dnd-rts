import type { Screen } from "../screen";
import { Game } from "../../game/Game";
import { ContentService } from "../../../bindings/dnd-rts/internal/services";
import { ModePlayground } from "../../net/protocol";

/**
 * Playground: the game on a test field — each player gets one finished building of every type and one
 * horde of every type of its faction (internal/sim SetupPlayground). Two selects choose the factions;
 * changing one restarts the field with the new bases and armies.
 */
export function createPlaygroundScreen(onExit: () => void): Screen {
    let container: HTMLElement | null = null;
    let bar: HTMLElement | null = null;
    let game: Game | null = null;
    const factions = ["empire", "warband"];
    let fogOn = false;

    const onKey = (e: KeyboardEvent) => {
        if ((e.target as HTMLElement).closest?.("select")) return;
        if (e.key === "Escape" && !game?.handleEscape()) onExit();
    };

    const restart = () => {
        if (!container) return;
        game?.stop();
        container.querySelectorAll(":scope > :not(.playground-bar)").forEach((n) => n.remove());
        game = new Game(container);
        game.start(undefined, { mode: ModePlayground, factions: [...factions] });
        fogOn = false;
        syncFog();
    };

    const syncFog = () => {
        const b = bar?.querySelector<HTMLButtonElement>("[data-fog]");
        if (b) b.classList.toggle("is-active", fogOn);
    };

    const fillSelects = async () => {
        let list: { id: string; name: string }[] = [];
        try {
            list = (await ContentService.Factions()) ?? [];
        } catch (err) {
            console.warn("factions:", err);
        }
        if (!list.length) list = factions.map((id) => ({ id, name: id }));
        bar?.querySelectorAll<HTMLSelectElement>("select[data-player]").forEach((sel) => {
            const p = Number(sel.dataset.player);
            sel.innerHTML = list.map((f) => `<option value="${f.id}">${f.name}</option>`).join("");
            if (!list.some((f) => f.id === factions[p])) factions[p] = list[Math.min(p, list.length - 1)].id;
            sel.value = factions[p];
        });
    };

    return {
        mount(root) {
            container = document.createElement("div");
            container.className = "screen game-screen";
            root.appendChild(container);

            bar = document.createElement("div");
            bar.className = "playground-bar";
            bar.innerHTML = `
                <span class="playground-bar__title">Playground</span>
                <label>You <select data-player="0"></select></label>
                <label>Enemy <select data-player="1"></select></label>
                <button class="segmented__btn" data-fog title="fog of war on / off">Fog</button>
                <button class="segmented__btn" data-reset title="rebuild the field">Reset</button>
            `;
            bar.addEventListener("change", (e) => {
                const sel = e.target as HTMLSelectElement;
                if (sel.dataset.player === undefined) return;
                factions[Number(sel.dataset.player)] = sel.value;
                sel.blur();
                restart();
            });
            bar.querySelector("[data-reset]")!.addEventListener("click", restart);
            bar.querySelector("[data-fog]")!.addEventListener("click", (e) => {
                fogOn = !fogOn;
                game?.devCommand(fogOn ? "fog on" : "fog off");
                syncFog();
                (e.currentTarget as HTMLElement).blur();
            });
            container.appendChild(bar);

            void fillSelects().then(restart);
            window.addEventListener("keydown", onKey);
        },

        unmount() {
            window.removeEventListener("keydown", onKey);
            game?.stop();
            game = null;
            container?.remove();
            container = null;
            bar = null;
        },
    };
}
