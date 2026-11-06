"""High Elves: white stone and silver, greenery and turquoise, slender spires, trees inside buildings."""
from art.props._style import *  # noqa: F401,F403

WHITE = (0.90, 0.89, 0.85)
WHITE_D = (0.76, 0.76, 0.73)
SILVER = (0.78, 0.80, 0.84)
TURQ = (0.20, 0.62, 0.62)
TURQ_D = (0.14, 0.45, 0.48)
MOON = (0.62, 0.85, 1.0)
GREEN = (0.30, 0.55, 0.30)


def spire(x, z, r, h, roof_h=None, y=0.0, seg=10, glow=None):
    """A slender white tower with a tall turquoise cone and a silver ring under it."""
    out = cyl((x, y, z), r, h, r_top=r * 0.85, seg=seg, color=WHITE)
    out += cyl((x, y + h, z), r * 1.05, 0.25, seg=seg, color=SILVER, kind="metal", detail=1)
    out += cone((x, y + h + 0.25, z), r * 1.15, roof_h or r * 3.5, seg=seg, color=TURQ)
    if glow:
        out += sphere((x, y + h + 0.25 + (roof_h or r * 3.5) + 0.4, z), 0.35, seg=6, rings=3, color=glow, kind="glow", detail=1)
    return out


def elf_tree(x, z, h=5.0, y=0.0, detail=0):
    """A tall elven tree: a pale trunk and two stacked round crowns."""
    return (cyl((x, y, z), 0.22, h * 0.7, r_top=0.14, seg=6, color=(0.70, 0.66, 0.58), detail=detail) +
            sphere((x, y + h * 0.62, z), h * 0.26, seg=8, rings=4, color=GREEN, detail=detail) +
            sphere((x + 0.3, y + h * 0.88, z - 0.2), h * 0.18, seg=7, rings=3, color=LEAF_L, detail=max(detail, 1)))


def keep_high_elves():
    out = cyl((0, 0, 0), 6.8, 0.6, seg=16, color=WHITE_D)
    out += cyl((0, 0.6, 0), 6.2, 5.0, seg=16, color=WHITE)                         # round ward
    out += cyl((0, 5.6, 0), 6.4, 0.3, seg=16, color=SILVER, kind="metal", detail=1)
    out += cyl((0, 5.6, 0), 4.2, 6.0, r_top=3.4, seg=12, color=WHITE)                # hall
    out += dome((0, 11.6, 0), 3.6, h=2.6, seg=12, rings=4, color=TURQ)
    out += spire(0, 0, 1.3, 9.0, roof_h=6.0, y=12.8, glow=MOON)                     # central spire
    for k in range(4):
        import math
        a = math.pi / 4 + k * math.pi / 2
        out += spire(5.4 * math.cos(a), 5.4 * math.sin(a), 1.1, 9.5, roof_h=4.5, y=0.6)
    out += door(0, 6.2, w=2.2, h=2.8, y=0.6, color=TURQ_D)
    out += elf_tree(-3.0, 3.8, h=6.0, y=5.6, detail=1)
    out += elf_tree(3.2, -3.5, h=5.0, y=5.6, detail=1)
    for x in (-2.0, 2.0):
        out += banner((x, 4.8, 6.25), w=0.9, h=2.6, pole=False)
    out += windows([-2.5, 2.5], 8.0, 3.9, w=0.5, h=1.6, color=MOON, kind="glow")
    return out


def order_of_the_sword():
    out = box((0, 0, 0), (9.6, 0.4, 9.6), color=WHITE_D)
    out += hall(8.0, 5.0, 4.5, WHITE, TURQ, roof="hip", at=(0, 0.4, -1.8), roof_h=2.6)
    out += spire(-3.6, -1.8, 0.9, 6.5, y=0.4)
    out += spire(3.6, -1.8, 0.9, 6.5, y=0.4)
    out += door(0, 0.72, w=2.0, h=2.6, y=0.4, color=TURQ_D)
    for x in (-2.0, 2.0):                                                           # sword statues
        out += box((x, 0.4, 2.8), (0.8, 0.8, 0.8), color=WHITE_D, detail=1)
        out += box((x, 1.2, 2.8), (0.18, 2.4, 0.08), color=SILVER, kind="metal", detail=1)
        out += box((x, 2.0, 2.8), (0.9, 0.14, 0.14), color=SILVER, kind="metal", detail=2)
    out += elf_tree(0, 3.6, h=4.2, y=0.4)
    out += banner((-2.0, 4.2, 0.72), w=0.8, h=2.0, pole=False)
    out += banner((2.0, 4.2, 0.72), w=0.8, h=2.0, pole=False)
    return out


