package main

import (
	"testing"
)

func TestNewProgressBar(t *testing.T) {
	total := int64(100)
	pb := NewProgressBar(total)

	if pb.Total != total {
		t.Errorf("Total = %v, want %v", pb.Total, total)
	}

	if !pb.ShowPercent {
		t.Error("ShowPercent should be true by default")
	}

	if !pb.ShowBar {
		t.Error("ShowBar should be true by default")
	}

	if pb.RefreshRate != DEFAULT_REFRESH_RATE {
		t.Errorf("RefreshRate = %v, want %v", pb.RefreshRate, DEFAULT_REFRESH_RATE)
	}
}

func TestProgressBarAdd(t *testing.T) {
	pb := NewProgressBar(100)

	result := pb.Add(10)
	if result != 10 {
		t.Errorf("Add(10) = %v, want 10", result)
	}

	if pb.current != 10 {
		t.Errorf("current = %v, want 10", pb.current)
	}
}

func TestProgressBarIncrement(t *testing.T) {
	pb := NewProgressBar(100)

	result := pb.Increment()
	if result != 1 {
		t.Errorf("Increment() = %v, want 1", result)
	}

	if pb.current != 1 {
		t.Errorf("current = %v, want 1", pb.current)
	}
}

func TestProgressBarSet(t *testing.T) {
	pb := NewProgressBar(100)

	pb.Set(50)
	if pb.current != 50 {
		t.Errorf("current = %v, want 50", pb.current)
	}
}

func TestProgressBarWrite(t *testing.T) {
	pb := NewProgressBar(100)
	data := []byte("test")

	n, err := pb.Write(data)
	if err != nil {
		t.Errorf("Write() error = %v", err)
	}
	if n != len(data) {
		t.Errorf("Write() = %v, want %v", n, len(data))
	}
	if pb.current != int64(len(data)) {
		t.Errorf("current = %v, want %v", pb.current, len(data))
	}
}
