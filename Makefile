.DEFAULT_GOAL := build-current
SQLCIPHER_DIR := $(CURDIR)/external/sqlcipher
GO_BUILD_TAGS := libsqlite3 no_postgres no_mysql no_ydb no_clickhouse no_libsql no_mssql no_vertica
GO_SQLITE3_MODFILE := $(CURDIR)/build/go-sqlite3-sqlcipher.mod

SQLCIPHER_CFLAGS := -O2 -fPIC \
	-DSQLITE_HAS_CODEC \
	-DSQLCIPHER_CRYPTO_OPENSSL \
	-DSQLITE_EXTRA_INIT=sqlcipher_extra_init \
	-DSQLITE_EXTRA_SHUTDOWN=sqlcipher_extra_shutdown

SQLCIPHER_CURRENT_DIR := $(CURDIR)/build/sqlcipher/current
SQLCIPHER_CURRENT_LIB := $(SQLCIPHER_CURRENT_INSTALL)/lib/libsqlite3.a
SQLCIPHER_CURRENT_INSTALL := $(SQLCIPHER_CURRENT_DIR)/install


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
		./cmd

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
	clean-sqlcipher \
	clean
