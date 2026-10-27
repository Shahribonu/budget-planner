// Command sttbench measures how well a speech-to-text provider gets amounts right on real
// voice notes — the metric that matters for logging, not word error rate.
//
// It reads a manifest CSV with a header row and columns:
//
//	file,transcript,amounts
//	clips/001.oga,"taksi yigirma besh ming",25000 UZS
//	clips/002.oga,"non 8 ming, sut 14 ming",8000 UZS; 14000 UZS
//
// file is the audio path relative to the manifest; transcript is what a person heard
// (used with -provider transcript to measure the normalizer alone); amounts are the
// expected major-unit amounts in order.
//
//	go run ./cmd/sttbench -manifest clips/manifest.csv -provider transcript
//	go run ./cmd/sttbench -manifest clips/manifest.csv -provider <name>
package main

import (
	"context"
	"encoding/csv"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/Shahribonu/budget-planner/backend/internal/capture/normalizer"
	"github.com/Shahribonu/budget-planner/backend/internal/capture/stt"
	"github.com/Shahribonu/budget-planner/backend/internal/money"
)

type clip struct {
	file, transcript string
	want             []money.Amount
}

func main() {
	manifest := flag.String("manifest", "", "path to manifest.csv")
	provider := flag.String("provider", "transcript", `"transcript" to test the normalizer on human transcripts, or an STT provider name`)
	lang := flag.String("lang", "uz-Latn", "language hint passed to the provider")
	verbose := flag.Bool("v", false, "print every clip, not just failures")
	flag.Parse()
	if *manifest == "" {
		flag.Usage()
		os.Exit(2)
	}
	clips, err := load(*manifest)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	var tr stt.Transcriber
	if *provider != "transcript" {
		if tr, err = stt.New(*provider); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}

	dir := filepath.Dir(*manifest)
	ok, errs := 0, 0
	var latencies []time.Duration
	for _, c := range clips {
		text := c.transcript
		if tr != nil {
			audio, err := os.ReadFile(filepath.Join(dir, c.file))
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			start := time.Now()
			text, err = tr.Transcribe(context.Background(), audio, "audio/ogg", *lang)
			latencies = append(latencies, time.Since(start))
			if err != nil {
				errs++
				fmt.Printf("ERROR %s: %v\n", c.file, err)
				continue
			}
		}
		got := amounts(text)
		pass := equal(got, c.want)
		if pass {
			ok++
		}
		if !pass || *verbose {
			mark := "FAIL"
			if pass {
				mark = "ok  "
			}
			fmt.Printf("%s %s\n     heard: %q\n     got:   %v\n     want:  %v\n", mark, c.file, text, got, c.want)
		}
	}

	n := len(clips)
	fmt.Printf("\nprovider %s: %d/%d clips with every amount right (%.1f%%), %d errors\n", *provider, ok, n, 100*float64(ok)/float64(max(n, 1)), errs)
	if len(latencies) > 0 {
		sort.Slice(latencies, func(i, j int) bool { return latencies[i] < latencies[j] })
		fmt.Printf("latency p50 %v, p90 %v\n", latencies[len(latencies)/2], latencies[len(latencies)*9/10])
	}
}

func amounts(text string) []money.Amount {
	var out []money.Amount
	for _, it := range normalizer.Parse(text, normalizer.Options{DefaultCurrency: money.UZS}) {
		out = append(out, it.Amount)
	}
	return out
}

func equal(a, b []money.Amount) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func load(path string) ([]clip, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	rows, err := csv.NewReader(f).ReadAll()
	if err != nil {
		return nil, err
	}
	if len(rows) < 2 {
		return nil, fmt.Errorf("%s: no clips", path)
	}
	var clips []clip
	for i, r := range rows[1:] {
		if len(r) < 3 {
			return nil, fmt.Errorf("%s:%d: want file,transcript,amounts", path, i+2)
		}
		c := clip{file: r[0], transcript: r[1]}
		for _, a := range strings.Split(r[2], ";") {
			fields := strings.Fields(a)
			if len(fields) != 2 {
				return nil, fmt.Errorf("%s:%d: amount %q, want like \"25000 UZS\"", path, i+2, a)
			}
			major, err := strconv.ParseFloat(fields[0], 64)
			if err != nil {
				return nil, fmt.Errorf("%s:%d: %v", path, i+2, err)
			}
			cur := money.Currency(fields[1])
			c.want = append(c.want, money.Amount{Minor: int64(major*100 + 0.5), Currency: cur})
		}
		clips = append(clips, c)
	}
	return clips, nil
}
