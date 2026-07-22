#!/bin/bash
set -euo pipefail

main() {
    printf '%s\n' \
        'Error: signed binary bootstrap is not available.' \
        'MARS will not install release assets until an independently trusted bootstrap can verify the signed archive contract.' \
        '' \
        'Install from an independently reviewed source checkout using Go 1.25.12 or newer:' \
        '  git clone https://github.com/greaveselliott/MARS.git' \
        '  cd MARS' \
        '  Review and check out the exact source commit you intend to trust.' \
        '  make install' \
        '' \
        'If you already have a reviewed checkout, run make install there.' >&2
    exit 1
}

main "$@"
