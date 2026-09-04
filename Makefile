build:
	go build -o mappy ./cli

# gofmt is a check here, not a rewrite: signoff must fail on unformatted code
# rather than quietly fixing it and signing off on something else.
lint:
	@unformatted=$$(gofmt -l .); \
	if [ -n "$$unformatted" ]; then \
		echo "not gofmt'd:"; echo "$$unformatted"; exit 1; \
	fi
	go vet ./...

# Run the checks locally and post the results to GitHub as commit statuses,
# replacing the per-PR test job. flash-test is not signed: it needs a board
# attached, and a red status for an unplugged keyboard says nothing.
signoff:
	./scripts/signoff.sh

# The default: no qmk, no firmware tree, no keyboard needed.
test: unit-test

unit-test:
	go test -v ./...

# Adds the tests that shell out to qmk c2json, so this runs the unit tests too.
# Needs qmk on PATH and the ZSA fork reachable - see cli/README.md.
integration-test:
	go test -v -tags integration ./...

# Adds the tests that build real firmware. Needs the arm toolchain and a keymap
# in the userspace, and a cold build takes minutes, which is why it is not part
# of integration-test. Runs the unit tests too, for the same reason.
compile-test:
	go test -v -tags compile ./...

# Adds the tests that drive qmk flash against an attached keyboard. Nothing in
# them writes firmware - a real flash needs someone to press the reset button -
# so this covers argument assembly through qmk's dry run and the refusals.
flash-test:
	go test -v -tags flash ./...

# Regenerate cli/testdata/keymap.json from keymap.c, so the two cannot drift.
# Needs qmk and the ZSA fork; nothing in `make test` depends on it.
testdata:
	qmk c2json -kb zsa/moonlander/reva -km vimster \
		cli/testdata/keymap.c --no-cpp > cli/testdata/keymap.json
