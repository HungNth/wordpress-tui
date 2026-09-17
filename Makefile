BINARY_NAME := wptui
CMD_DIR := ./cmd/wptui
BUILD_DIR := bin

# Detect OS and set OS-specific commands
ifeq ($(OS),Windows_NT)
    DETECTED_OS := Windows
    BINARY_EXT := .exe

    # Force GNU Make to use Windows cmd.exe
    SHELL := cmd.exe
    .SHELLFLAGS := /C

    MKDIR_CMD := if not exist "$(BUILD_DIR)" mkdir "$(BUILD_DIR)"
    CLEAN_CMD := if exist "$(BUILD_DIR)" rmdir /S /Q "$(BUILD_DIR)"
else
    DETECTED_OS := $(shell uname -s)
    BINARY_EXT :=
    MKDIR_CMD := mkdir -p "$(BUILD_DIR)"
    CLEAN_CMD := rm -rf "$(BUILD_DIR)"
endif

TARGET := $(BUILD_DIR)/$(BINARY_NAME)$(BINARY_EXT)

# Command used to run the binary
ifeq ($(OS),Windows_NT)
    RUN_CMD := "$(TARGET)"
else
    RUN_CMD := ./$(TARGET)
endif

.PHONY: all build run test test-race vet fmt clean help

all: build

$(BUILD_DIR):
	@$(MKDIR_CMD)

## build: Build the binary for the current operating system
build: $(BUILD_DIR)
	go build -o "$(TARGET)" $(CMD_DIR)
	@echo Built $(TARGET) for $(DETECTED_OS)

## run: Build and run wptui
run: build
	@$(RUN_CMD)

## test: Run all test suites
test:
	go test -v ./...

## test-race: Run all test suites with the race detector
test-race:
	go test -v -race ./...

## vet: Run go vet to check the code
vet:
	go vet ./...

## fmt: Format all source code
fmt:
	go fmt ./...

## clean: Remove the build directory using the appropriate OS command
clean:
	@$(CLEAN_CMD)
	@echo Cleaned $(BUILD_DIR) for $(DETECTED_OS)

## help: Show usage information
help:
	@echo Usage: make [target]
	@echo.
	@echo Targets:
	@echo   build       Build the binary for $(DETECTED_OS) ($(TARGET))
	@echo   run         Build and run the application
	@echo   test        Run go test
	@echo   test-race   Run go test with -race
	@echo   vet         Run go vet
	@echo   fmt         Run go fmt
	@echo   clean       Remove the build directory ($(BUILD_DIR)) for $(DETECTED_OS)
