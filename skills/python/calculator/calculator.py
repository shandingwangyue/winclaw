#!/usr/bin/env python3
"""
Calculator Skill for WinClaw
Advanced calculator with math functions
"""

import sys
import json
import math
import cmath
import re


def safe_eval(expr):
    allowed_names = {
        "abs": abs,
        "round": round,
        "min": min,
        "max": max,
        "sum": sum,
        "pow": pow,
        "sqrt": math.sqrt,
        "cbrt": lambda x: x ** (1 / 3),
        "exp": math.exp,
        "log": math.log,
        "log10": math.log10,
        "log2": math.log2,
        "sin": math.sin,
        "cos": math.cos,
        "tan": math.tan,
        "asin": math.asin,
        "acos": math.acos,
        "atan": math.atan,
        "sinh": math.sinh,
        "cosh": math.cosh,
        "tanh": math.tanh,
        "degrees": math.degrees,
        "radians": math.radians,
        "factorial": math.factorial,
        "gcd": math.gcd,
        "floor": math.floor,
        "ceil": math.ceil,
        "pi": math.pi,
        "e": math.e,
        "tau": math.tau,
        "inf": math.inf,
    }

    expr = expr.replace("^", "**")
    expr = re.sub(r"(\d+)\s*!", r"math.factorial(\1)", expr)
    expr = re.sub(r"sqrt(\d+)", r"math.sqrt(\1)", expr)

    try:
        result = eval(expr, {"__builtins__": {}}, allowed_names)
        return result
    except Exception as e:
        return f"Error: {str(e)}"


def calculate(expression):
    result = safe_eval(expression)
    return result


def convert_number(value, from_base, to_base):
    try:
        if from_base == to_base:
            return value

        decimal = int(value, from_base)

        if to_base == 10:
            return str(decimal)
        elif to_base in (2, 8, 16):
            if decimal == 0:
                return "0"
            digits = "0123456789ABCDEF"
            result = ""
            while decimal > 0:
                result = digits[decimal % to_base] + result
                decimal //= to_base
            return result

        return str(decimal)
    except Exception as e:
        return f"Error: {str(e)}"


def main():
    if len(sys.argv) < 2:
        print(json.dumps({"error": "No parameters provided"}))
        return

    try:
        params = json.loads(sys.argv[1])
    except:
        params = {}

    action = params.get("action", "calculate")

    if action == "calculate":
        expression = params.get("expression", "")
        if not expression:
            print(json.dumps({"error": "No expression provided"}))
            return

        result = calculate(expression)
        print(json.dumps({"expression": expression, "result": str(result)}))

    elif action == "convert":
        value = params.get("value", "")
        from_base = params.get("from_base", 10)
        to_base = params.get("to_base", 10)

        if not value:
            print(json.dumps({"error": "No value provided"}))
            return

        result = convert_number(value, from_base, to_base)
        print(
            json.dumps(
                {
                    "value": value,
                    "from_base": from_base,
                    "to_base": to_base,
                    "result": result,
                }
            )
        )

    elif action == "help":
        help_text = """
Available functions:
- Basic: +, -, *, /, ** (power), % (modulo)
- Math: sqrt, cbrt, exp, log, log10, log2
- Trigonometry: sin, cos, tan, asin, acos, atan
- Hyperbolic: sinh, cosh, tanh
- Constants: pi, e, tau
- Other: factorial(n), abs, round, floor, ceil, gcd
- Examples: 2**10, sqrt(16), sin(pi/2), factorial(5)
"""
        print(json.dumps({"help": help_text}))

    else:
        print(json.dumps({"error": f"Unknown action: {action}"}))


if __name__ == "__main__":
    main()
