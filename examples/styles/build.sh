#!/usr/bin/env sh
# Regenerates preview/*.html: the sample recipe rendered with each stylesheet.
#
#   ./build.sh            # from examples/styles
#
# Then optionally capture screenshots (needs Node + Playwright):
#
#   node screenshot.js
set -eu

cd "$(dirname "$0")"
mkdir -p preview
article=$(cd ../.. && go run ./examples/renderhtml examples/styles/sample.md)

for css in [0-9][0-9]-*.css; do
  name=${css%.css}
  toggle=""
  if [ "$name" = "05-complete" ]; then
    toggle='<button class="theme-toggle" type="button" aria-label="Toggle colour theme">◐ Theme</button>
<script>
  // Cycle: system -> light -> dark -> system. Persisted in localStorage.
  (function () {
    var root = document.documentElement, key = "recipemd-theme";
    var saved = localStorage.getItem(key);
    if (saved) root.dataset.theme = saved;
    document.querySelector(".theme-toggle").addEventListener("click", function () {
      var next = { "": "light", light: "dark", dark: "" }[root.dataset.theme || ""];
      if (next) { root.dataset.theme = next; localStorage.setItem(key, next); }
      else { delete root.dataset.theme; localStorage.removeItem(key); }
    });
  })();
</script>'
  fi
  cat > "preview/$name.html" <<HTML
<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>recipemd — $name</title>
<link rel="stylesheet" href="../$css">
<style>
  body { margin: 0; }
  .theme-toggle { position: fixed; bottom: 1rem; right: 1rem; z-index: 1;
    padding: 0.4rem 0.8rem; border: 1px solid #8884; border-radius: 999px;
    background: Canvas; color: CanvasText; font: 0.9rem system-ui, sans-serif; cursor: pointer; }
  @media print { .theme-toggle { display: none; } }
</style>
</head>
<body>
$article
$toggle
</body>
</html>
HTML
  echo "wrote preview/$name.html"
done
