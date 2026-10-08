"""Generate localized Store introduction covers and the shared transparent icon."""

import argparse
import xml.etree.ElementTree as ET
from io import BytesIO
from pathlib import Path

import resvg_py
from PIL import Image, ImageDraw, ImageFont


ROOT = Path(__file__).resolve().parents[1]
ASSETS = ROOT / "assets/microsoft-store"
BASELINE = ROOT / "assets/social-preview/social-preview.svg"
ICON = ROOT / "assets/icons/app/eqm-logo.svg"
FONT = ROOT / "assets/social-preview/fonts/FloriInputUI-Medium.ttf"
SVG = "{http://www.w3.org/2000/svg}"
ET.register_namespace("", "http://www.w3.org/2000/svg")


def make_cover(language: str) -> Path:
    """Extend the approved composition without cropping or stretching it."""
    tree = ET.parse(BASELINE)
    root = tree.getroot()
    root.set("width", "1600")
    root.set("height", "900")
    root.set("viewBox", "0 0 1280 720")
    root.find(f"{SVG}rect").set("height", "720")
    style = root.find(f"{SVG}style")
    style.text = style.text.replace(
        'url("fonts/', 'url("../../social-preview/fonts/'
    )
    copy = root.find(f"{SVG}g")
    copy.set("transform", "translate(80 118)")
    logo = root.find(f"{SVG}svg")
    logo.set("y", f"{float(logo.get('y')) + 40:.6f}")

    if language == "en-US":
        root.set("lang", "en")
        root.find(f"{SVG}title").text = "EQM — Switch EQ profiles in any terminal"
        root.find(f"{SVG}desc").text = (
            "Easily switch EQ profiles in any terminal, without opening Equalizer "
            "APO’s Configuration Editor. Import, switch and remove preset EQ "
            "profiles. English Store introduction cover based on the final approved composition."
        )
        introductions = copy.findall(f"{SVG}text")
        introductions[1].text = "Easily switch EQ profiles in any terminal"
        introductions[1].set("font-size", "28")
        introductions[2].text = "Without opening Equalizer APO’s Configuration Editor"
        introductions[2].set("font-size", "22")
        translations = {
            "导入": "Import",
            "切换": "Switch",
            "移除": "Remove",
            "预设 EQ 配置": "Preset EQ profiles",
        }
        for node in copy.iter():
            if node.text in translations:
                node.text = translations[node.text]
        switch = next(node for node in copy.iter(f"{SVG}text") if node.text == "Switch")
        shared = switch.find(f"{SVG}tspan")
        font = ImageFont.truetype(str(FONT), 23)
        shared.set("x", f"{225 + font.getlength('Switch') + 34:g}")
    else:
        root.set("lang", "zh-CN")
        root.find(f"{SVG}desc").text += " 商店介绍头图，16:9，上下延展同色背景。"

    source = ASSETS / language / "cover.svg"
    source.parent.mkdir(parents=True, exist_ok=True)
    tree.write(source, encoding="utf-8", xml_declaration=True)
    source.write_text(
        "\n".join(line.rstrip() for line in source.read_text(encoding="utf-8").splitlines()) + "\n",
        encoding="utf-8",
    )
    return source


def render_cover(source: Path) -> None:
    """Render with the exact licensed brand font and an opaque PNG background."""
    png = resvg_py.svg_to_bytes(
        svg_path=str(source),
        resources_dir=str(source.parent),
        skip_system_fonts=True,
        font_files=[str(FONT)],
        font_family="FloriInputUI",
        sans_serif_family="FloriInputUI",
        languages=["zh-CN", "en"],
    )
    with Image.open(BytesIO(png)) as rendered:
        if rendered.size != (1600, 900):
            raise ValueError(f"Expected 1600x900 cover, got {rendered.size}")
        rendered.convert("RGB").save(source.with_suffix(".png"), optimize=True)


def make_review_sheet(output: Path) -> None:
    """Preview both localized covers and the icon's transparent surroundings."""
    review = Image.new("RGB", (1344, 530), "#d0d7de")
    draw = ImageDraw.Draw(review)
    font = ImageFont.truetype(str(FONT), 18)
    for left, language, label, background, foreground in (
        (12, "zh-CN", "中文 / zh-CN", "#ffffff", "#304455"),
        (682, "en-US", "English / en-US", "#0d1117", "#aeb6c3"),
    ):
        draw.rounded_rectangle((left, 12, left + 650, 428), radius=12, fill=background)
        draw.text((left + 10, 22), label, font=font, fill=foreground)
        with Image.open(ASSETS / language / "cover.png") as cover:
            review.paste(cover.resize((640, 360), Image.Resampling.LANCZOS), (left + 5, 56))

    for y in range(444, 516, 8):
        for x in range(16, 88, 8):
            color = "#ffffff" if ((x - 16) + (y - 444)) // 8 % 2 == 0 else "#b8c1cb"
            draw.rectangle((x, y, x + 7, y + 7), fill=color)
    with Image.open(ASSETS / "app-icon.png") as icon:
        thumbnail = icon.resize((72, 72), Image.Resampling.LANCZOS)
        review.paste(thumbnail, (16, 444), thumbnail)
    draw.text((108, 458), "Shared icon / 中英文共用：300×300 RGBA PNG", font=font, fill="#304455")
    output = output.resolve()
    output.parent.mkdir(parents=True, exist_ok=True)
    review.save(output, optimize=True)
    print(f"Review sheet: {output}")


def main() -> None:
    """Write only the dedicated Store assets, preserving all source artwork."""
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument(
        "--review-output", type=Path,
        help="Optional review sheet path; no review is generated by default",
    )
    args = parser.parse_args()
    ASSETS.mkdir(parents=True, exist_ok=True)
    for language in ("zh-CN", "en-US"):
        source = make_cover(language)
        render_cover(source)
    png = resvg_py.svg_to_bytes(svg_path=str(ICON), width=300, height=300)
    with Image.open(BytesIO(png)) as icon:
        if icon.size != (300, 300):
            raise ValueError(f"Expected 300x300 icon, got {icon.size}")
        icon.convert("RGBA").save(ASSETS / "app-icon.png", optimize=True)
    if args.review_output is not None:
        make_review_sheet(args.review_output)
    for path in (ASSETS / "app-icon.png", ASSETS / "zh-CN/cover.png", ASSETS / "en-US/cover.png"):
        with Image.open(path) as asset:
            print(f"{path.relative_to(ROOT)}: {asset.width}x{asset.height}, {asset.mode}, {path.stat().st_size:,} bytes")


if __name__ == "__main__":
    main()
