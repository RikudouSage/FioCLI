#!/bin/sh

set -eu

case "$(uname -m)" in
    x86_64)
        printf '%s\n' /lib64/ld-linux-x86-64.so.2
        ;;
    aarch64)
        printf '%s\n' /lib/ld-linux-aarch64.so.1
        ;;
    armv7l | armv8l)
        printf '%s\n' /lib/ld-linux-armhf.so.3
        ;;
    i?86)
        printf '%s\n' /lib/ld-linux.so.2
        ;;
    ppc64le)
        printf '%s\n' /lib64/ld64.so.2
        ;;
    s390x)
        printf '%s\n' /lib/ld64.so.1
        ;;
    riscv64)
        printf '%s\n' /lib/ld-linux-riscv64-lp64d.so.1
        ;;
    *)
        printf 'unsupported Linux architecture: %s\n' "$(uname -m)" >&2
        printf 'set ELF_INTERPRETER to the target dynamic loader path\n' >&2
        exit 1
        ;;
esac
