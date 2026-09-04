# Contributing

## Branches

`development` is where work lands. `main` is the release branch — pushing to
it is what asks release-please for a release.

```
feature branch  --squash merge-->  development  --merge commit-->  main
```

**The merge into `main` must be a merge commit, not a squash.** Squashing
collapses every commit in the release into one, and release-please builds the
changelog from the individual commits. Squash-and-merge is right for a PR into
`development`, where one PR becomes one conventional commit; it is wrong for
`development` into `main`.

## Commits

[Conventional commits](https://www.conventionalcommits.org). The type decides
the version bump, and the scope decides which component it belongs to:

```
feat(cli): detect the keyboard from USB          minor bump on cli
fix(cli): stop set from clobbering the padding   patch bump on cli
feat(plugin): draw the thumb cluster             minor bump on plugin
chore: bump the go version                       no release
docs: correct the udev step                      no release
```

`cli` and `plugin` version independently, so a change to one does not bump the
other. Commits are routed by the path they touch, so the scope should match
where the change actually lands.

Anything with a `!` after the type, or a `BREAKING CHANGE:` footer, is a major
bump. Below 1.0.0 that is a minor bump instead.

Everything before this file used plain imperative subjects, and
`bootstrap-sha` in `release-please-config.json` points at the last of them.
release-please does not look further back, so that history produces no
changelog rather than a misleading one.

## Tests

```sh
make lint              # gofmt + go vet
make test              # no toolchain needed
make integration-test  # + qmk c2json
make compile-test      # + a real firmware build
make flash-test        # + an attached keyboard, writes nothing
```

## Signing off

There is no GitHub Actions job running the tests. Three of the four tiers
cannot run on a hosted runner — `integration` needs `qmk` and ZSA's fork,
`compile` needs the ARM toolchain, `flash` needs a keyboard attached — so a
runner could only ever have enforced the first one.

Instead the checks run here and the results are posted to GitHub as commit
statuses, which branch protection gates on:

```sh
make signoff           # runs lint, unit, integration, compile and posts each
./scripts/signoff.sh --dry-run   # preview the statuses, run nothing
```

It signs off on committed, pushed code only. A green status has to describe
the tree that is actually on the PR, so it refuses on a dirty working tree or
a commit that is not on a remote branch.

Every check runs even after one fails, so the PR shows which is red rather
than stopping at the first. `flash` is not signed: its tests write no firmware
but do need a board plugged in, and a red status because the keyboard is
unplugged says nothing about the change. Run `make flash-test` by hand when
touching flashing.

## Releases

Merging `development` into `main` opens a release-please PR with the
changelog and version bump. Merging *that* tags the release and, if `cli`
released, builds `linux/amd64` and `linux/arm64` binaries onto it.

Linux only: detection reads `/sys/bus/usb/devices`, so a binary for another OS
would be one that cannot find a keyboard.
