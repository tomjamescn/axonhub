#!/bin/sh
# Download Google Fonts (woff2) and generate a self-contained fonts.css.
# Usage: sh download-fonts.sh <output-dir>
# The output is committed to the repo so dev/build/runtime need no external network.
set -e

FONT_DIR="$1"
if [ -z "$FONT_DIR" ]; then
  echo "Usage: $0 <output-dir>" >&2
  exit 1
fi
mkdir -p "$FONT_DIR"

# Modern browser UA so Google Fonts serves woff2 (variable fonts) instead of ttf.
UA="Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0 Safari/537.36"

# Keep in sync with the <link> block that used to live in frontend/index.html
# and the font lists in frontend/src/config/fonts.ts.
CSS_URL="https://fonts.googleapis.com/css2?family=Fira+Code:wght@300..700&family=Inter:ital,opsz,wght@0,14..32,100..900;1,14..32,100..900&family=JetBrains+Mono:wght@100..800&family=Manrope:wght@200..800&family=Montserrat:wght@100..900&family=Open+Sans:ital,wght@0,300..800;1,300..800&family=Source+Serif+4:opsz,wght@8..60,200..900&display=swap"

TMP_CSS="${FONT_DIR}/.fonts-remote.css"

echo "Fetching Google Fonts CSS..."
curl -sfL -A "$UA" "$CSS_URL" -o "$TMP_CSS"

count=0
for url in $(grep -o 'https://fonts\.gstatic\.com/[^ )]*\.woff2' "$TMP_CSS" | sort -u); do
    fname=$(basename "$url")
    if [ ! -f "${FONT_DIR}/${fname}" ]; then
        echo "  ${fname}"
        curl -sfL -o "${FONT_DIR}/${fname}" "$url"
    fi
    count=$((count + 1))
done

# Rewrite remote gstatic URLs to local relative paths. Vite resolves these
# against the project base at build time.
sed -E 's|https://fonts\.gstatic\.com/s/[^/]+/[^/]+/([^ )]+\.woff2)|./\1|g' \
    "$TMP_CSS" > "${FONT_DIR}/fonts.css"

rm -f "$TMP_CSS"
echo "Done. ${count} font files in ${FONT_DIR}/"
