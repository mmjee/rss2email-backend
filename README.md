# rss2email

This repo. contains the daemon for RSS2Email, the accompanying front-end is [here](https://git.maharshi.ninja/root/rss2email-web).

## Setup

```shell
go build -o app.bin -ldflags '-s -w' -trimpath -v .
cp config.toml.example config.toml
$EDITOR config.toml
./app.bin
```

Or see the [Dockerfile](Dockerfile).

## Licence

All code here is licensed under AGPL 3.0 **only**, see [LICENCE](LICENCE).
