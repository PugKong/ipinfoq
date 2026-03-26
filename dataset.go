package main

import (
	"fmt"
	"io"
	"iter"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

const datasetFileName = "ipinfo_lite.gz"

type Record struct {
	Network       string `json:"network"`
	Country       string `json:"country"`
	CountryCode   string `json:"country_code"`
	Continent     string `json:"continent"`
	ContinentCode string `json:"continent_code"`
	ASN           string `json:"asn"`
	Name          string `json:"as_name"`
	Domain        string `json:"as_domain"`
}

type DatasetService struct {
	URL string
	Dir string
}

func NewDatasetService(url, dir string) *DatasetService {
	return &DatasetService{
		URL: url,
		Dir: dir,
	}
}

type DownloadOptions struct {
	Token string
	IPv   string
}

func (d *DatasetService) Download(options DownloadOptions) error {
	if err := os.MkdirAll(d.Dir, 0750); err != nil {
		return fmt.Errorf("create dataset dir: %w", err)
	}

	request, err := http.NewRequest(http.MethodGet, d.URL, nil)
	if err != nil {
		return fmt.Errorf("prepare http request: %w", err)
	}
	request.Header.Set("Authorization", "Bearer "+options.Token)

	client := &http.Client{Timeout: time.Minute}
	resp, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("perform download request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected HTTP status: %s", resp.Status)
	}

	filePath := filepath.Join(d.Dir, datasetFileName)
	tmpFilePath := filePath + ".tmp"
	if err := saveDataset(options, tmpFilePath, resp.Body); err != nil {
		_ = os.Remove(tmpFilePath)

		return err
	}

	if err := os.Rename(tmpFilePath, filePath); err != nil {
		return fmt.Errorf("move temp file: %w", err)
	}

	return nil
}

func saveDataset(options DownloadOptions, to string, r io.Reader) error {
	file, err := os.Create(to)
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	defer func() { _ = file.Close() }()

	writer := NewGzipJSONWriter[Record](file)
	defer func() { _ = writer.Close() }()

	reader, err := NewGzipJSONReader[Record](r)
	if err != nil {
		return fmt.Errorf("make response reader: %w", err)
	}
	defer func() { _ = reader.Close() }()

	for record, err := range reader.All() {
		if err != nil {
			return fmt.Errorf("read response record: %w", err)
		}

		version := "4"
		if strings.Contains(record.Network, ":") {
			version = "6"
		}

		if options.IPv != "" && options.IPv != version {
			continue
		}

		if version == "4" && !strings.Contains(record.Network, "/") {
			record.Network += "/32"
		}

		if err := writer.Write(record); err != nil {
			return fmt.Errorf("write record: %w", err)
		}
	}

	if err := writer.Close(); err != nil {
		return fmt.Errorf("close gzip writer: %w", err)
	}

	return nil
}

type QueryOptions struct {
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

func (d *DatasetService) Query(options QueryOptions) iter.Seq2[Record, error] {
	return func(yield func(Record, error) bool) {
		file, err := os.Open(filepath.Join(d.Dir, datasetFileName))
		if err != nil {
			yield(Record{}, fmt.Errorf("open dataset file: %w", err))

			return
		}
		defer func() { _ = file.Close() }()

		reader, err := NewGzipJSONReader[Record](file)
		if err != nil {
			yield(Record{}, fmt.Errorf("init dataset reader: %w", err))

			return
		}
		defer func() { _ = reader.Close() }()

		if options.CaseInsensitive {
			lower := func(items []string) {
				for i, item := range items {
					items[i] = strings.ToLower(item)
				}
			}

			lower(options.Countries)
			lower(options.CountryCodes)
			lower(options.Continents)
			lower(options.ContinentCodes)
			lower(options.ASNs)
			lower(options.Names)
			lower(options.Domains)
		}

		for record, err := range reader.All() {
			if err != nil {
				yield(record, fmt.Errorf("read record: %w", err))

				return
			}

			_, network, err := net.ParseCIDR(record.Network)
			if err != nil {
				yield(record, fmt.Errorf("parse %q network: %w", record.Network, err))

				return
			}

			if !match(options, record, network) {
				continue
			}

			if !yield(record, nil) {
				return
			}
		}
	}
}

func match(options QueryOptions, record Record, network *net.IPNet) bool {
	if len(options.IPs) > 0 {
		contains := func(ip net.IP) bool { return network.Contains(ip) }

		if !slices.ContainsFunc(options.IPs, contains) {
			return false
		}
	}

	compare := func(str string) func(string) bool {
		if options.CaseInsensitive {
			str = strings.ToLower(str)
		}

		return func(substr string) bool {
			if options.Contains {
				return strings.Contains(str, substr)
			}

			return str == substr
		}
	}

	in := func(needle string, haystack []string) bool {
		if len(haystack) == 0 {
			return true
		}

		return slices.ContainsFunc(haystack, compare(needle))
	}

	return in(record.Country, options.Countries) &&
		in(record.CountryCode, options.CountryCodes) &&
		in(record.Continent, options.Continents) &&
		in(record.ContinentCode, options.ContinentCodes) &&
		in(record.ASN, options.ASNs) &&
		in(record.Name, options.Names) &&
		in(record.Domain, options.Domains)
}
