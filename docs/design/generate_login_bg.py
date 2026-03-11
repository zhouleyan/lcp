#!/usr/bin/env python3
"""
Stratified Topology — LCP Platform Login Background (Final)
Layered cloud infrastructure as luminous network topology.
"""

import math
import random
from PIL import Image, ImageDraw, ImageFont

W, H = 1920, 1080
FONT_DIR = "/Users/zhouleyan/.claude/plugins/cache/anthropic-agent-skills/document-skills/3d5951151859/skills/canvas-design/canvas-fonts"
random.seed(42)

# Palette
BG          = (10, 14, 32)
BG_LIGHT    = (22, 30, 60)
CYAN        = (0, 210, 235)
CYAN_MED    = (0, 150, 175)
CYAN_DIM    = (0, 90, 115)
TEAL        = (0, 195, 205)
AMBER       = (255, 185, 65)
VIOLET      = (140, 105, 230)
VIOLET_DIM  = (85, 60, 150)
WHITE_SOFT  = (170, 190, 215)
WHITE_DIM   = (90, 110, 135)


def bl(fg, a):
    return tuple(int(BG[i]*(1-a) + fg[i]*a) for i in range(3))


def add_glow(img_rgba, cx, cy, radius, color, intensity):
    """Single efficient glow pass."""
    overlay = Image.new("RGBA", (W, H), (0, 0, 0, 0))
    od = ImageDraw.Draw(overlay)
    steps = max(3, radius // 2)
    for i in range(steps, 0, -1):
        r = radius * i / steps
        t = i / steps
        a = int(255 * intensity * (1 - t) ** 1.8)
        if a < 1:
            continue
        od.ellipse([cx-r, cy-r, cx+r, cy+r], fill=color + (a,))
    return Image.alpha_composite(img_rgba, overlay)


def main():
    img = Image.new("RGB", (W, H), BG)
    draw = ImageDraw.Draw(img)

    # ═══════════════════════════════════════════════════════════
    # 1. ATMOSPHERIC GRADIENT
    # ═══════════════════════════════════════════════════════════
    for y in range(H):
        t = y / H
        b = math.exp(-((t - 0.35)**2) / 0.08) * 0.45
        r = int(BG[0] + (BG_LIGHT[0] - BG[0]) * b)
        g = int(BG[1] + (BG_LIGHT[1] - BG[1]) * b)
        b2 = int(BG[2] + (BG_LIGHT[2] - BG[2]) * b)
        draw.line([(0, y), (W, y)], fill=(r, g, b2))

    # ═══════════════════════════════════════════════════════════
    # 2. FINE GRID (subtle coordinate system)
    # ═══════════════════════════════════════════════════════════
    for x in range(0, W, 48):
        draw.line([(x, 0), (x, H)], fill=bl(CYAN_DIM, 0.045), width=1)
    for y in range(0, H, 48):
        draw.line([(0, y), (W, y)], fill=bl(CYAN_DIM, 0.035), width=1)

    # ═══════════════════════════════════════════════════════════
    # 3. TOPOLOGY FLOW CURVES (PaaS strata: host/db/mw/container)
    # ═══════════════════════════════════════════════════════════
    strata = [
        (190, CYAN_MED,   0.22, [(30, 0.003), (15, 0.008), (6, 0.018)]),
        (360, TEAL,       0.18, [(24, 0.004), (12, 0.01),  (5, 0.022)]),
        (530, VIOLET_DIM, 0.15, [(35, 0.0025),(16, 0.007), (7, 0.016)]),
        (700, CYAN_DIM,   0.20, [(20, 0.0045),(10, 0.012), (4, 0.025)]),
        (860, CYAN_MED,   0.11, [(26, 0.0035),(12, 0.009)]),
    ]
    for yb, color, alpha, harms in strata:
        for off in [0, 3, 7]:
            pts = []
            for x in range(0, W+2, 2):
                y = yb + off
                for amp, freq in harms:
                    y += amp * math.sin(x * freq + off * 0.7 + 1.0)
                pts.append((x, y))
            a = alpha * (1.0 if off == 0 else 0.3 if off == 3 else 0.12)
            draw.line(pts, fill=bl(color, a), width=1)

    # ═══════════════════════════════════════════════════════════
    # 4. HEXAGONAL GRID (container/orchestration topology)
    # ═══════════════════════════════════════════════════════════
    hex_r = 30
    hex_h = hex_r * math.sqrt(3)
    fx, fy = W * 0.72, H * 0.40
    for row in range(-2, int(H / hex_h) + 3):
        for col in range(-2, int(W / (hex_r * 1.5)) + 3):
            cx = col * hex_r * 1.5
            cy = row * hex_h + (hex_h / 2 if col % 2 else 0)
            d = math.hypot(cx - fx, cy - fy)
            if d > 420:
                continue
            falloff = (1 - d / 420) ** 1.3
            a = 0.16 * falloff
            if a < 0.01:
                continue
            pts = [(cx + hex_r * 0.65 * math.cos(math.pi/3*k + math.pi/6),
                    cy + hex_r * 0.65 * math.sin(math.pi/3*k + math.pi/6)) for k in range(6)]
            c = bl(CYAN_DIM, a)
            for k in range(6):
                draw.line([pts[k], pts[(k+1)%6]], fill=c, width=1)
            # Subtle center dot for inner hexes
            if falloff > 0.6:
                draw.point((int(cx), int(cy)), fill=bl(CYAN_DIM, a * 0.5))

    # ═══════════════════════════════════════════════════════════
    # 5. NETWORK NODES
    # ═══════════════════════════════════════════════════════════
    clusters = [
        (1380, 340, 190, 24, True),
        (1560, 260, 130, 12, True),
        (1220, 500, 160, 16, True),
        (1470, 620, 140, 10, True),
        (1620, 440, 110, 8, True),
        (300, 400, 190, 7, False),
        (400, 210, 140, 5, False),
        (170, 650, 130, 3, False),
        (760, 300, 170, 5, False),
        (830, 770, 150, 5, False),
        (620, 170, 110, 3, False),
        (550, 600, 100, 3, False),
    ]
    nodes = []
    for cx, cy, spread, count, pri in clusters:
        for _ in range(count):
            ang = random.uniform(0, 2*math.pi)
            r = abs(random.gauss(0, spread * 0.38))
            nx = max(25, min(W-25, cx + r*math.cos(ang)))
            ny = max(25, min(H-25, cy + r*math.sin(ang)))
            sz = random.choice([2, 3, 3, 4, 4, 5, 5])
            nt = random.choice(["circle"]*4 + ["hex", "square", "diamond"])
            nc = random.choice([CYAN, CYAN, TEAL, VIOLET, AMBER])
            nodes.append({"x": nx, "y": ny, "s": sz, "t": nt, "c": nc, "p": pri})

    # ═══════════════════════════════════════════════════════════
    # 6. CONNECTION PATHWAYS
    # ═══════════════════════════════════════════════════════════
    for i, n1 in enumerate(nodes):
        for j, n2 in enumerate(nodes):
            if j <= i:
                continue
            d = math.hypot(n1["x"]-n2["x"], n1["y"]-n2["y"])
            if d < 130 and random.random() < 0.4:
                a = max(0.05, 0.20 * (1 - d/130))
                draw.line([(n1["x"],n1["y"]), (n2["x"],n2["y"])], fill=bl(CYAN_MED, a), width=1)
            elif d < 220 and random.random() < 0.08:
                steps = int(d / 4)
                for si in range(0, steps, 2):
                    t = si / max(1, steps)
                    px = n1["x"] + (n2["x"]-n1["x"])*t
                    py = n1["y"] + (n2["y"]-n1["y"])*t
                    draw.point((int(px), int(py)), fill=bl(CYAN_DIM, 0.14))

    # ═══════════════════════════════════════════════════════════
    # 7. BEZIER DATA FLOW ARCS
    # ═══════════════════════════════════════════════════════════
    arcs = [
        ((1270, 280), (1370, 200), (1510, 360), CYAN,     0.14),
        ((1370, 410), (1510, 340), (1590, 510), TEAL,     0.11),
        ((1120, 460), (1250, 390), (1320, 580), VIOLET,   0.09),
        ((1470, 220), (1590, 280), (1650, 180), CYAN_MED, 0.09),
        ((250, 340), (350, 280), (420, 460), CYAN_DIM,    0.07),
        ((1330, 560), (1420, 510), (1540, 640), TEAL,     0.07),
        ((1520, 380), (1600, 310), (1680, 420), CYAN,     0.06),
    ]
    for p0, p1, p2, ac, aa in arcs:
        pts = []
        for i in range(81):
            t = i / 80
            x = (1-t)**2*p0[0] + 2*(1-t)*t*p1[0] + t**2*p2[0]
            y = (1-t)**2*p0[1] + 2*(1-t)*t*p1[1] + t**2*p2[1]
            pts.append((x, y))
        for k in range(len(pts)-1):
            draw.line([pts[k], pts[k+1]], fill=bl(ac, aa), width=1)
        # Bright endpoint dots
        for ep in [pts[0], pts[-1]]:
            draw.ellipse([ep[0]-2, ep[1]-2, ep[0]+2, ep[1]+2], fill=bl(ac, aa*2))

    # ═══════════════════════════════════════════════════════════
    # 8. RADIAL GLOWS (energy centers + atmospheric haze)
    # ═══════════════════════════════════════════════════════════
    img_rgba = img.convert("RGBA")

    # Large atmospheric hazes first
    img_rgba = add_glow(img_rgba, 1350, 380, 320, CYAN,     0.020)
    img_rgba = add_glow(img_rgba, int(W*0.7), int(H*0.38), 380, TEAL, 0.012)
    img_rgba = add_glow(img_rgba, 1220, 500, 250, VIOLET,   0.010)

    # Cluster center glows
    cluster_glows = [
        (1380, 340, 90,  CYAN,     0.07),
        (1560, 265, 65,  TEAL,     0.06),
        (1220, 500, 75,  VIOLET,   0.05),
        (1470, 620, 60,  CYAN_MED, 0.04),
        (1620, 440, 50,  TEAL,     0.035),
        (300,  400, 70,  CYAN_DIM, 0.03),
        (760,  300, 50,  CYAN_DIM, 0.02),
    ]
    for gx, gy, gr, gc, gi in cluster_glows:
        img_rgba = add_glow(img_rgba, gx, gy, gr, gc, gi)

    # Node glows (small, focused)
    for n in nodes:
        if n["s"] >= 4:
            r = n["s"] * 6
            inten = 0.10 if n["p"] else 0.05
            img_rgba = add_glow(img_rgba, int(n["x"]), int(n["y"]), r, n["c"], inten)

    img = img_rgba.convert("RGB")
    draw = ImageDraw.Draw(img)

    # ═══════════════════════════════════════════════════════════
    # 9. RENDER NODE SHAPES
    # ═══════════════════════════════════════════════════════════
    for n in nodes:
        x, y, s = n["x"], n["y"], n["s"]
        co = bl(n["c"], 0.80)
        cb = bl(n["c"], 0.95)

        if n["t"] == "circle":
            draw.ellipse([x-s, y-s, x+s, y+s], outline=co, width=1)
            if s >= 3:
                draw.ellipse([x-1, y-1, x+1, y+1], fill=cb)
        elif n["t"] == "hex":
            pts = [(x + s*math.cos(math.pi/3*k), y + s*math.sin(math.pi/3*k)) for k in range(6)]
            draw.polygon(pts, outline=co)
            draw.point((int(x), int(y)), fill=cb)
        elif n["t"] == "square":
            h = s * 0.65
            draw.rectangle([x-h, y-h, x+h, y+h], outline=co, width=1)
            draw.point((int(x), int(y)), fill=cb)
        elif n["t"] == "diamond":
            pts = [(x, y-s), (x+s, y), (x, y+s), (x-s, y)]
            draw.polygon(pts, outline=co)
            draw.point((int(x), int(y)), fill=cb)

    # ═══════════════════════════════════════════════════════════
    # 10. CONCENTRIC RINGS (service boundaries)
    # ═══════════════════════════════════════════════════════════
    for rcx, rcy, radii, rc, ra in [
        (1380, 340, [42, 68], CYAN_DIM,   0.11),
        (1560, 265, [30, 52], TEAL,       0.09),
        (1220, 500, [36, 60], VIOLET_DIM, 0.07),
        (300,  400, [40, 65], CYAN_DIM,   0.05),
    ]:
        for r in radii:
            draw.ellipse([rcx-r, rcy-r, rcx+r, rcy+r], outline=bl(rc, ra), width=1)
            # Partial arc segments for visual interest
            if r > 40:
                for seg_start in range(0, 360, 90):
                    seg_end = seg_start + random.randint(20, 50)
                    draw.arc([rcx-r-4, rcy-r-4, rcx+r+4, rcy+r+4],
                            seg_start, seg_end, fill=bl(rc, ra*0.5), width=1)

    # ═══════════════════════════════════════════════════════════
    # 11. AMBIENT PARTICLES
    # ═══════════════════════════════════════════════════════════
    for _ in range(500):
        px, py = random.randint(0, W), random.randint(0, H)
        a = random.uniform(0.06, 0.28)
        pc = random.choice([CYAN, WHITE_SOFT, TEAL, VIOLET])
        draw.point((px, py), fill=bl(pc, a))

    # ═══════════════════════════════════════════════════════════
    # 12. TYPOGRAPHY (specimen labels)
    # ═══════════════════════════════════════════════════════════
    try:
        fm_xs = ImageFont.truetype(f"{FONT_DIR}/GeistMono-Regular.ttf", 9)
        fm_sm = ImageFont.truetype(f"{FONT_DIR}/GeistMono-Regular.ttf", 11)
        fj_lt = ImageFont.truetype(f"{FONT_DIR}/Jura-Light.ttf", 10)
    except:
        fm_xs = fm_sm = fj_lt = ImageFont.load_default()

    labels = [
        # Stratum layer labels
        ("HOST",       1268, 170, fm_sm, CYAN_MED,  0.28),
        ("stratum.01", 1268, 184, fm_xs, WHITE_DIM, 0.17),
        ("DATABASE",   1298, 340, fm_sm, TEAL,      0.25),
        ("stratum.02", 1298, 354, fm_xs, WHITE_DIM, 0.15),
        ("MIDDLEWARE",  1128, 510, fm_sm, VIOLET,    0.22),
        ("stratum.03", 1128, 524, fm_xs, WHITE_DIM, 0.14),
        ("CONTAINER",  1358, 680, fm_sm, CYAN_MED,  0.25),
        ("stratum.04", 1358, 694, fm_xs, WHITE_DIM, 0.15),
        # Service metadata
        ("node.active", 1415, 365, fm_xs, CYAN_DIM, 0.20),
        ("v2.8.1",      1575, 285, fm_xs, WHITE_DIM, 0.14),
        ("latency:47ms",1525, 460, fm_xs, TEAL,      0.12),
        ("cluster.α",   278,  420, fm_xs, WHITE_DIM, 0.13),
        ("svc:4318",    1500, 640, fm_xs, WHITE_DIM, 0.11),
        ("ns:core",     1240, 525, fm_xs, WHITE_DIM, 0.11),
        ("gw:443",      1610, 250, fm_xs, WHITE_DIM, 0.12),
        # Top-left cartouche
        ("TOPOLOGY",    60,   44,  fj_lt, WHITE_DIM, 0.12),
    ]
    for text, lx, ly, font, lc, la in labels:
        draw.text((lx, ly), text, fill=bl(lc, la), font=font)

    # Tick marks beside stratum labels
    tick_c = bl(WHITE_DIM, 0.12)
    for lx, ly in [(1263, 177), (1293, 347), (1123, 517), (1353, 687)]:
        draw.line([(lx-8, ly+4), (lx-2, ly+4)], fill=tick_c, width=1)

    # ═══════════════════════════════════════════════════════════
    # 13. CORNER FRAME MARKS
    # ═══════════════════════════════════════════════════════════
    mc = bl(WHITE_DIM, 0.09)
    m, ml = 30, 28
    for cx, cy, dx, dy in [(m,m,1,1),(W-m,m,-1,1),(m,H-m,1,-1),(W-m,H-m,-1,-1)]:
        draw.line([(cx, cy), (cx+ml*dx, cy)], fill=mc, width=1)
        draw.line([(cx, cy), (cx, cy+ml*dy)], fill=mc, width=1)

    # ═══════════════════════════════════════════════════════════
    # 14. EDGE VIGNETTE + LEFT DARKENING
    # ═══════════════════════════════════════════════════════════
    img_rgba = img.convert("RGBA")

    # Edge vignette
    edge = Image.new("RGBA", (W, H), (0, 0, 0, 0))
    ed = ImageDraw.Draw(edge)
    for i in range(90):
        a = int(40 * (1 - i/90)**1.8)
        if a < 1: continue
        f = (0, 0, 0, a)
        ed.rectangle([0, i, W, i+1], fill=f)
        ed.rectangle([0, H-1-i, W, H-i], fill=f)
        ed.rectangle([i, 0, i+1, H], fill=f)
        ed.rectangle([W-1-i, 0, W-i, H], fill=f)
    img_rgba = Image.alpha_composite(img_rgba, edge)

    # Left darkening (login form area — left ~35%)
    left = Image.new("RGBA", (W, H), (0, 0, 0, 0))
    ld = ImageDraw.Draw(left)
    for x in range(int(W * 0.38)):
        a = int(25 * (1 - x/(W*0.38))**2.5)
        if a > 0:
            ld.line([(x, 0), (x, H)], fill=(0, 0, 0, a))
    img_rgba = Image.alpha_composite(img_rgba, left)

    img = img_rgba.convert("RGB")

    # ═══════════════════════════════════════════════════════════
    # SAVE
    # ═══════════════════════════════════════════════════════════
    out = "/Users/zhouleyan/Projects/lcp/docs/design/lcp-login-bg.png"
    img.save(out, "PNG")
    print(f"Done: {out} ({W}x{H})")


if __name__ == "__main__":
    main()
