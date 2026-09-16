BINARY_NAME := wptui
CMD_DIR := ./cmd/wptui
BUILD_DIR := bin

# Detect OS
ifeq ($(OS),Windows_NT)
    DETECTED_OS := Windows
    BINARY_EXT := .exe
else
    DETECTED_OS := $(shell uname -s)
    BINARY_EXT :=
endif

TARGET := $(BUILD_DIR)/$(BINARY_NAME)$(BINARY_EXT)

.PHONY: all build run test test-race vet fmt clean help

all: build

$(BUILD_DIR):
	@mkdir -p $(BUILD_DIR)

## build: Biên dịch binary cho hệ điều hành hiện tại
build: $(BUILD_DIR)
	go build -o $(TARGET) $(CMD_DIR)
	@echo Built $(TARGET) for $(DETECTED_OS)

## run: Biên dịch và khởi chạy wptui
run: build
	./$(TARGET)

## test: Chạy toàn bộ test suites
test:
	go test -v ./...

## test-race: Chạy toàn bộ test suites với race detector
test-race:
	go test -v -race ./...

## vet: Chạy go vet kiểm tra code
vet:
	go vet ./...

## fmt: Format toàn bộ mã nguồn
fmt:
	go fmt ./...

## clean: Dọn dẹp binary được tạo trong thư mục build
clean:
	rm -f $(BUILD_DIR)/*
	@echo Cleaned $(BUILD_DIR)

## help: Hướng dẫn sử dụng
help:
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@echo "  build       Biên dịch binary cho $(DETECTED_OS) ($(TARGET))"
	@echo "  run         Biên dịch và khởi chạy ứng dụng"
	@echo "  test        Chạy go test"
	@echo "  test-race   Chạy go test với -race"
	@echo "  vet         Chạy go vet"
	@echo "  fmt         Chạy go fmt"
	@echo "  clean       Xóa các file binary trong thư mục build ($(BUILD_DIR))"
