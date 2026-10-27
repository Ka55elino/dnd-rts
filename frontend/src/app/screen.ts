/** A full-window screen: main menu, game, editor, ... */
export interface Screen {
    /** Build DOM / start systems inside `root`. */
    mount(root: HTMLElement): void;
    /** Tear everything down: DOM, listeners, WebGL resources. */
    unmount(): void;
}

export type ScreenFactory = () => Screen;

/** Shows exactly one screen at a time and guarantees the previous one is unmounted. */
export class ScreenManager {
    private current: Screen | null = null;

    constructor(private readonly root: HTMLElement) {}

    show(factory: ScreenFactory): void {
        this.current?.unmount();
        this.root.replaceChildren();
        this.current = factory();
        this.current.mount(this.root);
    }
}
