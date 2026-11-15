"""Can local signals pick what the LLM filter picked? Simulated on recall3's
union pools (top 20 ungated candidates per message). Fit on development,
report on heldout. Unjudged picks are counted separately."""
import json,hashlib,re,math,random,collections,os
R3=os.environ['JCODE_SCRATCH_DIR']+'/recall3'
msgs={m['id']:m for m in json.load(open(R3+'/replay.json'))}
J=json.load(open(os.environ['JCODE_SCRATCH_DIR']+'/recall4/real/judgments.json'))
k=lambda i,t:hashlib.sha256((str(i)+'\0'+t).encode()).hexdigest()[:20]
pools={}
for s in ('development','heldout'):
    for l in open(f'{R3}/union-{s}.jsonl'):
        r=json.loads(l); m=msgs[r['id']]; c=[]
        for p,h in enumerate(r['response'].get('hits') or []):
            w=h.get('why',''); g=lambda pat:float(re.search(pat,w).group(1)) if re.search(pat,w) else 0.0
            j=J.get(k(r['id'],h['text']))
            c.append(dict(text=h['text'],rank=p,total=h['score'],sem=g(r'semantic=([\d.]+)'),cov=g(r'coverage=([\d.]+)'),kw=g(r'keyword=(\d+)'),
                own=float(m['instance'].lower() in h['scope'].lower()),loglen=math.log(1+len(h['text'])),rel=None if j is None else bool(j['relevant'])))
        pools[r['id']]=c
def base(i):
    rows=[]
    for s in ('development','heldout'):
        pass
F=['sem','own','loglen','cov','kw1','rankf','long']
def feats(c,i):
    return [c['sem'],c['own'],c['loglen']/8,c['cov'],min(c['kw'],60)/60,c['rank']/20,float(msgs[i]['stratum']=='long')]
# logistic regression, plain SGD
def fit(ids,epochs=300,lr=0.3,l2=1e-3):
    X=[(feats(c,i),c['rel']) for i in ids for c in pools[i] if c['rel'] is not None]
    w=[0.0]*(len(F)+1)
    for _ in range(epochs):
        g=[0.0]*len(w)
        for x,y in X:
            z=w[-1]+sum(a*b for a,b in zip(w,x)); p=1/(1+math.exp(-z)); e=p-y
            for j,v in enumerate(x): g[j]+=e*v
            g[-1]+=e
        w=[wi-lr*(gi/len(X)+l2*wi) for wi,gi in zip(w,g)]
    return w
def score(w,c,i): return 1/(1+math.exp(-(w[-1]+sum(a*b for a,b in zip(w,feats(c,i))))))
def evaluate(pick,ids):
    rel=irr=unk=empty=0; seen=collections.defaultdict(set)
    for i in sorted(ids, key=lambda i: msgs[i]['timestamp']):
        s=msgs[i]['session']; got=[c for c in pick(i) if c['text'] not in seen[s]][:5]
        for c in got: seen[s].add(c['text'])
        empty+=not got
        for c in got:
            if c['rel'] is None: unk+=1
            elif c['rel']: rel+=1
            else: irr+=1
    n=len(ids); return f"rel/msg={rel/n:.2f} irr/msg={irr/n:.2f} unjudged/msg={unk/n:.2f} empty={empty/n:.2f}"
dev=[i for i in pools if msgs[i]['split']=='development']; held=[i for i in pools if msgs[i]['split']=='heldout']
w=fit(dev)
print('weights',dict(zip(F+['bias'],[round(x,2) for x in w])))
def llm(i):
    return None
for split,ids in (('development',dev),('heldout',held)):
    for st in ('human','long'):
        sub=[i for i in ids if msgs[i]['stratum']==st]
        print(f'== {split} {st} n={len(sub)}')
        for th in (0.35,0.45,0.55):
            print(f'  local p>={th}:', evaluate(lambda i:[c for c in sorted(pools[i],key=lambda c:-score(w,c,i)) if score(w,c,i)>=th],sub))

print('\n# simple rules on the same pools (no fitting)')
def rule(min_sem,maxlen,own_only):
    return lambda i:[c for c in sorted(pools[i],key=lambda c:-c['sem']) if c['sem']>=min_sem and len(c['text'])<=maxlen and (c['own'] or not own_only)]
B={}
for s in ('development','heldout'):
    for l in open(f'{R3}/baseline-{s}.jsonl'):
        r=json.loads(l); B[r['id']]=[dict(text=h['text'],rel=(J.get(k(r['id'],h['text'])) or {}).get('relevant')) for h in r['response'].get('hits') or []]
for split,ids in (('development',dev),('heldout',held)):
    for st in ('human','long'):
        sub=[i for i in ids if msgs[i]['stratum']==st]
        print(f'== {split} {st} n={len(sub)}')
        print('  current recall      ', evaluate(lambda i:B[i],sub))
        for ms,ml,oo in ((0.60,2000,True),(0.64,2000,True),(0.64,10**7,True),(0.60,600,True),(0.68,2000,True)):
            print(f'  sem>={ms} len<={ml if ml<10**7 else "any":>4} own={oo}', evaluate(rule(ms,ml,oo),sub))
