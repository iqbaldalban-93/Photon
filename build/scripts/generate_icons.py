import io, re, cairosvg
from PIL import Image

SRC = open('build/logo.svg').read()

def svg_for(stroke):
    s = SRC.replace('width="1024" height="1024" ', '')
    return s.replace('stroke-width="1.5"', f'stroke-width="{stroke}"')

def render(size, stroke=1.5):
    big = size * 4 if size < 256 else size * 2
    png = cairosvg.svg2png(bytestring=svg_for(stroke).encode(), output_width=big, output_height=big)
    im = Image.open(io.BytesIO(png)).convert('RGBA')
    if big != size:
        im = im.resize((size, size), Image.LANCZOS)
    return im

def stroke_for(size):
    if size <= 24: return 2.4
    if size <= 32: return 2.1
    if size <= 48: return 1.8
    return 1.5

render(1024).save('build/appicon.png', optimize=True)

sizes = [16, 20, 24, 32, 40, 48, 64, 96, 128, 256]
frames = [render(s, stroke_for(s)) for s in sizes]
for name in ('icon.ico', 'icons.ico'):
    frames[-1].save(f'build/windows/{name}', format='ICO', sizes=[(s, s) for s in sizes], append_images=frames[:-1])

render(64, stroke_for(64)).save('build/trayicon.png', optimize=True)
render(256).save('build/notification.png', optimize=True)

open('frontend/public/favicon.svg', 'w').write(svg_for(1.5).replace('<svg ', '<svg ', 1))
for s in (16, 32, 48):
    render(s, stroke_for(s)).save(f'/tmp/fav{s}.png')
fav = [render(s, stroke_for(s)) for s in (16, 32, 48)]
fav[-1].save('frontend/public/favicon.ico', format='ICO', sizes=[(16,16),(32,32),(48,48)], append_images=fav[:-1])
render(32, stroke_for(32)).save('frontend/public/favicon-32.png', optimize=True)
render(192).save('frontend/public/icon-192.png', optimize=True)
render(180).save('frontend/public/apple-touch-icon.png', optimize=True)