def order_of_archers():
    out = cyl((0, 0, -1.5), 4.4, 0.4, seg=12, color=WHITE_D)
    for k in range(8):                                                              # open pavilion
        import math
        a = 2 * math.pi * k / 8
        out += cyl((3.6 * math.cos(a), 0.4, -1.5 + 3.6 * math.sin(a)), 0.3, 5.0, seg=6, color=WHITE)
    out += cyl((0, 5.4, -1.5), 4.1, 0.6, seg=12, color=WHITE)
    out += dome((0, 6.0, -1.5), 4.1, h=3.0, seg=12, rings=4, color=TURQ)
    out += spire(0, -1.5, 0.5, 1.5, roof_h=2.0, y=8.8, glow=MOON)
    out += elf_tree(0, -1.5, h=4.6, y=0.4)
    for x in (-3.0, 0.0, 3.0):
        out += target(x, 4.2, ry=180)
    out += banner((0, 5.3, 2.2), w=1.0, h=1.8, pole=False)
    return out


def griffon_tower():
    out = cyl((0, 0, 0), 5.6, 0.5, seg=14, color=WHITE_D)
    out += cyl((0, 0.5, 0), 3.0, 14.0, r_top=2.4, seg=12, color=WHITE)
    out += cyl((0, 14.5, 0), 5.0, 0.6, seg=14, color=WHITE_D)                       # platform
    out += cyl((0, 15.1, 0), 4.6, 0.9, seg=14, r_top=4.9, color=(0.48, 0.38, 0.24))  # the nest
    for k in range(6):
        import math
        a = 2 * math.pi * k / 6
        out += cyl((4.4 * math.cos(a), 15.1, 4.4 * math.sin(a)), 0.18, 4.0, seg=5, color=SILVER, kind="metal", detail=1)
    out += cone((0, 19.1, 0), 5.4, 3.0, seg=14, color=TURQ)
    for k in range(3):
        import math
        a = 2 * math.pi * k / 3 + 0.4
        out += spire(4.4 * math.cos(a), 4.4 * math.sin(a), 0.8, 5.0, y=0.5)
    out += door(0, 2.85, w=1.6, h=2.2, y=0.5, color=TURQ_D)
    out += banner((0, 12.5, 2.5), w=1.0, h=2.6, pole=False)
    out += elf_tree(-3.6, 4.6, h=4.0)
    return out


def tiger_pen():
    out = []
    pts = circle(16, 4.6)
    for (x0, z0), (x1, z1) in zip(pts, pts[1:]):                                   # low curved wall, open front
        if z0 > 3.5 and abs(x0) < 2.2:
            continue
        mx, mz = (x0 + x1) / 2, (z0 + z1) / 2
        import math
        out += box((mx, 0, mz), (math.hypot(x1 - x0, z1 - z0) + 0.1, 1.4, 0.5), ry=-math.degrees(math.atan2(z1 - z0, x1 - x0)), color=WHITE)
    out += box((0, 0, 0), (8.6, 0.12, 8.6), color=(0.42, 0.55, 0.30))
    out += hall(4.4, 3.0, 2.6, WHITE, TURQ, roof="hip", at=(0, 0, -2.6), roof_h=1.8)  # shelter
    out += elf_tree(-2.8, 0.6, h=5.0)
    out += elf_tree(3.0, 1.2, h=4.4)
    out += spire(-2.6, 4.0, 0.45, 2.6, y=0.0)
    out += spire(2.6, 4.0, 0.45, 2.6, y=0.0)
    out += banner((0, 2.4, -1.08), w=0.8, h=1.4, pole=False)
    return out


def spirit_garden():
    out = cyl((0, 0, 0), 6.2, 0.4, seg=16, color=WHITE_D)
    out += glow_pool(0, 0, 2.6, MOON, WHITE, y=0.4, rim_h=0.5, seg=14)
    import math
    for k in range(10):                                                             # a ring of pillars
        a = 2 * math.pi * k / 10
        x, z = 5.4 * math.cos(a), 5.4 * math.sin(a)
        out += cyl((x, 0.4, z), 0.3, 4.2, seg=6, color=WHITE)
        out += box((x, 4.6, z), (0.8, 0.3, 0.8), color=SILVER, kind="metal", detail=1)
    for k, col in enumerate([(1.0, 0.45, 0.15), (0.35, 0.65, 1.0), (1.0, 0.92, 0.45)]):  # fire, water, storm
        a = 2 * math.pi * k / 3 + 0.5
        x, z = 3.6 * math.cos(a), 3.6 * math.sin(a)
        out += cyl((x, 0.4, z), 0.5, 1.0, seg=6, color=WHITE_D)
        out += cone((x, 1.4, z), 0.45, 1.4, seg=5, color=col, kind="glow")
        out += cone((x, 1.4, z), 0.45, -0.6, seg=5, color=col, kind="glow", detail=1)
    out += elf_tree(-2.6, -4.0, h=5.0, detail=1)
    out += elf_tree(3.0, -3.6, h=4.2, detail=1)
    out += elf_tree(0.4, 4.4, h=4.6, detail=1)
    return out


