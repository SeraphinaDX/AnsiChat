APP_ID := ca.cerberusgames.copperline
GUI_BIN := ansichat-gui
PREFIX ?= $(HOME)/.local
BINDIR := $(PREFIX)/bin
APPLICATIONS_DIR := $(PREFIX)/share/applications
ICON_DIR := $(PREFIX)/share/icons/hicolor/512x512/apps

.PHONY: gui tui install-gui install-gui-prebuilt install-gui-files uninstall-gui

gui:
	go build -o $(GUI_BIN) ./cmd/ansichat-gui

tui:
	go build -o ansichat ./cmd/ansichat

install-gui: gui install-gui-files

install-gui-prebuilt: install-gui-files

install-gui-files:
	install -Dm755 $(GUI_BIN) $(BINDIR)/$(GUI_BIN)
	install -Dm644 cmd/ansichat-gui/Icon.png $(ICON_DIR)/$(APP_ID).png
	install -Dm644 packaging/linux/$(APP_ID).desktop $(APPLICATIONS_DIR)/$(APP_ID).desktop
	@command -v update-desktop-database >/dev/null 2>&1 && update-desktop-database $(APPLICATIONS_DIR) || true
	@command -v gtk-update-icon-cache >/dev/null 2>&1 && gtk-update-icon-cache -f -t $(PREFIX)/share/icons/hicolor >/dev/null 2>&1 || true
	@command -v kbuildsycoca6 >/dev/null 2>&1 && kbuildsycoca6 >/dev/null 2>&1 || true
	@command -v kbuildsycoca5 >/dev/null 2>&1 && kbuildsycoca5 >/dev/null 2>&1 || true
	@echo "Installed AnsiChat GUI desktop integration under $(PREFIX)."
	@echo "On KDE/Wayland, fully close any running AnsiChat window and launch it again."

uninstall-gui:
	rm -f $(BINDIR)/$(GUI_BIN)
	rm -f $(ICON_DIR)/$(APP_ID).png
	rm -f $(APPLICATIONS_DIR)/$(APP_ID).desktop
	@command -v update-desktop-database >/dev/null 2>&1 && update-desktop-database $(APPLICATIONS_DIR) || true
	@command -v gtk-update-icon-cache >/dev/null 2>&1 && gtk-update-icon-cache -f -t $(PREFIX)/share/icons/hicolor >/dev/null 2>&1 || true
