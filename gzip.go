package main

import (
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"iter"
)

type GzipJSONReader[T any] struct {
	gzr     *gzip.Reader
	decoder *json.Decoder
}

func NewGzipJSONReader[T any](r io.Reader) (*GzipJSONReader[T], error) {
	gzr, err := gzip.NewReader(r)
	if err != nil {
		return nil, fmt.Errorf("init json gzip reader: %w", err)
	}

	reader := &GzipJSONReader[T]{
		gzr:     gzr,
		decoder: json.NewDecoder(gzr),
	}

	return reader, nil
}

func (j *GzipJSONReader[T]) All() iter.Seq2[T, error] {
	return func(yield func(T, error) bool) {
		for {
			var v T
			if err := j.decoder.Decode(&v); err != nil {
				if !errors.Is(err, io.EOF) {
					yield(v, fmt.Errorf("read json item from gzip archive: %w", err))
				}

				return
			}

			if !yield(v, nil) {
				return
			}
		}
	}
}

func (j *GzipJSONReader[T]) Close() error {
	if err := j.gzr.Close(); err != nil {
		return fmt.Errorf("close gzip reader: %w", err)
	}

	return nil
}

type GzipJSONWriter[T any] struct {
	gzw     *gzip.Writer
	encoder *json.Encoder
}

func NewGzipJSONWriter[T any](w io.Writer) *GzipJSONWriter[T] {
	gzw := gzip.NewWriter(w)

	return &GzipJSONWriter[T]{
		gzw:     gzw,
		encoder: json.NewEncoder(gzw),
	}
}

func (j *GzipJSONWriter[T]) Write(v T) error {
	if err := j.encoder.Encode(v); err != nil {
		return fmt.Errorf("write json item to gzip archive: %w", err)
	}

	return nil
}

func (j *GzipJSONWriter[T]) Close() error {
	if err := j.gzw.Close(); err != nil {
		return fmt.Errorf("close gzip writer: %w", err)
	}

	return nil
}
