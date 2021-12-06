package plex

import "testing"

func Test_reader_Close_nil(t *testing.T) {
	r := &reader{}

	err := r.Close()
	if err != nil {
		t.Errorf("Expected nil, got %v", err)
	}
}

func Test_writer_Close_nil(t *testing.T) {
	w := &writer{}

	err := w.Close()
	if err != nil {
		t.Errorf("Expected nil, got %v", err)
	}
}
