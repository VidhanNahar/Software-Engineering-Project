import os
import re

files = [
    "frontend/src/app/pages/Dashboard.tsx",
    "frontend/src/app/pages/MarketOverview.tsx",
    "frontend/src/app/pages/Portfolio.tsx",
    "frontend/src/app/pages/Trade.tsx"
]

for file in files:
    with open(file, 'r') as f:
        content = f.read()
    
    def replacer(match):
        head = match.group(1)
        incoming = match.group(2)
        return incoming.replace("$", "₹")
        
    pattern = re.compile(r'<<<<<<< [^\n]+\n(.*?)\n=======\n(.*?)\n>>>>>>> [^\n]+', re.DOTALL)
    
    new_content = pattern.sub(replacer, content)
    
    with open(file, 'w') as f:
        f.write(new_content)

