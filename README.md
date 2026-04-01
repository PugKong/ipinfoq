[![License: Unlicense](https://img.shields.io/badge/license-Unlicense-blue.svg)](http://unlicense.org/)
[![CI Workflow Status](https://github.com/PugKong/ipinfoq/actions/workflows/ci.yml/badge.svg)](https://github.com/PugKong/ipinfoq/actions/workflows/ci.yml)
[![Coverage Status](https://coveralls.io/repos/github/PugKong/ipinfoq/badge.svg?branch=main)](https://coveralls.io/github/PugKong/ipinfoq?branch=main)
[![Go Report Card](https://goreportcard.com/badge/github.com/PugKong/ipinfoq)](https://goreportcard.com/report/github.com/PugKong/ipinfoq)
[![GitHub Release](https://img.shields.io/github/release/PugKong/ipinfoq.svg?style=flat)](https://github.com/PugKong/ipinfoq/releases/latest)

# ipinfoq

**ipinfoq** is a CLI tool to download and inspect the [IPinfo Lite](https://ipinfo.io/lite) dataset.

## Installation

Download the proper binary from [Releases](https://github.com/PugKong/ipinfoq/releases) page.

## Usage

Generate the autocompletion script for your shell (optional)

```
$ ipinfoq help completion
Generate the autocompletion script for ipinfoq for the specified shell.
See each sub-command's help for details on how to use the generated script.

Usage:
  ipinfoq completion [command]

Available Commands:
  bash        Generate the autocompletion script for bash
  fish        Generate the autocompletion script for fish
  powershell  Generate the autocompletion script for powershell
  zsh         Generate the autocompletion script for zsh

Flags:
  -h, --help   help for completion

Use "ipinfoq completion [command] --help" for more information about a command.
```

Download the dataset

```
$ ipinfoq help download
Download the dataset

Usage:
  ipinfoq download [flags]

Examples:
  ipinfoq download -t "$IPINFOQ_TOKEN" --ipv 4

Flags:
  -h, --help           help for download
      --ipv string     ip version (4 or 6)
  -t, --token string   api token (required)
```

Query it

```
$ ipinfoq help query
Query the local dataset using filters

Filters can be combined to narrow down results:
- Multiple values within the same filter use OR logic
  (e.g. --country LT --country LV matches either Lithuania or Latvia)
- Different filters are combined using AND logic
  (e.g. --country LT --asn AS13335 matches records that satisfy both conditions)

String filters support exact matching by default. Use --contains (-c) to enable substring matching,
and --case-insensitive (-i) to ignore letter case.

If --ip is provided, records are matched by whether the given IPs fall within the network range.

Output formats:
- jsonl (default): one JSON object per line
- template: use a Go text/template via --template (-t)

When using --template, the following fields are available:
  .Network
  .Country
  .CountryCode
  .Continent
  .ContinentCode
  .ASN
  .Name
  .Domain

Usage:
  ipinfoq query [flags]

Examples:
  - Search information about a company by its name
    ipinfoq query -ci -n "telegram" | jq
  - Generate an ipset config to block Telegram by domain
    ipinfoq query -f template -t '{{ printf "add block_net_set %s\\n" .Network }}' -d "telegram.org" > telegram.conf
  - Inspect domain data using dig
    ipinfoq query --ip "$(dig +short telegram.org | paste -sd, -)"


Flags:
      --asn strings                      include ASNs
  -i, --case-insensitive                 case insensitive search
  -c, --contains                         substring matching
      --continent strings                include continents
      --continent-code strings           include continent codes
      --country strings                  include countries
      --country-code strings             include country codes
  -d, --domain strings                   include organization domains
      --exclude-asn strings              exclude ASNs
      --exclude-continent strings        exclude continents
      --exclude-continent-code strings   exclude continent codes
      --exclude-country strings          exclude countries
      --exclude-country-code strings     exclude country codes
  -D, --exclude-domain strings           exclude organization domains
      --exclude-ip ipSlice               exclude IPs (default [])
  -N, --exclude-name strings             exclude organization names
  -f, --format string                    output format (jsonl, template) (default "jsonl")
  -h, --help                             help for query
      --ip ipSlice                       include IPs (default [])
  -n, --name strings                     include organization names
  -t, --template string                  go text/template in case of "template" format
```

## Contributing

Please feel free to open a discussion, submit issues, fork the repository, and send pull requests.

## License

This project is licensed under the Unlicense License. See the [LICENSE](LICENSE) file for more information.
