package main

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRun(t *testing.T) {
	env, _, stderr := newTestEnv(t, []string{})

	exitCode := Run(env)

	require.Equal(t, 0, exitCode, stderr.String())
}

func TestRunDownload(t *testing.T) {
	dataset := []rawRecord{
		{
			"network":        "1.0.0.0/24",
			"country":        "Australia",
			"country_code":   "AU",
			"continent":      "Oceania",
			"continent_code": "OC",
			"asn":            "AS13335",
			"as_name":        "Cloudflare, Inc.",
			"as_domain":      "cloudflare.com",
		},
		{
			"network":        "2001:503:ff40::/46",
			"country":        "United States",
			"country_code":   "US",
			"continent":      "North America",
			"continent_code": "NA",
			"asn":            "AS209242",
			"as_name":        "Cloudflare London, LLC",
			"as_domain":      "cloudflare.com",
		},
	}

	tests := map[string]struct {
		Args    []string
		Token   string
		Records []rawRecord
	}{
		"all": {
			Args:    []string{"download", "--token", "foo"},
			Token:   "foo",
			Records: []rawRecord{dataset[0], dataset[1]},
		},
		"ipv4": {
			Args:    []string{"download", "--token", "bar", "--ipv", "4"},
			Token:   "bar",
			Records: []rawRecord{dataset[0]},
		},
		"ipv6": {
			Args:    []string{"download", "--token", "foo", "--ipv", "6"},
			Token:   "foo",
			Records: []rawRecord{dataset[1]},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Header.Get("Authorization") != "Bearer "+test.Token {
					w.WriteHeader(http.StatusUnauthorized)

					return
				}

				gzw := gzip.NewWriter(w)
				defer func() { _ = gzw.Close() }()

				encoder := json.NewEncoder(gzw)
				for _, record := range dataset {
					_ = encoder.Encode(record)
				}
			}))
			defer server.Close()

			env, stdout, stderr := newTestEnv(t, test.Args)
			env.URL = server.URL

			exitCode := Run(env)

			require.Equal(t, 0, exitCode, stderr.String())
			require.Empty(t, stdout.String())

			records := readDataset(t, env)
			require.Equal(t, test.Records, records)
		})
	}
}

