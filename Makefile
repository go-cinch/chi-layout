.PHONY: lint test default full

SCAFFOLD ?= scaffold
PROJECT ?= chi-service
OUTPUT_DIR ?= ..
HTTP_ROUTER ?= chi
ENABLE_GRPC ?= true

ifneq ($(ENABLE_GRPC),true)
ifneq ($(ENABLE_GRPC),false)
$(error ENABLE_GRPC must be true or false)
endif
endif

ifneq ($(HTTP_ROUTER),chi)
ifneq ($(HTTP_ROUTER),gin)
$(error HTTP_ROUTER must be chi or gin)
endif
endif

lint:
	$(SCAFFOLD) lint scaffold.yml

test: lint
	./scripts/test-template.sh

default:
	$(SCAFFOLD) new "$(CURDIR)" --output-dir="$(OUTPUT_DIR)" --run-hooks=always --no-prompt --preset=default "Project=$(PROJECT)" "http_router=$(HTTP_ROUTER)" "enable_grpc=$(ENABLE_GRPC)"

full:
	$(SCAFFOLD) new "$(CURDIR)" --output-dir="$(OUTPUT_DIR)" --run-hooks=always --no-prompt --preset=full "Project=$(PROJECT)" "http_router=$(HTTP_ROUTER)" "enable_grpc=$(ENABLE_GRPC)"
