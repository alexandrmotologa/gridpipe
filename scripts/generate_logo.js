const fs = require('fs');
const path = require('path');
const { Resvg } = require('@resvg/resvg-js');

function buildLogoSvg() {
  return `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 1024 1024" width="1024" height="1024">
  <defs>
    <!-- Squircle container clipping mask -->
    <clipPath id="squircle-clip">
      <rect x="24" y="24" width="976" height="976" rx="220" />
    </clipPath>

    <!-- Electric Cyan Energy Gradients -->
    <linearGradient id="cyan-glow" x1="0%" y1="0%" x2="100%" y2="100%">
      <stop offset="0%" stop-color="#00f5ff"/>
      <stop offset="100%" stop-color="#0284c7"/>
    </linearGradient>

    <linearGradient id="cyan-glow-light" x1="0%" y1="0%" x2="0%" y2="100%">
      <stop offset="0%" stop-color="#38bdf8"/>
      <stop offset="100%" stop-color="#00f5ff"/>
    </linearGradient>

    <linearGradient id="cyan-beam" x1="0%" y1="0%" x2="100%" y2="0%">
      <stop offset="0%" stop-color="#0284c7"/>
      <stop offset="50%" stop-color="#00f5ff"/>
      <stop offset="100%" stop-color="#0284c7"/>
    </linearGradient>

    <!-- Metallic Obsidian & Slate Gradients -->
    <linearGradient id="slate-top-lit" x1="0%" y1="0%" x2="100%" y2="100%">
      <stop offset="0%" stop-color="#64748b"/>
      <stop offset="100%" stop-color="#475569"/>
    </linearGradient>

    <linearGradient id="slate-plate-light" x1="0%" y1="0%" x2="100%" y2="100%">
      <stop offset="0%" stop-color="#475569"/>
      <stop offset="100%" stop-color="#334155"/>
    </linearGradient>

    <linearGradient id="slate-plate-mid" x1="0%" y1="0%" x2="100%" y2="100%">
      <stop offset="0%" stop-color="#334155"/>
      <stop offset="100%" stop-color="#1e293b"/>
    </linearGradient>

    <linearGradient id="slate-plate-dark" x1="0%" y1="0%" x2="100%" y2="100%">
      <stop offset="0%" stop-color="#1e293b"/>
      <stop offset="100%" stop-color="#0f172a"/>
    </linearGradient>

    <linearGradient id="obsidian-deep" x1="0%" y1="0%" x2="100%" y2="100%">
      <stop offset="0%" stop-color="#0f172a"/>
      <stop offset="100%" stop-color="#020617"/>
    </linearGradient>

    <!-- Subtle Container Shadow -->
    <filter id="subtle-shadow" x="-15%" y="-15%" width="130%" height="130%">
      <feDropShadow dx="0" dy="16" stdDeviation="20" flood-color="#0f172a" flood-opacity="0.16" />
    </filter>

    <filter id="eye-glow" x="-30%" y="-30%" width="160%" height="160%">
      <feDropShadow dx="0" dy="0" stdDeviation="6" flood-color="#00f5ff" flood-opacity="0.8" />
    </filter>
  </defs>

  <!-- Luxury White Squircle Container -->
  <rect x="24" y="24" width="976" height="976" rx="220" fill="#ffffff" stroke="#e2e8f0" stroke-width="6" />

  <g clip-path="url(#squircle-clip)">
    <!-- Architectural Engineering Coordinate Grid Motif -->
    <g opacity="0.08" stroke="#0f172a" stroke-width="1.5">
      <line x1="128" y1="0" x2="128" y2="1024" />
      <line x1="256" y1="0" x2="256" y2="1024" />
      <line x1="384" y1="0" x2="384" y2="1024" />
      <line x1="512" y1="0" x2="512" y2="1024" />
      <line x1="640" y1="0" x2="640" y2="1024" />
      <line x1="768" y1="0" x2="768" y2="1024" />
      <line x1="896" y1="0" x2="896" y2="1024" />
      <line x1="0" y1="128" x2="1024" y2="128" />
      <line x1="0" y1="256" x2="1024" y2="256" />
      <line x1="0" y1="384" x2="1024" y2="384" />
      <line x1="0" y1="512" x2="1024" y2="512" />
      <line x1="0" y1="640" x2="1024" y2="640" />
      <line x1="0" y1="768" x2="1024" y2="768" />
      <line x1="0" y1="896" x2="1024" y2="896" />
    </g>

    <g transform="translate(512, 512)" filter="url(#subtle-shadow)">

      <!-- Hexagonal Architectural Gateway Frame -->
      <polygon points="
        0,-410
        355,-205
        355,205
        0,410
        -355,205
        -355,-205
      " fill="none" stroke="#0f172a" stroke-width="42" stroke-linejoin="round" />

      <!-- Inner High-Tech Telemetry Accent Ring -->
      <polygon points="
        0,-374
        324,-187
        324,187
        0,374
        -324,187
        -324,-187
      " fill="none" stroke="#00f5ff" stroke-width="4" opacity="0.5" stroke-dasharray="16, 12" />

      <!-- Corner Crosshairs -->
      <g stroke="#0284c7" stroke-width="2.5" opacity="0.65">
        <line x1="-324" y1="-187" x2="-304" y2="-187" />
        <line x1="-324" y1="-187" x2="-324" y2="-167" />
        <line x1="324" y1="-187" x2="304" y2="-187" />
        <line x1="324" y1="-187" x2="324" y2="-167" />
        <line x1="0" y1="-374" x2="-15" y2="-374" />
        <line x1="0" y1="-374" x2="15" y2="-374" />
        <line x1="0" y1="374" x2="-15" y2="374" />
        <line x1="0" y1="374" x2="15" y2="374" />
      </g>

      <!-- ============================================================== -->
      <!-- 100% Watertight Solid Base Silhouette (Zero-Leak Protection)   -->
      <!-- ============================================================== -->
      <polygon points="
        0,-240
        80,-280
        145,-305
        175,-220
        205,-155
        255,-65
        275,45
        250,155
        210,245
        125,305
        0,335
        -125,305
        -210,245
        -250,155
        -275,45
        -255,-65
        -205,-155
        -175,-220
        -145,-305
        -80,-280
      " fill="#020617" />

      <!-- ============================================================== -->
      <!-- BEAVER FOREHEAD & EARS MESH                                    -->
      <!-- ============================================================== -->

      <!-- EARS (Architectural Low-Poly Beaver Ears) -->
      <!-- Right Ear -->
      <polygon points="80,-240 145,-305 175,-220 115,-200" fill="url(#slate-plate-light)" />
      <polygon points="95,-240 138,-292 162,-225 118,-210" fill="url(#obsidian-deep)" />
      <polygon points="138,-292 162,-225 152,-230" fill="#00f5ff" opacity="0.9" />

      <!-- Left Ear -->
      <polygon points="-80,-240 -145,-305 -175,-220 -115,-200" fill="url(#slate-plate-mid)" />
      <polygon points="-95,-240 -138,-292 -162,-225 -118,-210" fill="url(#obsidian-deep)" />
      <polygon points="-138,-292 -162,-225 -152,-230" fill="#00f5ff" opacity="0.7" />

      <!-- CROWN / FOREHEAD ARCHITECTURE -->
      <polygon points="0,-240 80,-240 55,-160 0,-170" fill="url(#slate-top-lit)" />
      <polygon points="0,-240 -80,-240 -55,-160 0,-170" fill="url(#slate-plate-light)" />

      <!-- Crown Side Temples -->
      <polygon points="80,-240 115,-200 150,-155 55,-160" fill="url(#slate-plate-light)" />
      <polygon points="-80,-240 -115,-200 -150,-155 -55,-160" fill="url(#slate-plate-dark)" />

      <polygon points="150,-155 205,-155 180,-85 125,-95 55,-160" fill="url(#slate-plate-mid)" />
      <polygon points="-150,-155 -205,-155 -180,-85 -125,-95 -55,-160" fill="url(#obsidian-deep)" />

      <!-- FOREHEAD TO BROW FACETS -->
      <polygon points="0,-170 55,-160 40,-105 0,-115" fill="url(#slate-plate-light)" />
      <polygon points="0,-170 -55,-160 -40,-105 0,-115" fill="url(#slate-plate-mid)" />

      <polygon points="55,-160 125,-95 40,-105" fill="url(#slate-top-lit)" />
      <polygon points="-55,-160 -125,-95 -40,-105" fill="url(#slate-plate-dark)" />

      <!-- BROWS (Sharp Structural Hood with Predatory / Alert Upward Slant) -->
      <polygon points="0,-115 40,-105 120,-95 45,-70 0,-80" fill="url(#slate-plate-light)" />
      <polygon points="0,-115 -40,-105 -120,-95 -45,-70 0,-80" fill="url(#slate-plate-mid)" />

      <polygon points="120,-95 180,-85 135,-45 90,-58 45,-70" fill="url(#slate-plate-mid)" />
      <polygon points="-120,-95 -180,-85 -135,-45 -90,-58 -45,-70" fill="url(#slate-plate-dark)" />

      <!-- ============================================================== -->
      <!-- EYES: Sharp, High-Intelligence Cyan Optic Prisms               -->
      <!-- Natural Predatory / Alert Cant: Outer corner (-82) HIGHER than Inner (-58) -->
      <!-- ============================================================== -->
      <!-- Right Eye Socket -->
      <polygon points="45,-68 116,-90 122,-64 52,-48" fill="#020617" />
      <!-- Right Eye Optical Gem (Slanted up toward temples) -->
      <polygon points="52,-64 112,-84 116,-68 58,-52" fill="url(#cyan-glow)" filter="url(#eye-glow)" />
      <!-- Razor Pupil / Reflection -->
      <polygon points="78,-68 86,-72 82,-58 74,-56" fill="#ffffff" />

      <!-- Left Eye Socket -->
      <polygon points="-45,-68 -116,-90 -122,-64 -52,-48" fill="#020617" />
      <!-- Left Eye Optical Gem (Slanted up toward temples) -->
      <polygon points="-52,-64 -112,-84 -116,-68 -58,-52" fill="url(#cyan-glow-light)" filter="url(#eye-glow)" />
      <!-- Razor Pupil / Reflection -->
      <polygon points="-78,-68 -86,-72 -82,-58 -74,-56" fill="#ffffff" />

      <!-- Eye Accent Brow Laser Rim -->
      <line x1="45" y1="-70" x2="120" y2="-95" stroke="#00f5ff" stroke-width="2.5" opacity="0.8" />
      <line x1="-45" y1="-70" x2="-120" y2="-95" stroke="#00f5ff" stroke-width="2.5" opacity="0.8" />

      <!-- ============================================================== -->
      <!-- SNOUT / MUZZLE & CHEEKS                                        -->
      <!-- Beaver Wide Zygomatic Cheek Profile (Architectural Power)      -->
      <!-- ============================================================== -->

      <!-- Bridge of Snout -->
      <polygon points="0,-80 45,-70 32,-15 0,0" fill="url(#slate-top-lit)" />
      <polygon points="0,-80 -45,-70 -32,-15 0,0" fill="url(#slate-plate-mid)" />

      <!-- Cheeks Upper -->
      <polygon points="135,-45 180,-85 235,-65 200,-5 130,-15" fill="url(#slate-plate-light)" />
      <polygon points="-135,-45 -180,-85 -235,-65 -200,-5 -130,-15" fill="url(#slate-plate-dark)" />

      <polygon points="45,-70 52,-48 116,-64 130,-15 75,10 32,-15" fill="url(#slate-plate-light)" />
      <polygon points="-45,-70 -52,-48 -116,-64 -130,-15 -75,10 -32,-15" fill="url(#slate-plate-dark)" />

      <!-- Wide Cheek Lower Wings -->
      <polygon points="130,-15 200,-5 255,-5 275,45 210,75 140,50" fill="url(#slate-plate-light)" />
      <polygon points="-130,-15 -200,-5 -255,-5 -275,45 -210,75 -140,50" fill="url(#obsidian-deep)" />

      <!-- NOSE PAD (Polygonal Beaver Nose with Cyan Top Line) -->
      <polygon points="0,0 32,-15 38,20 0,32" fill="url(#slate-plate-dark)" />
      <polygon points="0,0 -32,-15 -38,20 0,32" fill="url(#obsidian-deep)" />
      <polygon points="0,0 32,-15 0,14 -32,-15" fill="url(#cyan-glow)" />

      <!-- LATERAL WHISKER PADS (SLATE TITANIUM GEOMETRIC PLANES) -->
      <!-- Spread horizontally across cheeks - NO BUCK TEETH OPTICAL ILLUSIONS -->
      <polygon points="0,32 38,20 75,10 140,50 125,95 60,90 0,98" fill="url(#slate-top-lit)" />
      <polygon points="0,32 -38,20 -75,10 -140,50 -125,95 -60,90 0,98" fill="url(#slate-plate-mid)" />

      <polygon points="140,50 210,75 185,125 125,95" fill="url(#slate-plate-mid)" />
      <polygon points="-140,50 -210,75 -185,125 -125,95" fill="url(#slate-plate-dark)" />

      <!-- LOWER CHIN & MANDIBLE -->
      <polygon points="0,98 60,90 40,140 0,148" fill="url(#slate-plate-light)" />
      <polygon points="0,98 -60,90 -40,140 0,148" fill="url(#slate-plate-dark)" />
      <polygon points="60,90 125,95 95,140 40,140" fill="url(#slate-plate-mid)" />
      <polygon points="-60,90 -125,95 -95,140 -40,140" fill="url(#obsidian-deep)" />

      <!-- ============================================================== -->
      <!-- BODY, SHOULDERS & TABULAR GRID MATRIX                          -->
      <!-- ============================================================== -->

      <!-- Outer Shoulders -->
      <polygon points="210,75 275,45 250,155 195,165 185,125" fill="url(#slate-plate-mid)" />
      <polygon points="-210,75 -275,45 -250,155 -195,165 -185,125" fill="url(#slate-plate-dark)" />

      <polygon points="195,165 250,155 210,245 150,230" fill="url(#slate-plate-dark)" />
      <polygon points="-195,165 -250,155 -210,245 -150,230" fill="url(#obsidian-deep)" />

      <!-- Chest Flanks -->
      <polygon points="0,148 40,140 95,140 120,205 0,220" fill="url(#slate-plate-light)" />
      <polygon points="0,148 -40,140 -95,140 -120,205 0,220" fill="url(#slate-plate-dark)" />

      <polygon points="95,140 185,125 195,165 150,230 120,205" fill="url(#slate-plate-mid)" />
      <polygon points="-95,140 -185,125 -195,165 -150,230 -120,205" fill="url(#obsidian-deep)" />

      <!-- Lower Foundation -->
      <polygon points="0,220 120,205 150,230 125,305 0,335" fill="url(#slate-plate-mid)" />
      <polygon points="0,220 -120,205 -150,230 -125,305 0,335" fill="url(#slate-plate-dark)" />

      <polygon points="150,230 210,245 175,295 125,305" fill="url(#slate-plate-dark)" />
      <polygon points="-150,230 -210,245 -175,295 -125,305" fill="url(#obsidian-deep)" />

      <!-- ============================================================== -->
      <!-- RELATIONAL SPREADSHEET PIPELINE MATRIX                         -->
      <!-- Pure geometric 3x2 data cells aligned on sternum               -->
      <!-- ============================================================== -->
      <!-- Background Conduit Channel -->
      <polygon points="0,155 75,170 65,305 0,320 -65,305 -75,170" fill="#020617" />

      <!-- Grid Cells (Tabular Matrix) -->
      <!-- Row 1 -->
      <polygon points="-65,180 -8,180 -8,215 -60,215" fill="url(#cyan-glow)" opacity="0.95" />
      <polygon points="8,180 65,180 60,215 8,215" fill="url(#cyan-glow-light)" opacity="0.95" />

      <!-- Row 2 -->
      <polygon points="-60,225 -8,225 -8,260 -55,260" fill="url(#slate-plate-light)" />
      <polygon points="8,225 60,225 55,260 8,260" fill="url(#cyan-glow)" opacity="0.9" />

      <!-- Row 3 -->
      <polygon points="-55,270 -8,270 -8,298 -45,298" fill="url(#cyan-glow)" opacity="0.8" />
      <polygon points="8,270 55,270 45,298 8,298" fill="url(#slate-top-lit)" />

      <!-- Central Data Pipeline Divider -->
      <line x1="0" y1="165" x2="0" y2="310" stroke="#ffffff" stroke-width="2.5" opacity="0.9" />

    </g>
  </g>
</svg>`;
}

async function renderLogo(outputDir) {
  const svg = buildLogoSvg();
  const svgPath = path.join(outputDir, 'logo.svg');
  const pngPath = path.join(outputDir, 'logo.png');

  fs.writeFileSync(svgPath, svg, 'utf8');
  console.log('✓ Wrote SVG to:', svgPath);

  const resvg = new Resvg(svg, {
    fitTo: {
      mode: 'width',
      value: 1024,
    },
    font: {
      loadSystemFonts: false,
    },
  });

  const pngData = resvg.render();
  const pngBuffer = pngData.asPng();
  fs.writeFileSync(pngPath, pngBuffer);
  console.log(`✓ Successfully rendered logo.svg and logo.png at 1024x1024 (${pngBuffer.length} bytes)`);
}

const targetDir = path.resolve(__dirname, '../docs/images');
if (!fs.existsSync(targetDir)) {
  fs.mkdirSync(targetDir, { recursive: true });
}

renderLogo(targetDir).catch(err => {
  console.error('Error rendering logo:', err);
  process.exit(1);
});
