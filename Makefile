# Forward integration/load targets to testing/Makefile so paths stay correct.
# From repo root: `make integration`, `make load`, `make integration TEST=register`, etc.

REPO_ROOT := $(abspath $(dir $(lastword $(MAKEFILE_LIST))))

.PHONY: integration load teardown teardown-integration teardown-load

integration load teardown teardown-integration teardown-load:
	@$(MAKE) -f "$(REPO_ROOT)/testing/Makefile" $(MAKECMDGOALS)
