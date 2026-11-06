"""Draws the app icons: the chosen one to resources/icon.png, and the two
other candidates beside it, for trying colours and shapes.

Each is a 1024 square PNG: a body of 824 on Apple's grid, over a soft shadow.
Graphics are drawn at 4096 and scaled down, so edges stay smooth.
"""

from PIL import Image, ImageDraw, ImageFilter

S = 4096
BODY = 3296
MARGIN = (S - BODY) // 2
RADIUS = int(BODY * 0.2237)


def gradient(size, top, bottom):
    g = Image.new("RGB", size)
    d = ImageDraw.Draw(g)
    for y in range(size[1]):
        t = y / (size[1] - 1)
        c = tuple(round(top[i] + (bottom[i] - top[i]) * t) for i in range(3))
        d.line([(0, y), (size[0], y)], fill=c)
    return g


def rounded(size, radius):
    m = Image.new("L", size, 0)
    ImageDraw.Draw(m).rounded_rectangle([0, 0, size[0] - 1, size[1] - 1], radius, fill=255)
    return m


def body(top, bottom):
    canvas = Image.new("RGBA", (S, S), (0, 0, 0, 0))
    mask = rounded((BODY, BODY), RADIUS)

    shadow = Image.new("RGBA", (S, S), (0, 0, 0, 0))
    shadow.paste((0, 0, 0, 80), (MARGIN, MARGIN + 40), mask)
    shadow = shadow.filter(ImageFilter.GaussianBlur(56))
    canvas = Image.alpha_composite(canvas, shadow)

    canvas.paste(gradient((BODY, BODY), top, bottom).convert("RGBA"), (MARGIN, MARGIN), mask)
    return canvas


def page(w, h, radius=72, panel=(205, 214, 245, 255)):
    """A white page with a 2x2 comic grid."""
    p = Image.new("RGBA", (w, h), (0, 0, 0, 0))
    d = ImageDraw.Draw(p)
    d.rounded_rectangle([0, 0, w - 1, h - 1], radius, fill=(255, 255, 255, 255))
    gap, pad = 60, 120
    pw = (w - pad * 2 - gap) // 2
    ph = (h - pad * 2 - gap) // 2
    for i in range(2):
        for j in range(2):
            x = pad + i * (pw + gap)
            y = pad + j * (ph + gap)
            d.rounded_rectangle([x, y, x + pw, y + ph], 24, fill=panel)
    return p


def paste_above(canvas, img, pos, cut):
    """Composites img, keeping only what sits above y = cut."""
    layer = Image.new("RGBA", canvas.size, (0, 0, 0, 0))
    layer.paste(img, pos, img)
    mask = Image.new("L", canvas.size, 0)
    ImageDraw.Draw(mask).rectangle([0, 0, canvas.width, cut], fill=255)
    empty = Image.new("RGBA", canvas.size, (0, 0, 0, 0))
    canvas.alpha_composite(Image.composite(layer, empty, mask))


def pull(d, x, y, w=260, h=180):
    """A zipper slider with a ring under it."""
    d.rounded_rectangle([x - w // 2, y - h // 2, x + w // 2, y + h // 2], 48, fill=(240, 244, 250, 255))
    d.rounded_rectangle([x - 36, y + h // 2 - 20, x + 36, y + h // 2 + 130], 36, fill=(240, 244, 250, 255))
    cy = y + h // 2 + 210
    d.ellipse([x - 100, cy - 100, x + 100, cy + 100], outline=(240, 244, 250, 255), width=46)


def save(img, path):
    img.resize((1024, 1024), Image.LANCZOS).save(path)


# 1. A comic page with a zip down its right edge, on indigo.
def icon_a(path):
    c = body((96, 112, 255), (92, 54, 198))
    ox, oy = MARGIN, MARGIN

    c.alpha_composite(page(1960, 2500), (ox + 400, oy + 400))

    d = ImageDraw.Draw(c)
    x = ox + 2400
    d.rounded_rectangle([x - 100, oy + 380, x + 100, oy + 2900], 96, fill=(36, 24, 88, 255))
    flip, yy = False, oy + 440
    while yy < oy + 2680:
        bx = x - 72 if flip else x + 8
        d.rounded_rectangle([bx, yy, bx + 64, yy + 82], 22, fill=(226, 232, 246, 255))
        flip = not flip
        yy += 124
    pull(d, x, oy + 2790, 260, 180)
    save(c, path)


# 2. A zip pouch with comic pages standing in it, on orange.
def icon_b(path):
    c = body((255, 158, 84), (226, 70, 26))
    ox, oy = MARGIN, MARGIN

    pouch_top = oy + 2020
    for dx, dy, ang in ((0, -80, -5), (0, 0, 0), (0, 60, 5)):
        p = page(1320, 1740, panel=(228, 216, 242, 255))
        p = p.rotate(ang, resample=Image.BICUBIC, expand=True)
        paste_above(c, p, (ox + 1648 - p.width // 2 + dx, oy + 430 + dy), pouch_top)

    d = ImageDraw.Draw(c)
    box = [ox + 560, pouch_top, ox + 2740, oy + 2800]
    d.rounded_rectangle(box, 170, fill=(255, 255, 255, 255))
    d.rounded_rectangle([box[0] + 46, box[1] + 46, box[2] - 46, box[3] - 46], 140, fill=(255, 233, 214, 255))

    zy = pouch_top + 96
    d.rounded_rectangle([box[0] + 70, zy - 38, box[2] - 70, zy + 38], 38, fill=(212, 218, 233, 255))
    for i in range(19):
        bx = box[0] + 110 + i * 100
        d.rounded_rectangle([bx, zy - 58, bx + 58, zy + 58], 20, fill=(152, 160, 182, 255))
    pull(d, box[2] - 230, zy + 30, 230, 160)
    save(c, path)


# 3. A page pulled half out of a zip slot, on teal.
def icon_c(path):
    c = body((74, 222, 206), (10, 132, 128))
    ox, oy = MARGIN, MARGIN

    top, bottom = oy + 2360, oy + 2620
    p = page(1700, 2100, panel=(208, 232, 228, 255)).rotate(-7, resample=Image.BICUBIC, expand=True)
    paste_above(c, p, (ox + 1648 - p.width // 2, oy + 440), top + 30)

    d = ImageDraw.Draw(c)
    d.rounded_rectangle([ox + 330, top, ox + 2870, bottom], 130, fill=(8, 78, 76, 255))
    for i in range(20):
        bx = ox + 380 + i * 118
        d.rounded_rectangle([bx, top + 36, bx + 64, bottom - 36], 26, fill=(130, 234, 222, 255))
    pull(d, ox + 2700, (top + bottom) // 2 + 30, 240, 170)
    save(c, path)


if __name__ == "__main__":
    icon_a("icon/a-comic-zipper.png")
    icon_b("icon/b-zip-pouch.png")
    icon_c("resources/icon.png")
    print("done")
