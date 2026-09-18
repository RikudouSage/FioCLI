.DEFAULT_GOAL := build-current
SQLCIPHER_DIR := $(CURDIR)/external/sqlcipher
GO_BUILD_TAGS := libsqlite3 no_postgres no_mysql no_ydb no_clickhouse no_libsql no_mssql no_vertica
GO_SQLITE3_MODFILE := $(CURDIR)/build/go-sqlite3-sqlcipher.mod

PACKAGE_NAME := fio-cli
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null | sed -e 's/^v//' -e 's/-/./g' -e 's/[^[:alnum:].]/_/g')
PACKAGE_BUILD_DIR := $(CURDIR)/build/package
DEB_ROOT := $(PACKAGE_BUILD_DIR)/deb
RPM_TOPDIR := $(PACKAGE_BUILD_DIR)/rpm
APPIMAGE_DIR := $(PACKAGE_BUILD_DIR)/AppDir
APPIMAGE_ARCH := $(shell uname -m)
APPIMAGETOOL := $(PACKAGE_BUILD_DIR)/tools/appimagetool-$(APPIMAGE_ARCH).AppImage
APPIMAGETOOL_URL ?= https://github.com/AppImage/appimagetool/releases/download/continuous/appimagetool-$(APPIMAGE_ARCH).AppImage
OUT_DIR := $(CURDIR)/out
# Override the detected loader path for an unusual libc or filesystem layout.
ELF_INTERPRETER ?=

SQLCIPHER_CFLAGS := -O2 -fPIC \
	-DSQLITE_HAS_CODEC \
	-DSQLCIPHER_CRYPTO_OPENSSL \
	-DSQLITE_EXTRA_INIT=sqlcipher_extra_init \
	-DSQLITE_EXTRA_SHUTDOWN=sqlcipher_extra_shutdown

SQLCIPHER_CURRENT_DIR := $(CURDIR)/build/sqlcipher/current
SQLCIPHER_CURRENT_INSTALL := $(SQLCIPHER_CURRENT_DIR)/install
SQLCIPHER_CURRENT_LIB := $(SQLCIPHER_CURRENT_INSTALL)/lib/libsqlite3.a


# ------------------------------------------------------------------------------
# SQLCipher
# ------------------------------------------------------------------------------

build-sqlcipher-current: $(SQLCIPHER_CURRENT_LIB)

$(SQLCIPHER_CURRENT_LIB): $(SQLCIPHER_DIR)/configure
	mkdir -p $(SQLCIPHER_CURRENT_DIR)
	cd $(SQLCIPHER_CURRENT_DIR) && \
		CC="gcc" \
		CPPFLAGS="-I$$OPENSSL_CURRENT_INCLUDE" \
		CFLAGS="$(SQLCIPHER_CFLAGS)" \
		LDFLAGS="-L$$OPENSSL_CURRENT_LIB -lcrypto" \
		$(SQLCIPHER_DIR)/configure \
			--with-tclsh="$$SQLCIPHER_TCLSH" \
			--with-tcl="$$SQLCIPHER_TCL_CONFIG_DIR" \
			--disable-shared \
			--enable-static \
			--with-tempstore=yes

	$(MAKE) -C $(SQLCIPHER_CURRENT_DIR) libsqlite3.a sqlite3.h

	mkdir -p $(SQLCIPHER_CURRENT_INSTALL)/lib
	mkdir -p $(SQLCIPHER_CURRENT_INSTALL)/include

	cp $(SQLCIPHER_CURRENT_DIR)/libsqlite3.a \
		$(SQLCIPHER_CURRENT_INSTALL)/lib/

	cp $(SQLCIPHER_CURRENT_DIR)/sqlite3.h \
		$(SQLCIPHER_CURRENT_INSTALL)/include/

	cp $(SQLCIPHER_DIR)/src/sqlite3ext.h \
		$(SQLCIPHER_CURRENT_INSTALL)/include/

generate-go-sqlite3:
	./scripts/generate-go-sqlite3-sqlcipher.sh


build-current: build-sqlcipher-current generate-go-sqlite3
	CGO_ENABLED=1 \
	CC="gcc" \
	CGO_CFLAGS="-I$(SQLCIPHER_CURRENT_INSTALL)/include" \
	CGO_LDFLAGS="-L$(SQLCIPHER_CURRENT_INSTALL)/lib -Wl,-Bstatic -lsqlite3 -Wl,-Bdynamic -L$$OPENSSL_CURRENT_LIB -lcrypto -lm" \
	go build \
		-modfile "$(GO_SQLITE3_MODFILE)" \
		-tags "$(GO_BUILD_TAGS)" \
		-o fio \
		.
	patchelf --set-interpreter "$(if $(ELF_INTERPRETER),$(ELF_INTERPRETER),$$(sh scripts/linux-elf-interpreter.sh))" --remove-rpath fio

