from decimal import *

expr = "${some_value} ${some_value_2}"
print(expr)
print(expr.strip("${"))
expr = expr.strip()
print(expr)
expr = expr.lstrip("${")
print(expr)
expr = expr.rstrip("}")
print(expr)
pos = expr.find('.')
print(pos)
print(Decimal(0))