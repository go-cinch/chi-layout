.PHONY: lint test default full

SCAFFOLD ?= scaffold
PROJECT ?= chi-service
OUTPUT_DIR ?= ..

lint:
	$(SCAFFOLD) lint scaffold.yml

test: lint
	./scripts/test-template.sh

default:
	$(SCAFFOLD) new "$(CURDIR)" --output-dir="$(OUTPUT_DIR)" --run-hooks=always --no-prompt --preset=default "Project=$(PROJECT)"

full:
	$(SCAFFOLD) new "$(CURDIR)" --output-dir="$(OUTPUT_DIR)" --run-hooks=always --no-prompt --preset=full "Project=$(PROJECT)"
