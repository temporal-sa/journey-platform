---
name: verify-component-sizes
description: Verifies UI component dimensions, bounding box size equality, padding alignment, and visual edge contours using image edge detection and computer vision scripts.
---

# Verify UI Component Sizes & Edge Alignment

This skill provides step-by-step instructions and automated edge-detection tooling to measure UI component bounding boxes, compare element widths and heights, and verify size equality across screenshots or rendered DOM elements.

---

## Overview

When building or refining user interfaces, visual components (e.g. `Previous` vs. `Next` pagination buttons, toolbar icons, modal actions) must often maintain identical dimensions `(width x height)`.

The bundled script `detect_component_edges.py` uses OpenCV Canny edge detection and contour extraction to measure bounding box dimensions directly from UI screenshots.

---

## Usage Instructions

### 1. Capture UI Screenshot
Capture a screenshot of the target UI components (or crop the specific bounding area):
```bash
screencapture -R<x,y,w,h> /tmp/component_screenshot.png
```

### 2. Execute Edge Detection Script
Run `detect_component_edges.py` to analyze bounding boxes and generate annotated output:

```bash
python3 .agents/skills/verify-component-sizes/scripts/detect_component_edges.py /tmp/component_screenshot.png --output /tmp/component_edges_annotated.png
```

### 3. Output Format
The script returns a JSON summary containing:
- `total_components_detected`: Count of UI elements found.
- `equal_widths`: Boolean indicating whether all detected target components share identical widths.
- `equal_heights`: Boolean indicating whether all detected target components share identical heights.
- `bounding_boxes`: Array of detected bounding boxes with exact `width`, `height`, `x`, `y`, `area`, and `aspect_ratio`.

---

## Verification Criteria
- **Equal Dimensions**: Confirm `equal_widths: true` and `equal_heights: true` for target paired elements.
- **Min-Width Enforcement**: Verify Tailwind CSS classes such as `min-w-[128px] w-32` prevent content-driven dimension drift.
