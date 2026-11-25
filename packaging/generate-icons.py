"""Render the supplied Elastic Defend SVG. Requires PyMuPDF and Pillow."""
from pathlib import Path
import io

import fitz
from PIL import Image

ROOT = Path(__file__).resolve().parent.parent
SIZES = (16, 20, 24, 32, 40, 48, 64, 96, 128, 256)

source = fitz.open(ROOT / "build/elastic-defend.svg")
pixmap = source[0].get_pixmap(matrix=fitz.Matrix(32, 32), alpha=True)
image = Image.open(io.BytesIO(pixmap.tobytes("png"))).convert("RGBA")
image.resize((1024, 1024), Image.Resampling.LANCZOS).save(ROOT / "build/appicon.png")
image.save(ROOT / "build/windows/icon.ico", format="ICO", sizes=[(n, n) for n in SIZES])
(ROOT / "internal/tray/icon.ico").write_bytes((ROOT / "build/windows/icon.ico").read_bytes())
print("Generated application and tray icons:", ", ".join(map(str, SIZES)))
