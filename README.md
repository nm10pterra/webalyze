# webalyze

Website technology fingerprinting CLI.

## Features

- Detects web technologies and categories for target domains/URLs.
- Supports single or multiple inputs via flags, file list, or stdin.
- Includes matcher and filter flags for technologies and categories.
- Offers plain output or JSONL output for pipeline-friendly processing.
- Configurable retries, timeout, concurrency, and redirect behavior.

## Installation

Install the latest version:

```bash
go install github.com/nm10pterra/webalyze@latest
```

Install a pinned version:

```bash
go install github.com/nm10pterra/webalyze@v0.1.2
```

## Usage


```sh
webalyze -h
```

This will display help for the tool. Here are all the switches it supports.

```yaml
Usage:
  webalyze [flags]

INPUT:
  -i, -input string[]       list of targets to process
  -l, -list string          file with targets (one per line)

MATCHER:
  -match-tech string[]      include targets with matching technologies
  -mcat, -match-category string[]
                           include targets with matching categories

FILTER:
  -filter-tech string[]     exclude targets with matching technologies
  -fcat, -filter-category string[]
                           exclude targets with matching categories

OUTPUT:
  -j, -jsonl                write output in jsonl format
  -o, -output string        write output to file
  -silent                   only display results in output
  -nc, -no-color            disable colors in cli output
  -v, -verbose              display verbose output
  -version                  display version

CONFIG:
  -retry int                maximum number of retries for requests (default 2)
  -timeout duration         per-request timeout (default 10s)
  -c, -concurrency int      number of concurrent workers (default 10)
  -fr, -follow-redirects    follow http redirects
```