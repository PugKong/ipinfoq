package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"text/template"

	"github.com/spf13/cobra"
)

const (
	appName    = "ipinfoq"
	datasetURL = "https://ipinfo.io/data/ipinfo_lite.json.gz"
)

func main() {
	os.Exit(Run(NewEnv()))
}

type Env struct {
	Args     []string
	CacheDir string
	Out      io.Writer
	Err      io.Writer
	URL      string
}

func NewEnv() *Env {
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		panic(err)
	}

	return &Env{
		Args:     os.Args[1:],
		CacheDir: cacheDir,
		Out:      os.Stdout,
		Err:      os.Stderr,
		URL:      datasetURL,
	}
}

func Run(env *Env) int {
	cmd := &cobra.Command{
		Use:   appName,
		Short: "A CLI tool to download and query ipinfo.io lite dataset",
	}

	cmd.SetArgs(env.Args)
	cmd.SetOut(env.Out)
	cmd.SetErr(env.Err)

	datasetService := NewDatasetService(env.URL, filepath.Join(env.CacheDir, appName))

	cmd.AddCommand(
		NewDownloadCommand(datasetService),
		NewQueryCommand(datasetService),
	)

	if err := cmd.Execute(); err != nil {
		return 1
	}

	return 0
}

func NewDownloadCommand(service *DatasetService) *cobra.Command {
	var options struct {
		Token string
		IPv   string
	}

	cmd := &cobra.Command{
		Use:     "download",
		Short:   "Download the dataset",
		Example: "  ipinfoq download -t \"$IPINFOQ_TOKEN\" --ipv 4",
		Args:    cobra.ExactArgs(0),
	}

	flags := cmd.Flags()
	flags.StringVarP(&options.Token, "token", "t", "", "api token (required)")
	flags.StringVar(&options.IPv, "ipv", "", "ip version (4 or 6)")

	_ = cmd.MarkFlagRequired("token")

	cmd.RunE = func(*cobra.Command, []string) error {
		downloadOptions := DownloadOptions{
			Token: options.Token,
			IPv:   options.IPv,
		}

		return service.Download(downloadOptions)
	}

	return cmd
}

func NewQueryCommand(service *DatasetService) *cobra.Command {
	var options struct {
		Format   string
		Template string

		Contains        bool
		CaseInsensitive bool

		IPs            []net.IP
		Countries      []string
		CountryCodes   []string
		Continents     []string
		ContinentCodes []string
		ASNs           []string
		Names          []string
		Domains        []string
	}

	cmd := &cobra.Command{
		Use:   "query",
		Short: "Query the dataset",
		Long: `Query the local dataset using filters

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
`,
		Example: `  - Search information about a company by its name
    ipinfoq query -ci -n "telegram" | jq
  - Generate an ipset config to block Telegram by domain (if your government insists)
    ipinfoq query -f template -t '{{ printf "add block_net_set %s\\n" .Network }}' -d "telegram.org" > telegram.conf
  - Inspect domain data using dig
    ipinfoq query --ip "$(dig +short telegram.org | paste -sd, -)"
`,
		Args: cobra.ExactArgs(0),
	}

	flags := cmd.Flags()

	flags.StringVarP(&options.Format, "format", "f", "jsonl", "output format (jsonl, template)")
	flags.StringVarP(&options.Template, "template", "t", "", "go text/template in case of \"template\" format")

	flags.BoolVarP(&options.Contains, "contains", "c", false, "substring matching")
	flags.BoolVarP(&options.CaseInsensitive, "case-insensitive", "i", false, "case insensitive search")

	flags.IPSliceVar(&options.IPs, "ip", []net.IP{}, "filter by IP addresses")
	flags.StringSliceVar(&options.Countries, "country", []string{}, "filter by countries")
	flags.StringSliceVar(&options.CountryCodes, "country-code", []string{}, "filter by country codes")
	flags.StringSliceVar(&options.Continents, "continent", []string{}, "filter by continents")
	flags.StringSliceVar(&options.ContinentCodes, "continent-code", []string{}, "filter by continent codes")
	flags.StringSliceVar(&options.ASNs, "asn", []string{}, "filter by ASNs")
	flags.StringSliceVarP(&options.Names, "name", "n", []string{}, "filter by organization names")
	flags.StringSliceVarP(&options.Domains, "domain", "d", []string{}, "filter by organization domains")

	cmd.RunE = func(cmd *cobra.Command, _ []string) error {
		queryOptions := QueryOptions{
			Contains:        options.Contains,
			CaseInsensitive: options.CaseInsensitive,

			IPs:            options.IPs,
			Countries:      options.Countries,
			CountryCodes:   options.CountryCodes,
			Continents:     options.Continents,
			ContinentCodes: options.ContinentCodes,
			ASNs:           options.ASNs,
			Names:          options.Names,
			Domains:        options.Domains,
		}

		out := cmd.OutOrStdout()
		var write func(Record) error
		switch options.Format {
		case "jsonl":
			encoder := json.NewEncoder(out)
			write = func(r Record) error { return encoder.Encode(r) }
		case "template":
			tmpl, err := template.New("").Parse(options.Template)
			if err != nil {
				return fmt.Errorf("parse template: %w", err)
			}
			write = func(r Record) error { return tmpl.Execute(out, r) }
		default:
			return fmt.Errorf("unknown format %q", options.Format)
		}

		for record, err := range service.Query(queryOptions) {
			if err != nil {
				return fmt.Errorf("read dataset: %w", err)
			}

			if err := write(record); err != nil {
				return fmt.Errorf("print record: %w", err)
			}
		}

		return nil
	}

	return cmd
}
