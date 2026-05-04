# AWS Connect Resource Dependencies Diagrams

This directory contains D2 diagrams showing AWS Connect resource dependencies.

## Files

- `overview.d2` - Source D2 diagram file
- `overview.svg` - Rendered SVG output
- `overview.png` - Rendered PNG output

## Overview Diagram

The overview diagram shows:
- **Wisdom Service** - Amazon Connect Wisdom with Assistant and Knowledge Base resources
- **Connect Service** - Amazon Connect Instance
- **Dependencies** - How resources relate to each other
- **Legend** - Visual guide for interpreting the diagram

## Updating the Diagram

### Prerequisites

Install D2:
```bash
brew install d2
```

### Edit and Render

1. Edit the source file:
```bash
vi overview.d2
```

2. Render to SVG and PNG:
```bash
d2 overview.d2 overview.svg
d2 overview.d2 overview.png
```

Or use a single command:
```bash
d2 overview.d2 overview.svg && d2 overview.d2 overview.png
```

### Live Preview

For live editing with auto-reload:
```bash
d2 --watch overview.d2 overview.svg
```

Then open `overview.svg` in a browser.

## D2 Syntax Examples

### Basic Shape
```d2
wisdom: {
  label: "Amazon Connect Wisdom"
  shape: rectangle
}
```

### Nested Resource
```d2
wisdom: {
  assistant: {
    label: "Wisdom Assistant"
    shape: cylinder
  }
}
```

### Dependency
```d2
assistant -> knowledge_base: {
  label: "associates"
}
```

### Styling
```d2
resource: {
  style: {
    fill: "#4A90E2"
    stroke: "#FF9900"
    stroke-width: 2
  }
}
```

## Reference

- D2 Documentation: https://d2lang.com/
- D2 Tour: https://d2lang.com/tour/intro/
- Shape Reference: https://d2lang.com/tour/shapes/