func TestRunQuery(t *testing.T) {
	dataset := []rawRecord{
		// 0
		{
			"network":        "1.0.0.0/24",
			"country":        "Australia",
			"country_code":   "AU",
			"continent":      "Oceania",
			"continent_code": "OC",
			"asn":            "AS13335",
			"as_name":        "Cloudflare, Inc.",
			"as_domain":      "cloudflare.com",
		},
		// 1
		{
			"network":        "1.1.1.0/24",
			"country":        "Australia",
			"country_code":   "AU",
			"continent":      "Oceania",
			"continent_code": "OC",
			"asn":            "AS13335",
			"as_name":        "Cloudflare, Inc.",
			"as_domain":      "cloudflare.com",
		},
		// 2
		{
			"network":        "5.10.214.0/23",
			"country":        "Armenia",
			"country_code":   "AM",
			"continent":      "Asia",
			"continent_code": "AS",
			"asn":            "AS13335",
			"as_name":        "Cloudflare, Inc.",
			"as_domain":      "cloudflare.com",
		},
		// 3
		{
			"network":        "31.13.24.0/29",
			"country":        "United States",
			"country_code":   "US",
			"continent":      "North America",
			"continent_code": "NA",
			"asn":            "AS32934",
			"as_name":        "Facebook, Inc.",
			"as_domain":      "facebook.com",
		},
		// 4
		{
			"network":        "91.105.192.0/23",
			"country":        "Finland",
			"country_code":   "FI",
			"continent":      "Europe",
			"continent_code": "EU",
			"asn":            "AS211157",
			"as_name":        "Telegram Messenger Inc",
			"as_domain":      "telegram.org",
		},
		// 5
		{
			"network":        "91.108.4.0/22",
			"country":        "The Netherlands",
			"country_code":   "NL",
			"continent":      "Europe",
			"continent_code": "EU",
			"asn":            "AS62041",
			"as_name":        "Telegram Messenger Inc",
			"as_domain":      "telegram.org",
		},
	}

	tests := map[string]struct {
		Args    []string
		Records []rawRecord
	}{
		"all": {
			Args:    []string{"query"},
			Records: dataset,
		},

		"include asn": {
			Args:    []string{"query", "--asn", "AS13335"},
			Records: []rawRecord{dataset[0], dataset[1], dataset[2]},
		},
		"include asn (contains)": {
			Args:    []string{"query", "--contains", "--asn", "13335"},
			Records: []rawRecord{dataset[0], dataset[1], dataset[2]},
		},
		"include asn (case insensitive)": {
			Args:    []string{"query", "--case-insensitive", "--asn", "as13335"},
			Records: []rawRecord{dataset[0], dataset[1], dataset[2]},
		},
		"include asns": {
			Args:    []string{"query", "--asn", "AS13335", "--asn", "AS62041"},
			Records: []rawRecord{dataset[0], dataset[1], dataset[2], dataset[5]},
		},

		"include continent": {
			Args:    []string{"query", "--continent", "Asia"},
			Records: []rawRecord{dataset[2]},
		},
		"include continent (contains)": {
			Args:    []string{"query", "--contains", "--continent", "si"},
			Records: []rawRecord{dataset[2]},
		},
		"include continent (case insensitive)": {
			Args:    []string{"query", "--case-insensitive", "--continent", "asia"},
			Records: []rawRecord{dataset[2]},
		},
		"include continents": {
			Args:    []string{"query", "--continent", "Oceania", "--continent", "Asia"},
			Records: []rawRecord{dataset[0], dataset[1], dataset[2]},
		},

		"include continent code": {
			Args:    []string{"query", "--continent-code", "AS"},
			Records: []rawRecord{dataset[2]},
		},
		"include continent code (contains)": {
			Args:    []string{"query", "--contains", "--continent-code", "S"},
			Records: []rawRecord{dataset[2]},
		},
		"include continent code (case insensitive)": {
			Args:    []string{"query", "--case-insensitive", "--continent-code", "as"},
			Records: []rawRecord{dataset[2]},
		},
		"include continent codes": {
			Args:    []string{"query", "--continent-code", "OC", "--continent-code", "AS"},
			Records: []rawRecord{dataset[0], dataset[1], dataset[2]},
		},

		"include country": {
			Args:    []string{"query", "--country", "The Netherlands"},
			Records: []rawRecord{dataset[5]},
		},
		"include country (contains)": {
			Args:    []string{"query", "--contains", "--country", "Netherlands"},
			Records: []rawRecord{dataset[5]},
		},
		"include country (case insensitive)": {
			Args:    []string{"query", "--case-insensitive", "--country", "the netherlands"},
			Records: []rawRecord{dataset[5]},
		},
		"include countries": {
			Args:    []string{"query", "--country", "Armenia", "--country", "The Netherlands"},
			Records: []rawRecord{dataset[2], dataset[5]},
		},

		"include country code": {
			Args:    []string{"query", "--country-code", "NL"},
			Records: []rawRecord{dataset[5]},
		},
		"include country code (contains)": {
			Args:    []string{"query", "--contains", "--country-code", "N"},
			Records: []rawRecord{dataset[5]},
		},
		"include country code (case insensitive)": {
			Args:    []string{"query", "--case-insensitive", "--country-code", "nl"},
			Records: []rawRecord{dataset[5]},
		},
		"include country codes": {
			Args:    []string{"query", "--country-code", "AM", "--country-code", "NL"},
			Records: []rawRecord{dataset[2], dataset[5]},
		},

		"include domain": {
			Args:    []string{"query", "--domain", "telegram.org"},
			Records: []rawRecord{dataset[4], dataset[5]},
		},
		"include domain (contains)": {
			Args:    []string{"query", "--contains", "--domain", "telegram"},
			Records: []rawRecord{dataset[4], dataset[5]},
		},
		"include domain (case insensitive)": {
			Args:    []string{"query", "--case-insensitive", "--domain", "TELEGRAM.ORG"},
			Records: []rawRecord{dataset[4], dataset[5]},
		},
		"include domains": {
			Args:    []string{"query", "--domain", "cloudflare.com", "--domain", "telegram.org"},
			Records: []rawRecord{dataset[0], dataset[1], dataset[2], dataset[4], dataset[5]},
		},

		"include name": {
			Args:    []string{"query", "--name", "Telegram Messenger Inc"},
			Records: []rawRecord{dataset[4], dataset[5]},
		},
		"include name (contains)": {
			Args:    []string{"query", "--contains", "--name", "Messenger"},
			Records: []rawRecord{dataset[4], dataset[5]},
		},
		"include name (case insensitive)": {
			Args:    []string{"query", "--case-insensitive", "--name", "telegram messenger inc"},
			Records: []rawRecord{dataset[4], dataset[5]},
		},
		"include names": {
			Args:    []string{"query", "--name", "\"Facebook, Inc.\"", "--name", "Telegram Messenger Inc"},
			Records: []rawRecord{dataset[3], dataset[4], dataset[5]},
		},

		"include ip": {
			Args:    []string{"query", "--ip", "1.1.1.1"},
			Records: []rawRecord{dataset[1]},
		},
		"include ips": {
			Args:    []string{"query", "--ip", "1.1.1.1", "--ip", "5.10.214.1"},
			Records: []rawRecord{dataset[1], dataset[2]},
		},

		"exclude asn": {
			Args:    []string{"query", "--exclude-asn", "AS13335"},
			Records: []rawRecord{dataset[3], dataset[4], dataset[5]},
		},
		"exclude asn (contains)": {
			Args:    []string{"query", "--contains", "--exclude-asn", "13335"},
			Records: []rawRecord{dataset[3], dataset[4], dataset[5]},
		},
		"exclude asn (case insensitive)": {
			Args:    []string{"query", "--case-insensitive", "--exclude-asn", "as13335"},
			Records: []rawRecord{dataset[3], dataset[4], dataset[5]},
		},
		"exclude asns": {
			Args:    []string{"query", "--exclude-asn", "AS13335", "--exclude-asn", "AS62041"},
			Records: []rawRecord{dataset[3], dataset[4]},
		},

		"exclude continent": {
			Args:    []string{"query", "--exclude-continent", "Asia"},
			Records: []rawRecord{dataset[0], dataset[1], dataset[3], dataset[4], dataset[5]},
		},
		"exclude continent (contains)": {
			Args:    []string{"query", "--contains", "--exclude-continent", "si"},
			Records: []rawRecord{dataset[0], dataset[1], dataset[3], dataset[4], dataset[5]},
		},
		"exclude continent (case insensitive)": {
			Args:    []string{"query", "--case-insensitive", "--exclude-continent", "asia"},
			Records: []rawRecord{dataset[0], dataset[1], dataset[3], dataset[4], dataset[5]},
		},
		"exclude continents": {
			Args:    []string{"query", "--exclude-continent", "Oceania", "--exclude-continent", "Asia"},
			Records: []rawRecord{dataset[3], dataset[4], dataset[5]},
		},

		"exclude continent code": {
			Args:    []string{"query", "--exclude-continent-code", "AS"},
			Records: []rawRecord{dataset[0], dataset[1], dataset[3], dataset[4], dataset[5]},
		},
		"exclude continent code (contains)": {
			Args:    []string{"query", "--contains", "--exclude-continent-code", "S"},
			Records: []rawRecord{dataset[0], dataset[1], dataset[3], dataset[4], dataset[5]},
		},
		"exclude continent code (case insensitive)": {
			Args:    []string{"query", "--case-insensitive", "--exclude-continent-code", "as"},
			Records: []rawRecord{dataset[0], dataset[1], dataset[3], dataset[4], dataset[5]},
		},
		"exclude continent codes": {
			Args:    []string{"query", "--exclude-continent-code", "OC", "--exclude-continent-code", "AS"},
			Records: []rawRecord{dataset[3], dataset[4], dataset[5]},
		},

		"exclude country": {
			Args:    []string{"query", "--exclude-country", "The Netherlands"},
			Records: []rawRecord{dataset[0], dataset[1], dataset[2], dataset[3], dataset[4]},
		},
		"exclude country (contains)": {
			Args:    []string{"query", "--contains", "--exclude-country", "Netherlands"},
			Records: []rawRecord{dataset[0], dataset[1], dataset[2], dataset[3], dataset[4]},
		},
		"exclude country (case insensitive)": {
			Args:    []string{"query", "--case-insensitive", "--exclude-country", "the netherlands"},
			Records: []rawRecord{dataset[0], dataset[1], dataset[2], dataset[3], dataset[4]},
		},
		"exclude countries": {
			Args:    []string{"query", "--exclude-country", "Armenia", "--exclude-country", "The Netherlands"},
			Records: []rawRecord{dataset[0], dataset[1], dataset[3], dataset[4]},
		},

		"exclude country code": {
			Args:    []string{"query", "--exclude-country-code", "NL"},
			Records: []rawRecord{dataset[0], dataset[1], dataset[2], dataset[3], dataset[4]},
		},
		"exclude country code (contains)": {
			Args:    []string{"query", "--contains", "--exclude-country-code", "N"},
			Records: []rawRecord{dataset[0], dataset[1], dataset[2], dataset[3], dataset[4]},
		},
		"exclude country code (case insensitive)": {
			Args:    []string{"query", "--case-insensitive", "--exclude-country-code", "nl"},
			Records: []rawRecord{dataset[0], dataset[1], dataset[2], dataset[3], dataset[4]},
		},
		"exclude country codes": {
			Args:    []string{"query", "--exclude-country-code", "AM", "--exclude-country-code", "NL"},
			Records: []rawRecord{dataset[0], dataset[1], dataset[3], dataset[4]},
		},

		"exclude domain": {
			Args:    []string{"query", "--exclude-domain", "telegram.org"},
			Records: []rawRecord{dataset[0], dataset[1], dataset[2], dataset[3]},
		},
		"exclude domain (contains)": {
			Args:    []string{"query", "--contains", "--exclude-domain", "telegram"},
			Records: []rawRecord{dataset[0], dataset[1], dataset[2], dataset[3]},
		},
		"exclude domain (case insensitive)": {
			Args:    []string{"query", "--case-insensitive", "--exclude-domain", "TELEGRAM.ORG"},
			Records: []rawRecord{dataset[0], dataset[1], dataset[2], dataset[3]},
		},
		"exclude domains": {
			Args:    []string{"query", "--exclude-domain", "cloudflare.com", "--exclude-domain", "telegram.org"},
			Records: []rawRecord{dataset[3]},
		},

		"exclude name": {
			Args:    []string{"query", "--exclude-name", "Telegram Messenger Inc"},
			Records: []rawRecord{dataset[0], dataset[1], dataset[2], dataset[3]},
		},
		"exclude name (contains)": {
			Args:    []string{"query", "--contains", "--exclude-name", "Messenger"},
			Records: []rawRecord{dataset[0], dataset[1], dataset[2], dataset[3]},
		},
		"exclude name (case insensitive)": {
			Args:    []string{"query", "--case-insensitive", "--exclude-name", "telegram messenger inc"},
			Records: []rawRecord{dataset[0], dataset[1], dataset[2], dataset[3]},
		},
		"exclude names": {
			Args:    []string{"query", "--exclude-name", "\"Facebook, Inc.\"", "--exclude-name", "Telegram Messenger Inc"},
			Records: []rawRecord{dataset[0], dataset[1], dataset[2]},
		},

		"exclude ip": {
			Args:    []string{"query", "--exclude-ip", "1.1.1.1"},
			Records: []rawRecord{dataset[0], dataset[2], dataset[3], dataset[4], dataset[5]},
		},
		"exclude ips": {
			Args:    []string{"query", "--exclude-ip", "1.1.1.1", "--exclude-ip", "5.10.214.1"},
			Records: []rawRecord{dataset[0], dataset[3], dataset[4], dataset[5]},
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			env, stdout, stderr := newTestEnv(t, test.Args)
			writeDataset(t, env, dataset)

			exitCode := Run(env)

			require.Equal(t, 0, exitCode, stderr.String())
			require.Equal(t, test.Records, parseJSONL(t, stdout))
		})
	}

	t.Run("template format", func(t *testing.T) {
		dataset := []rawRecord{
			{
				"network":        "1.0.0.0/24",
				"country":        "Australia",
				"country_code":   "AU",
				"continent":      "Oceania",
				"continent_code": "OC",
				"asn":            "AS13335",
				"as_name":        "Cloudflare, Inc.",
				"as_domain":      "cloudflare.com",
			},
		}

		env, stdout, stderr := newTestEnv(t, []string{
			"query",
			"--format",
			"template",
			"--template",
			"{{.Network}}:{{.Country}}:{{.CountryCode}}:{{.Continent}}:{{.ContinentCode}}:{{.ASN}}:{{.Name}}:{{.Domain}}",
		})
		writeDataset(t, env, dataset)

		exitCode := Run(env)

		require.Equal(t, 0, exitCode, stderr.String())
		require.Equal(t, "1.0.0.0/24:Australia:AU:Oceania:OC:AS13335:Cloudflare, Inc.:cloudflare.com", stdout.String())
	})
}