def enchantment_tower():
    out = cyl((0, 0, 0), 4.2, 0.4, seg=12, color=WHITE_D)
    out += spire(0, 0, 1.8, 11.0, roof_h=0.01, y=0.4)
    out += cyl((0, 11.6, 0), 2.6, 0.35, seg=12, color=SILVER, kind="metal")
    out += cone((0, 13.0, 0), 1.0, 1.8, seg=6, color=(0.4, 0.95, 0.85), kind="glow")   # floating crystal
    out += cone((0, 13.0, 0), 1.0, -1.2, seg=6, color=(0.4, 0.95, 0.85), kind="glow")
    for k in range(3):
        import math
        a = 2 * math.pi * k / 3
        out += rod((2.5 * math.cos(a), 11.9, 2.5 * math.sin(a)), (1.2 * math.cos(a), 14.6, 1.2 * math.sin(a)), 0.1, seg=4, color=SILVER, kind="metal", detail=1)
    out += hall(5.0, 3.2, 3.0, WHITE, TURQ, roof="hip", at=(0, 0.4, 2.4), roof_h=1.6)
    out += door(0, 4.02, w=1.4, h=1.8, y=0.4, color=TURQ_D)
    out += windows([0], 7.0, 1.6, w=0.4, h=1.4, color=MOON, kind="glow")
    out += elf_tree(-3.2, -2.6, h=4.0, detail=1)
    return out


def moonwell():
    out = cyl((0, 0, 0), 3.8, 0.3, seg=14, color=WHITE_D)
    out += glow_pool(0, 0, 2.2, MOON, WHITE, y=0.3, rim_h=0.8, seg=14)
    import math
    for k in range(4):                                                              # four arches
        a = math.pi / 4 + k * math.pi / 2
        x, z = 3.0 * math.cos(a), 3.0 * math.sin(a)
        out += cyl((x, 0.3, z), 0.28, 3.6, seg=6, color=WHITE)
        out += sphere((x, 4.1, z), 0.3, seg=6, rings=3, color=MOON, kind="glow", detail=1)
    out += rod((-2.1, 3.9, -2.1), (2.1, 3.9, 2.1), 0.14, seg=5, color=SILVER, kind="metal", detail=1)
    out += rod((-2.1, 3.9, 2.1), (2.1, 3.9, -2.1), 0.14, seg=5, color=SILVER, kind="metal", detail=1)
    out += elf_tree(-3.2, 2.6, h=3.6, detail=1)
    return out


def elf_archer_tower():
    out = cyl((0, 0, 0), 2.9, 0.6, seg=10, color=WHITE_D)
    out += cyl((0, 0.6, 0), 2.3, 10.0, r_top=2.0, seg=10, color=WHITE)
    out += cyl((0, 10.6, 0), 2.7, 0.4, seg=10, color=SILVER, kind="metal")
    import math
    for k in range(6):
        a = 2 * math.pi * k / 6
        out += cyl((2.3 * math.cos(a), 11.0, 2.3 * math.sin(a)), 0.15, 2.2, seg=4, color=WHITE, detail=1)
    out += cone((0, 13.2, 0), 3.0, 4.0, seg=10, color=TURQ)
    out += banner((0, 9.0, 2.12), w=0.8, h=2.0, pole=False)
    return out


def magic_tower():
    out = cyl((0, 0, 0), 2.9, 0.6, seg=10, color=WHITE_D)
    out += cyl((0, 0.6, 0), 2.2, 11.0, r_top=1.7, seg=10, color=WHITE)
    out += cyl((0, 11.6, 0), 2.4, 0.4, seg=10, color=SILVER, kind="metal")
    import math
    for k in range(4):
        a = 2 * math.pi * k / 4 + 0.4
        out += rod((2.0 * math.cos(a), 12.0, 2.0 * math.sin(a)), (0.5 * math.cos(a), 14.6, 0.5 * math.sin(a)), 0.12, seg=4, color=SILVER, kind="metal")
    out += sphere((0, 14.2, 0), 0.9, seg=8, rings=4, color=(1.0, 0.95, 0.7), kind="glow")
    out += windows([0], 7.5, 1.92, w=0.4, h=1.4, color=(1.0, 0.95, 0.7), kind="glow")
    out += banner((0, 6.5, 2.0), w=0.8, h=2.0, pole=False)
    return out


PROPS = [
    Prop("keep_high_elves", keep_high_elves(), "High Elf keep: round white ward, turquoise dome, spires, trees"),
    Prop("order_of_the_sword", order_of_the_sword(), "Order of the Sword: white hall, two spires, sword statues"),
    Prop("order_of_archers", order_of_archers(), "Order of Archers: open pavilion under a dome, a tree, targets"),
    Prop("griffon_tower", griffon_tower(), "Griffon tower: tall white tower with a nest platform"),
    Prop("tiger_pen", tiger_pen(), "Tiger pen: curved low wall, a shelter, trees"),
    Prop("spirit_garden", spirit_garden(), "Spirit garden: pillars around a glowing pool, three element crystals"),
    Prop("enchantment_tower", enchantment_tower(), "Enchantment tower: slim tower with a floating crystal"),
    Prop("moonwell", moonwell(), "Moonwell: glowing basin under four arches"),
    Prop("elf_archer_tower", elf_archer_tower(), "Elf archer tower: slim white tower, turquoise cone on pillars"),
    Prop("magic_tower", magic_tower(), "Magic tower: white tower with a radiant orb"),
]
