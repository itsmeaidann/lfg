# LFG

Lighting Fast Go


## Dry run 🥩

setup `lfg.yaml`
```yaml
exchange:
    melaniaBnf: # id for application reference
        exchange: bnf # must be `bnf` for binance future exchange
        envPrefix: BNF # prefix matching with .env settings
        subAccountId: 0 # exchange subaccount
agent:
    melania: 
        exchange:
        - "melaniaBnf"
        melania: "Open a long position with $500 USD if the 9/26 moving average crosses upwards on a 15-minute timeframe, and open a short position with $500 USD if the moving average crosses downwards."
```

setup `.env`

```
ENVIRONMENT=local
BNF_API_KEY=... # API key, matching the BNF prefix in .env
BNF_API_SECRET=... # API secret, matching the BNF prefix in .env
```

run

```
go run cmd/main.go
```
