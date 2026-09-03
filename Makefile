build:
	go build -o mappy ./cli

# The default: no qmk, no firmware tree, no keyboard needed.
test: unit-test

unit-test:
	go test -v ./...

# Adds the tests that shell out to qmk c2json, so this runs the unit tests too.
# Needs qmk on PATH and the ZSA fork reachable - see cli/README.md.
integration-test:
	go test -v -tags integration ./...

# Regenerate cli/testdata/keymap.json from keymap.c, so the two cannot drift.
# Needs qmk and the ZSA fork; nothing in `make test` depends on it.
testdata:
	qmk c2json -kb zsa/moonlander/reva -km vimster \
		cli/testdata/keymap.c --no-cpp > cli/testdata/keymap.json