type rawRecord map[string]string

func newTestEnv(t *testing.T, args []string) (*Env, *bytes.Buffer, *bytes.Buffer) {
	t.Helper()

	cacheDir := t.TempDir()

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	env := &Env{
		Args:     args,
		CacheDir: cacheDir,
		Out:      stdout,
		Err:      stderr,
	}

	return env, stdout, stderr
}

func writeDataset(t *testing.T, env *Env, dataset []rawRecord) {
	t.Helper()

	datasetDir := filepath.Join(env.CacheDir, appName)
	if err := os.MkdirAll(datasetDir, 0750); err != nil {
		t.Fatalf("init dataset dir: %s", err.Error())
	}

	datasetPath := filepath.Join(datasetDir, datasetFileName)
	datasetFile, err := os.Create(datasetPath)
	if err != nil {
		t.Fatalf("create dataset file: %s", err.Error())
	}
	defer func() { _ = datasetFile.Close() }()

	gzw := gzip.NewWriter(datasetFile)
	defer func() { _ = gzw.Close() }()

	encoder := json.NewEncoder(gzw)
	for _, record := range dataset {
		if err := encoder.Encode(record); err != nil {
			t.Fatalf("write dataset entry: %s", err.Error())
		}
	}
}

func readDataset(t *testing.T, env *Env) []rawRecord {
	t.Helper()

	datasetPath := filepath.Join(env.CacheDir, appName, datasetFileName)

	datasetFile, err := os.Open(datasetPath)
	if err != nil {
		t.Fatalf("open dataset file: %s", err.Error())
	}
	defer func() { _ = datasetFile.Close() }()

	gzr, err := gzip.NewReader(datasetFile)
	if err != nil {
		t.Fatalf("init gzip reader: %s", err.Error())
	}
	defer func() { _ = gzr.Close() }()

	return parseJSONL(t, gzr)
}

func parseJSONL(t *testing.T, r io.Reader) []rawRecord {
	t.Helper()

	records := []rawRecord{}

	decoder := json.NewDecoder(r)
	for {
		record := rawRecord{}
		if err := decoder.Decode(&record); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			t.Fatalf("parse query command output: %s", err.Error())
		}

		records = append(records, record)
	}

	return records
}
