# Install

## Homebrew (macOS / Linux)

```bash
brew install Omotolani98/foostash/foostash
```

Or tap first, then install:

```bash
brew tap Omotolani98/foostash
brew install foostash
```

Upgrade with `brew upgrade foostash`.

## From source

```bash
go install github.com/Omotolani98/foostash/cmd/foostash@latest
foostash -v
```

Requires Go 1.25+.

## From a release binary

Download a prebuilt archive for your OS/arch from [Releases](https://github.com/Omotolani98/foostash/releases), extract, and place `foostash` on your `PATH`.

```bash
# example: macOS arm64
curl -L https://github.com/Omotolani98/foostash/releases/latest/download/foostash_*_darwin_arm64.tar.gz | tar xz
sudo mv foostash /usr/local/bin/
foostash -v
```

## Verify

```bash
foostash -v
# foostash v0.2.3 (commit …, built …)
```