# ------------------------------------------------------------------------------
# Packages
# ------------------------------------------------------------------------------

deb: build-current
	rm -rf $(DEB_ROOT)
	mkdir -p $(DEB_ROOT)/DEBIAN $(DEB_ROOT)/usr/bin $(DEB_ROOT)/usr/share/doc/$(PACKAGE_NAME)
	install -m 755 fio $(DEB_ROOT)/usr/bin/fio
	install -m 644 README.md $(DEB_ROOT)/usr/share/doc/$(PACKAGE_NAME)/README.md
	install -m 644 LICENSE $(DEB_ROOT)/usr/share/doc/$(PACKAGE_NAME)/copyright
	sed -e 's/@VERSION@/$(VERSION)/g' -e "s/@ARCH@/$$(dpkg --print-architecture)/g" \
		packaging/debian/control.in > $(DEB_ROOT)/DEBIAN/control
	mkdir -p $(OUT_DIR)
	dpkg-deb --build --root-owner-group $(DEB_ROOT) \
		$(OUT_DIR)/$(PACKAGE_NAME)_$(VERSION)_$$(dpkg --print-architecture).deb

build-deb: deb

rpm: build-current
	rm -rf $(RPM_TOPDIR)
	mkdir -p $(RPM_TOPDIR)/BUILD $(RPM_TOPDIR)/BUILDROOT $(RPM_TOPDIR)/RPMS \
		$(RPM_TOPDIR)/SOURCES $(RPM_TOPDIR)/SPECS $(RPM_TOPDIR)/SRPMS
	install -m 755 fio $(RPM_TOPDIR)/SOURCES/fio
	install -m 644 README.md $(RPM_TOPDIR)/SOURCES/README.md
	install -m 644 LICENSE $(RPM_TOPDIR)/SOURCES/LICENSE
	sed -e 's/@VERSION@/$(VERSION)/g' \
		packaging/rpm/fio-cli.spec.in > $(RPM_TOPDIR)/SPECS/$(PACKAGE_NAME).spec
	rpmbuild -bb --define '_topdir $(RPM_TOPDIR)' --define '_prefix /usr' \
		$(RPM_TOPDIR)/SPECS/$(PACKAGE_NAME).spec
	mkdir -p $(OUT_DIR)
	cp $(RPM_TOPDIR)/RPMS/*/$(PACKAGE_NAME)-$(VERSION)-*.rpm $(OUT_DIR)/

build-rpm: rpm

$(APPIMAGETOOL):
	mkdir -p $(dir $@)
	curl --fail --location --output $@ $(APPIMAGETOOL_URL)
	chmod 755 $@

appimage: build-current $(APPIMAGETOOL)
	rm -rf $(APPIMAGE_DIR)
	mkdir -p $(APPIMAGE_DIR)/usr/bin $(APPIMAGE_DIR)/usr/lib $(APPIMAGE_DIR)/usr/share/metainfo
	install -m 755 fio $(APPIMAGE_DIR)/usr/bin/fio
	install -m 755 packaging/appimage/AppRun $(APPIMAGE_DIR)/AppRun
	install -m 644 packaging/appimage/fio-cli.desktop $(APPIMAGE_DIR)/fio-cli.desktop
	install -m 644 packaging/appimage/fio-cli.svg $(APPIMAGE_DIR)/fio-cli.svg
	install -m 644 packaging/appimage/dev.chrastecky.fio-cli.metainfo.xml $(APPIMAGE_DIR)/usr/share/metainfo/dev.chrastecky.fio-cli.metainfo.xml
	cp -L "$$OPENSSL_CURRENT_LIB/libcrypto.so.3" $(APPIMAGE_DIR)/usr/lib/
	patchelf --set-rpath '$$ORIGIN/../lib' $(APPIMAGE_DIR)/usr/bin/fio
	mkdir -p $(OUT_DIR)
	ARCH="$(APPIMAGE_ARCH)" APPIMAGE_EXTRACT_AND_RUN=1 $(APPIMAGETOOL) \
		--no-appstream $(APPIMAGE_DIR) $(OUT_DIR)/$(PACKAGE_NAME)_$(VERSION)_$(APPIMAGE_ARCH).AppImage

build-appimage: appimage

# ------------------------------------------------------------------------------
# Cleanup
# ------------------------------------------------------------------------------

clean-sqlcipher:
	rm -rf build/sqlcipher


clean:
	rm -rf build out
	rm -f fio

.PHONY: \
	build-sqlcipher-current \
	generate-go-sqlite3 \
	build-current \
	deb \
	build-deb \
	rpm \
	build-rpm \
	appimage \
	build-appimage \
	clean-sqlcipher \
	clean
