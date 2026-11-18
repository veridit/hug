| set | stratum | n | sessions | arm | rel/msg | irr/msg | unjudged | rel delta [95% CI] | irr delta [95% CI] |
|---|---|---:|---:|---|---:|---:|---:|---:|---:|
| frozen | human | 124 | 10 | shipped | 0.50 | 0.21 | 0 | - | - |
| frozen | human | 124 | 10 | rule60 | 1.23 | 1.03 | 0 | +0.73 [+0.48, +0.99] | +0.82 [+0.48, +1.42] |
| frozen | human | 124 | 10 | rule64 | 0.97 | 0.57 | 0 | +0.47 [+0.30, +0.61] | +0.36 [+0.06, +0.88] |
| frozen | human | 124 | 10 | rule68 | 0.65 | 0.24 | 0 | +0.15 [-0.10, +0.35] | +0.03 [-0.18, +0.34] |
| frozen | long | 153 | 53 | shipped | 0.92 | 1.02 | 0 | - | - |
| frozen | long | 153 | 53 | rule60 | 1.05 | 1.73 | 0 | +0.14 [-0.07, +0.34] | +0.71 [+0.25, +1.12] |
| frozen | long | 153 | 53 | rule64 | 1.01 | 1.62 | 0 | +0.10 [-0.10, +0.29] | +0.60 [+0.19, +0.99] |
| frozen | long | 153 | 53 | rule68 | 0.85 | 1.37 | 0 | -0.07 [-0.21, +0.09] | +0.35 [-0.04, +0.72] |
| fresh | human | 83 | 6 | shipped | 0.28 | 0.33 | 0 | - | - |
| fresh | human | 83 | 6 | rule60 | 0.39 | 0.58 | 0 | +0.11 [-0.22, +0.50] | +0.25 [-0.46, +0.93] |
| fresh | human | 83 | 6 | rule64 | 0.20 | 0.20 | 0 | -0.07 [-0.26, +0.11] | -0.12 [-0.50, +0.18] |
| fresh | human | 83 | 6 | rule68 | 0.12 | 0.01 | 0 | -0.16 [-0.27, -0.01] | -0.31 [-0.51, -0.18] |
| fresh | long | 64 | 62 | shipped | 0.38 | 3.38 | 0 | - | - |
| fresh | long | 64 | 62 | rule60 | 0.44 | 2.75 | 0 | +0.06 [-0.24, +0.34] | -0.62 [-0.95, -0.24] |
| fresh | long | 64 | 62 | rule64 | 0.44 | 2.50 | 0 | +0.06 [-0.24, +0.34] | -0.88 [-1.23, -0.48] |
| fresh | long | 64 | 62 | rule68 | 0.36 | 2.08 | 0 | -0.02 [-0.30, +0.26] | -1.30 [-1.67, -0.90] |

rule60: frozen irr upper CI < 0 on both strata: False; rel not worse (upper CI >= 0) on every bar split: True; PASS=False
rule64: frozen irr upper CI < 0 on both strata: False; rel not worse (upper CI >= 0) on every bar split: True; PASS=False
rule68: frozen irr upper CI < 0 on both strata: False; rel not worse (upper CI >= 0) on every bar split: True; PASS=False
