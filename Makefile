.PHONY: build build-windows clean run

APP_NAME := elixir-medics

# Build for the current platform
build:
	go build -o $(APP_NAME) .

# Build and run the app locally
run: build
	./$(APP_NAME)

# Build Windows GUI executable using fyne-cross (requires Docker)
build-windows:
	fyne-cross windows -arch=amd64 -app-id=com.schani.elixir-medics -icon Icon.png

# Clean build artifacts
clean:
	rm -f $(APP_NAME)
	rm -rf fyne-cross
