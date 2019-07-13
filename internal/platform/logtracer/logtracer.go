package logtracer

import (
	"log"

	"go.opencensus.io/trace"
)

type Tracer struct {
	Logger *log.Logger
}

func (t *Tracer) ExportSpan(s *trace.SpanData) {
	t.Logger.Printf("[TRACE] %#v\n", s)
}
