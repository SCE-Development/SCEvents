# Paths follow this Makefile location (not cwd), so you can run:
#   make -f testing/Makefile integration
# from the repo root, or from anywhere via absolute -f path.
_TESTING_ABS := $(abspath $(dir $(lastword $(MAKEFILE_LIST))))
REPO_ROOT     := $(abspath $(_TESTING_ABS)/..)
COMPOSE_BASE     := $(REPO_ROOT)/docker-compose.dev.yml
COMPOSE_INT      := $(REPO_ROOT)/testing/integration/docker-compose.integration.yml
COMPOSE_LOAD     := $(REPO_ROOT)/testing/load/docker-compose.load.yml
INTEGRATION_DIR  := $(REPO_ROOT)/testing/integration
LOAD_DIR         := $(REPO_ROOT)/testing/load

# Default to running all integration tests if no filter is provided
TEST ?= all

.PHONY: integration load teardown teardown-integration teardown-load _up_integration _up_load

# Tear down Compose stacks (absolute -f paths; works from any cwd).
teardown-integration:
	docker compose --project-directory "$(REPO_ROOT)" -f $(COMPOSE_BASE) -f $(COMPOSE_INT) down

teardown-load:
	docker compose --project-directory "$(REPO_ROOT)" -f $(COMPOSE_BASE) -f $(COMPOSE_LOAD) down

teardown: teardown-integration teardown-load

# Run integration tests.
# Usage:
#   make integration                  (runs full suite)
#   make integration TEST=register    (registration flow only)
#   make integration TEST=create      (event creation flow only)
integration: _up_integration
ifeq ($(TEST),register)
	cd $(INTEGRATION_DIR) && go test -tags=integration -v -run TestRegistration ./...
else ifeq ($(TEST),create)
	cd $(INTEGRATION_DIR) && go test -tags=integration -v -run TestEventCreation ./...
else
	cd $(INTEGRATION_DIR) && go test -tags=integration -v ./...
endif

# Flood the registration endpoint with concurrent requests via k6
load: _up_load
	k6 run $(LOAD_DIR)/register.js

# ── internal targets ──────────────────────────────────────────────────────────

# Spin up the stack with mock Clark for integration tests (leaves containers running)
_up_integration:
	docker compose \
		--project-directory "$(REPO_ROOT)" \
		-f $(COMPOSE_BASE) \
		-f $(COMPOSE_INT) \
		up -d --build

# Spin up the stack with mock Clark for load tests (leaves containers running)
_up_load:
	docker compose \
		--project-directory "$(REPO_ROOT)" \
		-f $(COMPOSE_BASE) \
		-f $(COMPOSE_LOAD) \
		up -d --build
