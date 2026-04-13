#!/bin/bash
# Script to download and prepare Cyrillic font for gofpdf
# Run this script to enable proper Cyrillic rendering in PDF documents

set -e

echo "Downloading DejaVu Sans font for Cyrillic support..."

# Create fonts directory
mkdir -p fonts

# Download DejaVu Sans (free font with Cyrillic support)
FONT_URL="https://github.com/dejavu-fonts/dejavu-fonts/releases/download/version_2_37/dejavu-fonts-ttf-2.37.tar.bz2"

echo "Downloading font package..."
curl -L -o fonts/dejavu.tar.bz2 "$FONT_URL"

echo "Extracting fonts..."
tar -xjf fonts/dejavu.tar.bz2 -C fonts/

# Copy the fonts we need
cp fonts/dejavu-fonts-ttf-2.37/ttf/DejaVuSans.ttf fonts/
cp fonts/dejavu-fonts-ttf-2.37/ttf/DejaVuSans-Bold.ttf fonts/

# Clean up
rm -rf fonts/dejavu-fonts-ttf-2.37 fonts/dejavu.tar.bz2

echo ""
echo "Font files downloaded successfully!"
echo "Files in fonts/:"
ls -la fonts/

echo ""
echo "NOTE: gofpdf also needs .json font definition files."
echo "To generate them, run:"
echo "  cd fonts && go run github.com/jung-kurt/gofpdf/makefont -b DejaVuSans.ttf -e cp1251 -z DejaVuSans-Bold.ttf -e cp1251"
echo ""
echo "Or use the simpler approach - gofpdf v1.16.2+ supports AddUTF8Font with just .ttf files."
echo "The current code uses AddUTF8Font which should work with the .ttf files directly."
echo ""
echo "Font setup complete!"
