"""Generate the Windows ICO and MSIX PNG assets from the same SVG source."""

import argparse
from io import BytesIO
from pathlib import Path

import resvg_py
from PIL import Image


ROOT = Path(__file__).resolve().parents[1]
SOURCE = ROOT / "assets/icons/app/eqm-logo.svg"
OUTPUT = ROOT / "internal/resources/icons/app/eqm.ico"


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--msix-assets", type=Path, help="Generate MSIX PNG assets instead of the ICO")
    args = parser.parse_args()
    if args.msix_assets is not None:
        args.msix_assets.mkdir(parents=True, exist_ok=True)
        for name, size in (("StoreLogo", 50), ("Square44x44Logo", 44), ("Square150x150Logo", 150)):
            png = resvg_py.svg_to_bytes(svg_path=str(SOURCE), width=size, height=size)
            (args.msix_assets / f"{name}.png").write_bytes(png)
        for size in (16, 24, 32, 48, 256):
            png = resvg_py.svg_to_bytes(svg_path=str(SOURCE), width=size, height=size)
            (args.msix_assets / f"Square44x44Logo.targetsize-{size}_altform-unplated.png").write_bytes(png)
        return
    png = resvg_py.svg_to_bytes(svg_path=str(SOURCE), width=512, height=512)
    OUTPUT.parent.mkdir(parents=True, exist_ok=True)
    with Image.open(BytesIO(png)) as image:
        image.save(OUTPUT, format="ICO", sizes=[(size, size) for size in (16, 32, 48, 256)])


if __name__ == "__main__":
    main()
