---
name: calculator
description: Performs advanced mathematical calculations including trigonometry, logarithms, and base conversion. Use when user needs to calculate something or convert number bases.
---

# Calculator Skill

Advanced mathematical calculator with functions and unit conversion.

## Parameters

- `action` (string, optional): Action - "calculate", "convert", or "help"
- `expression` (string, optional): Math expression for calculate action
- `value` (string, optional): Number to convert
- `from_base` (number, optional): Source base (2, 8, 10, 16)
- `to_base` (number, optional): Target base (2, 8, 10, 16)

## Functions

- Basic: +, -, *, /, ** (power), % (modulo)
- Math: sqrt, cbrt, exp, log, log10, log2
- Trigonometry: sin, cos, tan, asin, acos, atan
- Constants: pi, e, tau

## Example

```json
{"action": "calculate", "expression": "sqrt(16) + factorial(3)"}
{"action": "convert", "value": "255", "from_base": 10, "to_base": 16}
{"action": "help"}
```
