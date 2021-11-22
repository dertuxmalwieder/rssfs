# rssfs: A RSS reader as a file system

[![Scc Count Badge](https://sloc.xyz/github/dertuxmalwieder/rssfs?category=code)](https://github.com/dertuxmalwieder/rssfs) [![Donate](https://img.shields.io/badge/Donate-PayPal-green.svg)](https://paypal.me/GebtmireuerGeld)

Are you unsure how to read RSS feeds? Why don\'t you just mount them?

## What does this software do?

It will mirror RSS and Atom feeds as file systems. Example file system
structure for one feed with two articles:

    /tmp/mnt/rssfs/Open Source Feed/Hello World.html
    /tmp/mnt/rssfs/Open Source Feed/Second Article.html

## Compatibility

This software is written and tested mainly on macOS with
[macFUSE](http://osxfuse.github.io). Other FUSE implementations should
work as well.

### A note on Windows support

As Windows would require WinFsp which might or might not properly
support FUSE file systems, Windows is officially unsupported as of
now (which means: the time that you read this). Tests and contributions
are welcome!

## Build

    fossil clone https://code.rosaelefanten.org/rssfs
    cd rssfs
    go build

## Configure

Copy `rssfs.hcl-example` as `rssfs.hcl` to your configuration directory
and adjust your settings. The required path is:

-   On Windows: `%APPDATA%\rssfs.hcl`
-   On macOS: `$HOME/Library/Application Support/rssfs.hcl`
-   Elsewhere: `$XDG_CONFIG_HOME/rssfs.hcl` (or `$HOME/.config/rssfs.hcl`)

Set a `mountpoint` (unless you are on Windows) and one or more feeds which can be inside or outside a category. (Categories are not required. Subcategories are *not* supported.)

If you don't define `cache`, the feed will be fetched every time you open any other feed.

### Default values for feed settings

* `plaintext`: `false`
* `showlink`: `false`
* `cache`: `false`
* `cachemins`: `60`

## Run

### macOS and other non-Windows machines

    ./rssfs

### Windows

Not implemented yet. Sorry.

## Repositories

- AUR: [aur.archlinux.org](https://aur.archlinux.org/packages/rssfs)
