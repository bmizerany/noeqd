package main

import (
	"bytes"
	"testing"
)

type nopWriter struct{}

func (nw *nopWriter) Write(b []byte) (n int, err error) {
	return
}

func BenchmarkIdGeneration(b *testing.B) {
	for n := 0; n < b.N; n++ {
		nextId()
	}
}

func BenchmarkServe01(b *testing.B) {
	for n := 0; n < b.N; n++ {
		b.StopTimer()
		i, o := bytes.NewBuffer([]byte{1}), new(nopWriter)
		b.StartTimer()
		serve(i, o)
	}
}

func BenchmarkServe02(b *testing.B) {
	for n := 0; n < b.N; n++ {
		b.StopTimer()
		i, o := bytes.NewBuffer([]byte{2}), new(nopWriter)
		b.StartTimer()
		serve(i, o)
	}
}

func BenchmarkServe03(b *testing.B) {
	for n := 0; n < b.N; n++ {
		b.StopTimer()
		i, o := bytes.NewBuffer([]byte{3}), new(nopWriter)
		b.StartTimer()
		serve(i, o)
	}
}

func BenchmarkServe05(b *testing.B) {
	for n := 0; n < b.N; n++ {
		b.StopTimer()
		i, o := bytes.NewBuffer([]byte{5}), new(nopWriter)
		b.StartTimer()
		serve(i, o)
	}
}

func BenchmarkServe08(b *testing.B) {
	for n := 0; n < b.N; n++ {
		b.StopTimer()
		i, o := bytes.NewBuffer([]byte{8}), new(nopWriter)
		b.StartTimer()
		serve(i, o)
	}
}

func BenchmarkServe13(b *testing.B) {
	for n := 0; n < b.N; n++ {
		b.StopTimer()
		i, o := bytes.NewBuffer([]byte{13}), new(nopWriter)
		b.StartTimer()
		serve(i, o)
	}
}

func BenchmarkServe21(b *testing.B) {
	for n := 0; n < b.N; n++ {
		b.StopTimer()
		i, o := bytes.NewBuffer([]byte{21}), new(nopWriter)
		b.StartTimer()
		serve(i, o)
	}
}

func BenchmarkServe34(b *testing.B) {
	for n := 0; n < b.N; n++ {
		b.StopTimer()
		i, o := bytes.NewBuffer([]byte{34}), new(nopWriter)
		b.StartTimer()
		serve(i, o)
	}
}

func BenchmarkServe55(b *testing.B) {
	for n := 0; n < b.N; n++ {
		b.StopTimer()
		i, o := bytes.NewBuffer([]byte{55}), new(nopWriter)
		b.StartTimer()
		serve(i, o)
	}
}
