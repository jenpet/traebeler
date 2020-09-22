# Froxlor Processor
The Froxlor processor takes a given list of domains retrieved from the traefik API and registers them at a configured Froxlor instance using its API. It is individually configurable via environment variables.

The domain list is processed by utilizing bobesa/go-domain-util/domainutil which makes it easy to split up the top level domain and the actual sub domain for the DNS record.

## Env Var Configuration

All the environment variables are empty by default, so they have to be set in order for the Froxlor processor to work. 

ENV VAR |  DESCRIPTION
---| ---
TRAEBELER_PROCESSOR_FROXLOR_URI | base URI of the froxlor instance (without trailing slashes `/`, without path)
TRAEBELER_PROCESSOR_FROXLOR_KEY | API key of the user which should be used
TRAEBELER_PROCESSOR_FROXLOR_SECRET | API secret of the user which should be used


## Open Features
- Arbitrary cache eviction to be sure everything is still in sync with the actual froxlor API
- Logging improvements, only log diffs 