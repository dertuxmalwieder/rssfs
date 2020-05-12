rssfs: A RSS reader as a file system
====================================

[![Scc Count Badge](https://sloc.xyz/github/dertuxmalwieder/rssfs?category=code)](https://github.com/dertuxmalwieder/rssfs) [![Donate](https://img.shields.io/badge/Donate-PayPal-green.svg)](https://paypal.me/GebtmireuerGeld)

Are you unsure how to read RSS feeds? Why don\'t you just mount them?

What does this software do?
---------------------------

It will mirror RSS and Atom feeds as file systems. Example file system
structure for one feed with two articles:

    /tmp/mnt/rssfs/Open Source Feed/Hello World.html
    /tmp/mnt/rssfs/Open Source Feed/Second Article.html

Compatibility
-------------

This software is written and tested mainly on macOS with
[OSXFUSE](http://osxfuse.github.io). Other FUSE implementations should
work as well. Note that Windows support is still a work in progress and
does not really exist yet. (Contributions welcome!)

Build
-----

    fossil clone https://code.rosaelefanten.org/rssfs rssfs.fossil
    fossil open rssfs.fossil
    go build

(You\'ll need `GO111MODULES` to be set to \"on\"!)

Configure
---------

Copy `rssfs.hcl-example` as `rssfs.hcl` to your configuration directory
and adjust your settings. The required path is:

-   On Windows: `%APPDATA%\rssfs.hcl`
-   On macOS: `$HOME/Library/Application Support/rssfs.hcl`
-   Elsewhere: `$XDG_CONFIG_HOME/rssfs.hcl`

Set a `mountpoint` or, on Windows, a `driveletter` and one or more feeds
which can be inside or outside a category. (Categories are not required.
Subcategories are *not* supported.)

Run
---

### macOS and other non-Windows machines

    ./rssfs

### Windows

Not implemented yet. Sorry.

Debate
------

Discuss `rssfs` in *#rssfs* on freenode.net.
