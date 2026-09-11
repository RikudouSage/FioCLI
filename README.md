# FioCLI

FioCLI is a command-line client for managing Fio banka accounts. It stores account credentials and transaction data in a local encrypted database, lets you switch between configured accounts, synchronizes transactions, and can create domestic payments. An interactive terminal UI is available for browsing accounts and transactions.

## Requirements

- Go 1.27 or a development shell from `flake.nix`
- A Fio banka API key
- An encryption password supplied through `FIO_ENCRYPTION_PASSWORD`

## Build and run

```sh
nix develop       # optional; provides the native build dependencies
make              # builds ./fio
FIO_ENCRYPTION_PASSWORD='your-password' ./fio accounts
```

The default database is stored at the platform's user config directory under `fio-cli/fio-cli.db`. It can be changed with `--db`. Configuration is read from `$HOME/.fio.yaml` by default, and command-line flags can also be provided through `FIO_` environment variables.

## Common commands

```sh
FIO_ENCRYPTION_PASSWORD='your-password' ./fio add-account
./fio accounts
./fio transactions --sync
./fio transactions --json
./fio sync
./fio switch-account <account-number>
./fio ui
./fio create-payment <account>/<bank-code> <amount>
```

Run `./fio --help` or `./fio <command> --help` for all commands and flags. API keys are prompted for interactively when adding an account; avoid passing them directly on the command line where possible.

## Development

```sh
GOCACHE=/tmp/fiocli-go-cache go test ./...
GOCACHE=/tmp/fiocli-go-cache go vet ./...
```
