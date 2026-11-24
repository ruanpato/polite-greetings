// Builds the card URL from the form and shows the preview and the README snippet.
// No framework, no build step: the server embeds this file as it is.
"use strict";

const $ = (selector) => document.querySelector(selector);
const checked = (name) => [...document.querySelectorAll(`input[name="${name}"]:checked`)].map((i) => i.value);
const radio = (name) => checked(name)[0];

// WCAG 2 contrast, the same formula as the card package's tests.
function luminance(hex) {
  const [r, g, b] = [0, 2, 4].map((i) => parseInt(hex.slice(i, i + 2), 16) / 255)
    .map((c) => (c <= 0.04045 ? c / 12.92 : ((c + 0.055) / 1.055) ** 2.4));
  return 0.2126 * r + 0.7152 * g + 0.0722 * b;
}
function contrast(a, b) {
  const [hi, lo] = [luminance(a), luminance(b)].sort((x, y) => y - x);
  return (hi + 0.05) / (lo + 0.05);
}

const roundTo = (value, step) => Math.round(value / step) * step;

function query(theme) {
  const q = new URLSearchParams();
  q.set("tz", $("#tz").value.trim());
  q.set("lat", $("#lat").value);
  q.set("lon", $("#lon").value);
  const lang = $("#lang").value;
  if (lang !== "en") q.set("lang", lang);
  const layout = radio("layout");
  if (layout !== "classic") q.set("layout", layout);
  const facts = checked("show").join(",");
  if (layout !== "compact" && facts !== "sun") q.set("show", facts);
  if (theme !== "auto") q.set("theme", theme);
  if ($("#custom").checked) {
    for (const id of ["bg", "accent", "text"]) q.set(id, $(`#${id}`).value.slice(1).toUpperCase());
  }
  const radius = Number($("#radius").value);
  if (radius !== (layout === "compact" ? 24 : 12)) q.set("radius", String(radius));
  if (!$("#border").checked) q.set("border", "false");
  if ($("#animate").checked) q.set("animate", "true");
  if (radio("clock") === "12h") q.set("clock", "12h");
  if ($("#width").value) q.set("width", $("#width").value);
  return q;
}

const cardURL = (q) => `${location.origin}/api/v2/greeting?${q}`;
const htmlAttr = (url) => url.replaceAll("&", "&amp;");

function update() {
  const placed = $("#tz").value.trim() !== "" && $("#lat").value !== "" && $("#lon").value !== "";
  const factsOK = radio("layout") === "compact" || checked("show").length > 0;
  const ready = placed && factsOK;
  $("#missing").textContent = !placed
    ? "Fill in the time zone and the coordinates to see the card."
    : "Pick at least one fact, or use the compact layout for the greeting alone.";
  $("#missing").hidden = ready;
  $("#colors").hidden = !$("#custom").checked;

  const theme = radio("theme");
  const alt = "A greeting for the time of day where I am";
  let light, dark, snippet;
  if (theme === "github") {
    light = cardURL(query("light"));
    dark = cardURL(query("dark"));
    snippet = `<picture>\n  <source media="(prefers-color-scheme: dark)" srcset="${htmlAttr(dark)}">\n  <img alt="${alt}" src="${htmlAttr(light)}">\n</picture>`;
  } else {
    light = dark = cardURL(query(theme));
    snippet = `<img alt="${alt}" src="${htmlAttr(light)}">`;
  }
  $("#snippet").value = ready ? snippet : "";
  for (const [id, url] of [["#preview-light", light], ["#preview-dark", dark]]) {
    const img = $(id);
    img.hidden = !ready;
    if (ready && img.dataset.src !== url) {
      img.dataset.src = url;
      img.src = url;
      img.alt = alt;
    }
  }

  const warn = $("#contrast");
  warn.hidden = true;
  if ($("#custom").checked) {
    const [bg, accent, text] = ["bg", "accent", "text"].map((id) => $(`#${id}`).value.slice(1));
    const problems = [];
    if (contrast(accent, bg) < 3) problems.push(`accent on background is ${contrast(accent, bg).toFixed(1)}:1 (aim for 3:1)`);
    if (contrast(text, bg) < 4.5) problems.push(`text on background is ${contrast(text, bg).toFixed(1)}:1 (aim for 4.5:1)`);
    if (problems.length) {
      warn.textContent = `Hard to read: ${problems.join("; ")}.`;
      warn.hidden = false;
    }
  }
}

function setup() {
  const zones = typeof Intl.supportedValuesOf === "function" ? Intl.supportedValuesOf("timeZone") : [];
  const list = $("#zones");
  for (const zone of zones) list.append(new Option(zone));
  $("#tz").value = Intl.DateTimeFormat().resolvedOptions().timeZone || "UTC";

  $("#locate").addEventListener("click", () => {
    if (!navigator.geolocation) return;
    navigator.geolocation.getCurrentPosition((pos) => {
      $("#lat").value = roundTo(pos.coords.latitude, 0.1).toFixed(1);
      $("#lon").value = roundTo(pos.coords.longitude, 0.1).toFixed(1);
      update();
    }, () => {
      $("#missing").textContent = "The browser did not share a location. Type your city's coordinates instead.";
    }, { enableHighAccuracy: false, maximumAge: 600000, timeout: 10000 });
  });

  $("#builder").addEventListener("input", update);
  $("#builder").addEventListener("change", update);
  for (const input of document.querySelectorAll('input[name="layout"]')) {
    input.addEventListener("change", () => { $("#radius").value = input.value === "compact" ? 24 : 12; update(); });
  }
  $("#copy").addEventListener("click", async () => {
    try {
      await navigator.clipboard.writeText($("#snippet").value);
      $("#copied").textContent = "Copied.";
    } catch {
      $("#snippet").select();
      $("#copied").textContent = "Select the text and copy it.";
    }
  });
  update();
}

setup();
