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
    toggle='<button class="recipemd-theme-toggle" type="button" aria-label="Toggle colour theme">◐ Theme</button>
<script>
  // Cycle: system -> light -> dark -> system. Persisted in localStorage.
  (function () {
    var root = document.documentElement, key = "recipemd-theme";
    var saved = localStorage.getItem(key);
    if (saved) root.dataset.theme = saved;
    document.querySelector(".recipemd-theme-toggle").addEventListener("click", function () {
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
</head>
<body>
$article
$toggle
</body>
</html>
HTML
  echo "wrote preview/$name.html"
done
