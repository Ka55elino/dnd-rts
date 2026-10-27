/**
 * Developer console: toggled with F2 or the key left of 1 (any layout).
 * Lines go to the Go session (internal/devconsole), which parses them and
 * answers with text. Client-only commands: clear.
 *
 * Tab completes command names, content ids and player names; ↑/↓ walk the
 * history (kept for the app session).
 */

/** Mirror of internal/devconsole command names (for completion only). */
const COMMANDS = ["help", "spawn", "build", "kill", "heal", "list", "hit", "xp", "control", "give", "res", "pause", "resume", "step", "speed", "hash", "clear"];
const PLAYERS = ["me", "enemy", "neutral"];
const MAX_LINES = 400;
/**
 * Toggle keys. The key left of 1 reports "Backquote" on ANSI keyboards but
 * "IntlBackslash" on Mac ISO keyboards (Italian, German, ...), where macOS
 * swaps it with the key next to left Shift. F2 works everywhere.
 */
const TOGGLE_KEYS = new Set(["F2", "Backquote", "IntlBackslash"]);
const history: string[] = []; // shared across games in this app session

export interface DevConsoleOptions {
    overlay: HTMLElement;
    send: (line: string) => void;
    /** Horde and building type ids, for completion. */
    hordeTypes: string[];
    buildingTypes: string[];
    /** Stock resource ids, for completion. */
    resources?: string[];
}

export class DevConsole {
    private readonly el: HTMLDivElement;
    private readonly log: HTMLDivElement;
    private readonly input: HTMLInputElement;
    private historyPos = history.length;

    constructor(private readonly opts: DevConsoleOptions) {
        this.el = document.createElement("div");
        this.el.className = "dev-console";
        this.el.hidden = true;
        this.el.innerHTML = `
            <div class="dev-console__log"></div>
            <div class="dev-console__row">
                <span class="dev-console__prompt">&gt;</span>
                <input class="dev-console__input" spellcheck="false" autocomplete="off" />
            </div>
        `;
        this.log = this.el.querySelector(".dev-console__log")!;
        this.input = this.el.querySelector(".dev-console__input")!;
        this.input.addEventListener("keydown", this.onInputKey);
        // Clicks inside the console must not select units or move the camera.
        this.el.addEventListener("mousedown", (e) => e.stopPropagation());
        opts.overlay.appendChild(this.el);
        window.addEventListener("keydown", this.onGlobalKey, true);
        this.print("dev console — type help, Tab completes, F2 / Esc closes", "dim");
    }

    get open(): boolean {
        return !this.el.hidden;
    }

    toggle(open = !this.open): void {
        this.el.hidden = !open;
        if (open) {
            this.input.focus();
        } else {
            this.input.blur();
        }
    }

    /** Server reply. */
    print(text: string, kind: "out" | "error" | "dim" | "cmd" = "out"): void {
        for (const line of text.split("\n")) {
            const div = document.createElement("div");
            div.className = `dev-console__line is-${kind}`;
            div.textContent = line;
            this.log.appendChild(div);
        }
        while (this.log.childElementCount > MAX_LINES) this.log.firstElementChild!.remove();
        this.log.scrollTop = this.log.scrollHeight;
    }

    dispose(): void {
        window.removeEventListener("keydown", this.onGlobalKey, true);
        this.el.remove();
    }

    private onGlobalKey = (e: KeyboardEvent): void => {
        if (!TOGGLE_KEYS.has(e.code)) return;
        e.preventDefault();
        e.stopPropagation();
        this.toggle();
    };

    private onInputKey = (e: KeyboardEvent): void => {
        e.stopPropagation(); // typing must not drive the game
        switch (e.key) {
            case "Enter": {
                const line = this.input.value.trim();
                this.input.value = "";
                if (!line) return;
                if (history[history.length - 1] !== line) history.push(line);
                this.historyPos = history.length;
                this.print(`> ${line}`, "cmd");
                if (line === "clear") this.log.replaceChildren();
                else this.opts.send(line);
                return;
            }
            case "Escape":
                e.preventDefault();
                this.toggle(false);
                return;
            case "ArrowUp":
                e.preventDefault();
                this.historyPos = Math.max(0, this.historyPos - 1);
                this.input.value = history[this.historyPos] ?? "";
                return;
            case "ArrowDown":
                e.preventDefault();
                this.historyPos = Math.min(history.length, this.historyPos + 1);
                this.input.value = history[this.historyPos] ?? "";
                return;
            case "Tab":
                e.preventDefault();
                this.complete();
                return;
        }
    };

    /** Completes the last word; with several matches extends to their common prefix and lists them. */
    private complete(): void {
        const value = this.input.value;
        const words = value.split(/\s+/);
        const word = words[words.length - 1];
        const candidates = this.candidates(words.length - 1, words[0]);
        const matches = candidates.filter((c) => c.startsWith(word));
        if (matches.length === 0) return;
        let completion = matches[0];
        if (matches.length > 1) {
            completion = matches.reduce((a, b) => {
                let i = 0;
                while (i < a.length && a[i] === b[i]) i++;
                return a.slice(0, i);
            });
            this.print(matches.join("  "), "dim");
        }
        words[words.length - 1] = completion;
        this.input.value = words.join(" ") + (matches.length === 1 ? " " : "");
    }

    private candidates(index: number, command: string): string[] {
        if (index === 0) return COMMANDS;
        switch (command) {
            case "spawn":
                return index === 1 ? this.opts.hordeTypes : index === 3 ? PLAYERS : [];
            case "build":
                return index === 1 ? this.opts.buildingTypes : index === 2 ? PLAYERS : [];
            case "kill":
            case "heal":
                return index === 1 ? ["all"] : index === 2 ? PLAYERS : [];
            case "list":
                return index === 1 ? ["hordes", "buildings", "types"] : [];
            case "help":
                return index === 1 ? COMMANDS : [];
            case "hit":
                return index === 1 ? ["fire", "ice", "lightning", "acid", "poison", "necrotic", "radiant", "pierce", "slash", "crush"] : [];
            case "control":
            case "res":
                return index === 1 ? PLAYERS.filter((p) => p !== "neutral") : [];
            case "give":
                return index === 1 ? (this.opts.resources ?? []) : index === 3 ? PLAYERS : [];
        }
        return [];
    }
}
