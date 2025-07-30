SHELL := /bin/bash

GO_URL=https://go.dev/dl/
GO_INSTALL_DIR=/usr/local
GO_CURRENT_VERSION=$(shell if command -v go >/dev/null 2>&1; then go version | awk '{print $$3}' | sed 's/go//'; else echo "none"; fi)
LATEST_GO_VERSION=$(shell curl -s https://go.dev/VERSION?m=text | sed -nE 's/^go([0-9\.]+).*/\1/p')
GO_TARBALL=go$(LATEST_GO_VERSION).linux-amd64.tar.gz

GOPATH=$(HOME)/go
ESCAPED_GOPATH=$(shell echo $(GOPATH) | sed 's/\//\\\//g')
GOVULNCHECK_BINARY=govulncheck
GOLANGCI_LINT_BINARY=golangci-lint
VULNCHECK_PACKAGE=golang.org/x/vuln/cmd/$(GOVULNCHECK_BINARY)
LINT_PACKAGE=github.com/golangci/golangci-lint/cmd/$(GOLANGCI_LINT_BINARY)

# Module name for `go mod init` and the executable
MODULE_NAME=$(shell basename $(PWD))
LOCAL_BIN=~/.local/bin/
TEMPLATE_REPO_URL := https://github.com/sss7526/go_maker.git

.PHONY: all install update uninstall help .validate_latest \
		govulncheck-install golangci-lint-install tool-install \
		lint vulncheck mod-init mod-tidy mod-update \
		run build ex

all: help

## Default target - show help
help:
	@COLUMNS=$$(tput cols); \
	BORDER=$$(printf '=%.0s' $$(seq 1 $$COLUMNS)); \
	HEADER1=" ██████╗  ██████╗       ███╗   ███╗ █████╗ ██╗  ██╗███████╗██████╗"; \
	HEADER2="██╔════╝ ██╔═══██╗      ████╗ ████║██╔══██╗██║ ██╔╝██╔════╝██╔══██╗"; \
	HEADER3="██║  ███╗██║   ██║█████╗██╔████╔██║███████║█████╔╝ █████╗  ██████╔╝"; \
	HEADER4="██║   ██║██║   ██║╚════╝██║╚██╔╝██║██╔══██║██╔═██╗ ██╔══╝  ██╔══██╗"; \
	HEADER5="╚██████╔╝╚██████╔╝      ██║ ╚═╝ ██║██║  ██║██║  ██╗███████╗██║  ██║"; \
	HEADER6=" ╚═════╝  ╚═════╝       ╚═╝     ╚═╝╚═╝  ╚═╝╚═╝  ╚═╝╚══════╝╚═╝  ╚═╝"; \
	echo ""; \
	echo "$$BORDER"; \
	echo ""; \
	SPACES=$$((($$COLUMNS-$${#HEADER1})/2)); printf "%*s%s\n" $$SPACES "" "$$HEADER1"; \
	SPACES=$$((($$COLUMNS-$${#HEADER2})/2)); printf "%*s%s\n" $$SPACES "" "$$HEADER2"; \
	SPACES=$$((($$COLUMNS-$${#HEADER3})/2)); printf "%*s%s\n" $$SPACES "" "$$HEADER3"; \
	SPACES=$$((($$COLUMNS-$${#HEADER4})/2)); printf "%*s%s\n" $$SPACES "" "$$HEADER4"; \
	SPACES=$$((($$COLUMNS-$${#HEADER5})/2)); printf "%*s%s\n" $$SPACES "" "$$HEADER5"; \
	SPACES=$$((($$COLUMNS-$${#HEADER6})/2)); printf "%*s%s\n" $$SPACES "" "$$HEADER6"; \
	echo ""; \
	echo "$$BORDER"; \
    SPACES=$$((($$COLUMNS-43)/2)); printf "%*s%s\n" $$SPACES "" "COMMANDS FOR GO PROJECT MANAGEMENT"; \
	echo "$$BORDER"; \
	echo "     SOURCE: https://github.com/sss7526/go_maker"; \
	echo ""; \
	echo "  🛠  SETUP AND MAINTENANCE:"; \
	echo "      install       Install the latest Go version (if not installed)."; \
	echo "      update        Update Go to the latest version (if needed)."; \
	echo "      uninstall     Remove the currently installed Go version."; \
	echo ""; \
	echo "  📦 MODULE MANAGEMENT:"; \
	echo "      mod-init      Initialize a new Go module in the current directory."; \
	echo "      mod-tidy      Ensure go.mod and go.sum are in a tidy state."; \
	echo "      mod-update    Update all project dependencies to their latest versions."; \
	echo ""; \
	echo "  🔍 CODE QUALITY & SECURITY:"; \
	echo "      format        Format Go files to a consistent standard (via gofmt)."; \
	echo "      lint          Perform rigorous code linting with golangci-lint."; \
	echo "      vulncheck     Analyze dependencies for vulnerabilities (via govulncheck)."; \
	echo ""; \
	echo "  📖 MISCELLANEOUS:"; \
	echo "      run           Run the project's main entry point (main.go)"; \
	echo "      test          Run all tests recursively across all packages."; \
	echo "      build         Build the project and output the binary to $(LOCAL_BIN)$(MODULE_NAME)."; \
	echo "      ex            Execute the built binary."; \
	echo "      tree          Generate a directory structure summary and save it to tree.txt."; \
	echo "      clean         Remove the compiled binary from $(LOCAL_BIN)."; \
	echo "      help          Display this help screen."; \
	echo ""; \
	echo "$$BORDER"; \
	SPACES=$$((($$COLUMNS-38)/2)); printf "%*s%s\n" $$SPACES "" "INFO: Use 'make <target>' to run a command."; \
	echo "$$BORDER"; \
	echo ""

## Validate Go is installed
.validate_go_installed:
	@if [ "$(GO_CURRENT_VERSION)" = "none" ]; then \
		echo "Error: Go is not installed. Please install Go first using 'make install'."; \
		exit 1; \
	fi

## Validate latest version of Go
.validate_latest:
	@if [ "$(LATEST_GO_VERSION)" = "" ]; then \
		echo "Error: Unable to fetch the latest Go version. Check your internet connection or the Go website."; \
		exit 1; \
	fi
	@if ! echo "$(LATEST_GO_VERSION)" | grep -Eq '^[0-9]+\.[0-9]+\.[0-9]+$$'; then \
		echo "Error: Detected invalid Go version format ('$(LATEST_GO_VERSION)'). Please check the Go website or your internet connection."; \
		exit 1; \
    fi

## Ensure GOPATH and $GOPATH/bin in PATH
.validate_gopath:
	@if [ "$(GOPATH)" = "" ]; then \
		echo "GOPATH is not set. Defaulting to $(HOME)/go."; \
	fi
	@if ! echo $$PATH | grep -q "$(GOPATH)/bin"; then \
		echo "Adding $(GOPATH)/bin to PATH via ~/.bashrc..."; \
		grep -qxF 'export PATH=$${PATH}:$(GOPATH)/bin' ~/.bashrc || echo 'export PATH=$${PATH}:$(GOPATH)/bin' >> ~/.bashrc; \
		echo "updated PATH to include $(GOPATH)/bin. Please run 'source ~/.bashrc' or restart your shell to apply changes."; \
	fi

## Install the latest Go version if not already installed
install: .validate_latest .validate_gopath
	@if [ "$(GO_CURRENT_VERSION)" != "none" ] && [ "$(GO_CURRENT_VERSION)" != "$(LATEST_GO_VERSION)" ]; then \
		echo "An older version of Go ($(GO_CURRENT_VERSION)) is installed."; \
		echo "Please run 'make update' to safely upgrade to Go $(LATEST_GO_VERSION)."; \
		exit 1; \
	elif [ "$(GO_CURRENT_VERSION)" = "$(LATEST_GO_VERSION)" ]; then \
		echo "The latest version of Go ($(LATEST_GO_VERSION)) is already installed. No update required."; \
	else \
		echo "Installing Go $(LATEST_GO_VERSION)..."; \
		curl -OL $(GO_URL)$(GO_TARBALL); \
		sudo tar -C $(GO_INSTALL_DIR) -xzf $(GO_TARBALL); \
		rm $(GO_TARBALL); \
		grep -qxF 'export PATH=$${PATH}:$(GO_INSTALL_DIR)/go/bin' ~/.bashrc || echo 'export PATH=$${PATH}:$(GO_INSTALL_DIR)/go/bin' >> ~/.bashrc; \
		echo "Installation complete. Please run 'source ~/.bashrc' or restart your shell to apply changes."; \
	fi

## Update Go to the latest version (removing previous installation if necessary)
update: .validate_latest .validate_gopath
	@if [ "$(GO_CURRENT_VERSION)" = "$(LATEST_GO_VERSION)" ]; then \
		echo "The latest version of Go ($(LATEST_GO_VERSION)) is already installed. No update required."; \
	elif [ "$(GO_CURRENT_VERSION)" = "none" ]; then \
		echo "No Go version is currently installed. Please run 'make install'"; \
	else \
		$(MAKE) uninstall; \
		$(MAKE) install; \
	fi

## Remove currently installed Go version
uninstall:
	@if [ "$(GO_CURRENT_VERSION)" != "none" ]; then \
		echo "Removing Go $(GO_CURRENT_VERSION)..."; \
		sudo rm -rf $(GO_INSTALL_DIR)/go; \
		sed -i '/go\/bin/d' ~/.bashrc; \
		sed -i '/$(ESCAPED_GOPATH)\/bin/d' ~/.bashrc; \
		echo "Go $(GO_CURRENT_VERSION) has been removed. Please run 'source ~/.bashrc' or restart your shell to apply changes."; \
	else \
		echo "No Go installation found to remove."; \
	fi

## Install golangci-ling
golangci-lint-install: .validate_go_installed
	@if ! command -v $(GOLANGCI_LINT_BINARY) >/dev/null 2>&1; then \
		echo "Installing golangci-lint..."; \
		go install $(LINT_PACKAGE)@latest; \
	else \
		echo "golang-lint already installed."; \
	fi

## Run golangci-ling for linting and static analysis
lint: golangci-lint-install format
	@echo "Running golangci-lint with comprehensive checks..."; \
	$(GOLANGCI_LINT_BINARY) run --verbose --disable-all \
		--enable=errcheck \
		--enable=gosimple \
		--enable=govet \
		--enable=ineffassign \
		--enable=staticcheck \
		--enable=unused \
		--enable=gosec \
		--timeout=5m

## Install govulncheck
govulncheck-install: .validate_go_installed
	@if ! command -v $(GOVULNCHECK_BINARY) >/dev/null 2>&1; then \
		echo "Installing govulncheck..."; \
		go install $(VULNCHECK_PACKAGE)@latest; \
	else \
		echo "govulncheck is already installed."; \
	fi

## Run govulncheck for dependency vulnerability scans
vulncheck: govulncheck-install
	@echo "Running govulncheck vulnerability scan in verbose mode..."; \
	$(GOVULNCHECK_BINARY) -show verbose ./...


.generate-go-mod:
	@if [ ! -f go.mod ]; then \
		echo -e "\033[34mInitializing Go module with name: $(MODULE_NAME)...\033[0m"; \
		go mod init $(MODULE_NAME); \
		echo -e "\033[32mgo.mod created.\033[0m"; \
	else \
		echo -e "\033[33mgo.mod already exists. Skipping go mod init.\033[0m"; \
	fi

.generate-main-go:
	@if [ ! -f main.go ]; then \
		echo -e "\033[34mCreating main.go with a Hello World program...\033[0m"; \
		printf "%s\n" \
		"package main" \
		"" \
		"import \"fmt\"" \
		"" \
		"func main() {" \
		"    fmt.Println(\"Hello, World!\")" \
		"}" > main.go; \
		echo -e "\033[32mmain.go created.\033[0m"; \
	else \
		echo -e "\033[33mmain.go already exists. Skipping creation.\033[0m"; \
	fi

.generate-main-test-go:
	@if [ ! -f main_test.go ]; then \
		echo -e "\033[34mCreating main_test.go with a basic test...\033[0m"; \
		printf "%s\n" \
		"package main" \
		"" \
		"import (" \
		"    \"testing\"" \
		"    \"os\"" \
		"    \"io\"" \
		"    \"bytes\"" \
		")" \
		"" \
		"func TestMainProgram(t *testing.T) {" \
		"    // Capture standard output" \
		"    r, w, _ := os.Pipe()" \
		"    stdout := os.Stdout" \
		"    os.Stdout = w" \
		"    defer func() { os.Stdout = stdout }()" \
		"" \
		"    main()" \
		"" \
		"    // Close the pipe and read the output" \
		"    w.Close()" \
		"    var buf bytes.Buffer" \
		"    io.Copy(&buf, r)" \
		"    r.Close()" \
		"" \
		"    // Verify output" \
		"    expected := \"Hello, World!\\n\"" \
		"    actual := buf.String()" \
		"    if actual != expected {" \
		"        t.Errorf(\"Expected %q but got %q\", expected, actual)" \
		"    }" \
		"}" > main_test.go; \
		echo -e "\033[32mmain_test.go created.\033[0m"; \
	else \
		echo -e "\033[33mmain_test.go already exists. Skipping creation.\033[0m"; \
	fi

.initialize-git:
	@if [ -d .git ]; then \
		if git remote get-url origin 2>/dev/null | grep -q "^$(TEMPLATE_REPO_URL)$$"; then \
			echo -e "\033[31mThe repository is currently linked to the template upstream ($(TEMPLATE_REPO_URL)).\033[0m"; \
			echo -e "\033[34mResetting .git and initializing a new Git repository...\033[0m"; \
			rm -rf .git; \
			[ -f LICENSE ] && rm LICENSE; \
			[ -f README.md ] && rm README.md; \
			git init -b main; \
			echo -e "\033[32mGit repository has been reset and initialized.\033[0m"; \
		else \
			echo -e "\033[33mThe repository is not linked to the original go_maker template upstream. Skipping Git reset.\033[0m"; \
		fi \
	else \
		echo -e "\033[34mNo .git directory found. Initializing a new Git repository...\033[0m"; \
		git init -b main; \
		[ -f LICENSE ] && rm LICENSE; \
		[ -f README.md ] && rm README.md; \
		echo -e "\033[32mNew Git repository initialized.\033[0m"; \
	fi

## Initialize a new Go project in the current directory
mod-init: .validate_go_installed .generate-go-mod .generate-main-go .generate-main-test-go .initialize-git

## Clean up go.mod and go.sum files
mod-tidy: .validate_go_installed
	@echo "Tidying up go.mod and go.sum..."; \
	go mod tidy

## Update all dependencies to the latest compatible versions
mod-update: .validate_go_installed
	@echo "Updating all dependences to the latest compatible versions..."; \
	go get -u ./...
	echo "Running go mod tidy to clean up dependences..."; \
	go mod tidy

format:
	@echo "Formatting Go files using gofmt..."; \
	go fmt ./...

run:
	@go run .

test:
	@go test ./...

build:
	go build -o $(LOCAL_BIN)$(MODULE_NAME) .

ex:
	$(MODULE_NAME)

tree:
	@echo "Printing project structure to treefile"; \
	# rm tree.txt; \
	# @echo "$(shell basename $(PWD))"
	tree -n --dirsfirst -I "Makefile|tree.txt" -o tree.txt

clean:
	rm $(LOCAL_BIN)$(MODULE_NAME)