.PHONY: list-tests
list-tests:
	RUN_TESTBED=1 go test -v ./tests --test.list '.*' | grep "^Test"

.PHONY: run-tests
run-tests:
	RUN_TESTBED=1 go test -v ./tests -timeout 30m

.PHONY: run-gigwarm-tests
run-gigwarm-tests:
	RUN_TESTBED=1 go test -v ./tests -run TestGigWarm -timeout 30m

.PHONY: run-trace-tests
run-trace-tests:
	RUN_TESTBED=1 go test -v ./tests -run TestGigWarmTrace -timeout 30m

.PHONY: run-log-tests
run-log-tests:
	RUN_TESTBED=1 go test -v ./tests -run TestGigWarmLog -timeout 30m

.PHONY: run-high-throughput-test
run-high-throughput-test:
	RUN_TESTBED=1 go test -v ./tests -run TestGigWarmHighThroughput -timeout 60m

.PHONY: clean
clean:
	rm -rf ./tests/results

.PHONY: help
help:
	@echo "Available targets:"
	@echo "  list-tests              - List all available load tests"
	@echo "  run-tests               - Run all load tests"
	@echo "  run-gigwarm-tests       - Run all GigWarm exporter tests"
	@echo "  run-trace-tests         - Run trace-specific load tests"
	@echo "  run-log-tests           - Run log-specific load tests"
	@echo "  run-high-throughput-test - Run high throughput stress test"
	@echo "  clean                   - Clean test results"
