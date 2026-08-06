#!/usr/bin/env python3
"""
detect_component_edges.py

An edge detection and bounding box measurement script for verifying UI component
dimensions, bounding box equality, and alignment across UI screenshots.
"""

import sys
import json
import argparse
import numpy as np
import cv2
from PIL import Image

def detect_ui_bounding_boxes(image_path, min_area=300, max_area=500000, canny_thresh1=50, canny_thresh2=150):
    """Loads an image, applies Canny edge detection and contour extraction,
    and returns detected UI element bounding boxes with dimension metrics.
    """
    img = cv2.imread(image_path)
    if img is None:
        raise ValueError(f"Unable to load image at path: {image_path}")

    gray = cv2.cvtColor(img, cv2.COLOR_BGR2GRAY)
    blurred = cv2.GaussianBlur(gray, (5, 5), 0)
    edges = cv2.Canny(blurred, canny_thresh1, canny_thresh2)

    contours, _ = cv2.findContours(edges, cv2.RETR_EXTERNAL, cv2.CHAIN_APPROX_SIMPLE)

    boxes = []
    output_img = img.copy()

    for idx, cnt in enumerate(contours):
        area = cv2.contourArea(cnt)
        if min_area <= area <= max_area:
            x, y, w, h = cv2.boundingRect(cnt)
            boxes.append({
                "id": idx + 1,
                "x": x,
                "y": y,
                "width": w,
                "height": h,
                "area": float(area),
                "aspect_ratio": round(w / float(h), 3) if h > 0 else 0
            })

            # Draw bounding box and label text
            cv2.rectangle(output_img, (x, y), (x + w, y + h), (0, 215, 255), 2)
            cv2.putText(
                output_img,
                f"{w}x{h}px",
                (x, max(0, y - 5)),
                cv2.FONT_HERSHEY_SIMPLEX,
                0.4,
                (0, 215, 255),
                1
            )

    # Sort boxes by Y position (row) then X position
    boxes = sorted(boxes, key=lambda b: (b['y'] // 20, b['x']))

    widths = [b['width'] for b in boxes]
    heights = [b['height'] for b in boxes]

    equal_widths = len(set(widths)) <= 1 if widths else True
    equal_heights = len(set(heights)) <= 1 if heights else True

    summary = {
        "image_path": image_path,
        "total_components_detected": len(boxes),
        "equal_widths": equal_widths,
        "equal_heights": equal_heights,
        "bounding_boxes": boxes
    }

    return summary, output_img

def main():
    parser = argparse.ArgumentParser(description="UI Component Edge Detection & Size Verification")
    parser.add_argument("image_path", help="Path to UI screenshot image")
    parser.add_argument("--output", help="Path to save annotated image with bounding boxes", default="detected_edges.png")
    parser.add_argument("--min-area", type=int, default=300, help="Minimum contour area")
    parser.add_argument("--max-area", type=int, default=500000, help="Maximum contour area")
    args = parser.parse_args()

    try:
        summary, output_img = detect_ui_bounding_boxes(
            args.image_path,
            min_area=args.min_area,
            max_area=args.max_area
        )

        if args.output:
            cv2.imwrite(args.output, output_img)
            summary["annotated_image_saved"] = args.output

        print(json.dumps(summary, indent=2))

    except Exception as e:
        print(json.dumps({"error": str(e)}), file=sys.stderr)
        sys.exit(1)

if __name__ == "__main__":
    main()
