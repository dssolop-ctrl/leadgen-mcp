import json, sys
sys.stdout.reconfigure(encoding='utf-8')

base = r'C:\Users\animo\.claude\projects\C--git-leadgen-mcp\b4464b72-ca67-4177-93b9-170b35bfbc29\tool-results'

# Goals
with open(f'{base}\\toolu_01XyKAkMLQHLs5jgfZUAFhcf.json', 'r', encoding='utf-8') as f:
    data = json.load(f)
goals = json.loads(data[0]['text'])['goals']
target = ['общая', 'все лиды', 'все звонки', 'коллтрекинг', 'реальный звонок']
print("=== GOALS (Chelyabinsk) ===")
for g in goals:
    name = g['name']
    if any(t in name.lower() for t in target):
        print(f"  ID: {g['id']:>12} | Type: {g['type']:>20} | Name: {name}")
print(f"--- Total goals: {len(goals)}")

# Clients
print("\n=== AGENCY CLIENTS ===")
with open(f'{base}\\toolu_01HXNahR3B5ooXg8imJw3EzU.json', 'r', encoding='utf-8') as f:
    data = json.load(f)
clients = json.loads(data[0]['text'])['Clients']
for c in clients:
    print(f"  {c['Login']:>25} | {c.get('ClientInfo', '')}")
print(f"--- Total clients: {len(clients)}")
