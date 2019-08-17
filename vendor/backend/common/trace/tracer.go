package trace

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"sync"
)

const (
	CDTTraceid = "codoon_request_id"
	CDTSpanid  = "cdt-spanid"
	CDTSampled = "cdt-sampled"
	CDTCtxKey  = "cdt-ctx-key"
)

var (
	ErrNoTrace = errors.New("no trace found")
)

type cdTracer struct {
	spPool sync.Pool
}

var _tracer *cdTracer

func (t *cdTracer) newSpanForPool() interface{} {
	sp := newSpan("", "", 0, true)
	sp.tracer = t
	return sp
}

func (t *cdTracer) getSpan(name, traceId string, parentId uint64, sampled bool) *Span {
	sp := t.spPool.Get().(*Span)
	sp.reNew(name, traceId, parentId, sampled)
	return sp
}

func (t *cdTracer) putSpan(sp *Span) {
	t.spPool.Put(sp)
}

func init() {
	_tracer = &cdTracer{}
	_tracer.spPool = sync.Pool{
		New: func() interface{} {
			return _tracer.newSpanForPool()
		},
	}
}

func StartSpan(name string, sampled bool) *Span {
	return _tracer.getSpan(name, fmt.Sprintf("%d", generateID()), 0, sampled)
}

func StartSpanWithTraceID(name string, id string, sampled bool) *Span {
	return _tracer.getSpan(name, id, 0, sampled)
}

func StartChildSpan(name string, parentCtx SpanContext) *Span {
	return _tracer.getSpan(name, parentCtx.TraceID, parentCtx.SpanID, parentCtx.Sampled)
}

func ExtractContext(carrier interface{}) (SpanContext, error) {
	ctx := SpanContext{}
	if carrier == nil {
		return ctx, ErrNoTrace
	}
	switch t := carrier.(type) {
	case http.Header:
		traceId := t.Get(CDTTraceid)
		if traceId == "" {
			return ctx, ErrNoTrace
		}
		spanId := t.Get(CDTSpanid)
		if spanId == "" {
			return ctx, ErrNoTrace
		}
		spanIdU64, err := strconv.ParseUint(spanId, 10, 64)
		if err != nil {
			return ctx, err
		}
		ctx.TraceID = traceId
		ctx.SpanID = spanIdU64
		ctx.Sampled = t.Get(CDTSampled) == "true"
	case context.Context:
		if v := t.Value(CDTCtxKey); v != nil {
			if vCtx, ok := v.(SpanContext); ok {
				ctx = vCtx
			} else {
				return ctx, ErrNoTrace
			}
		} else {
			return ctx, ErrNoTrace
		}
	default:
		return ctx, errors.New("carrier type not supported")
	}

	return ctx, nil
}

func ExtractChildSpan(name string, carrier interface{}) (*Span, error) {
	ctx, err := ExtractContext(carrier)
	if err != nil {
		return nil, err
	}

	return StartChildSpan(name, ctx), nil
}

func Inject(spCtx SpanContext, carrier interface{}) {
	switch t := carrier.(type) {
	case http.Header:
		t.Set(CDTTraceid, fmt.Sprintf("%v", spCtx.TraceID))
		t.Set(CDTSpanid, fmt.Sprintf("%v", spCtx.SpanID))
		t.Set(CDTSampled, fmt.Sprintf("%v", spCtx.Sampled))
	}
}

func ExtractStdContext(parent context.Context, carrier interface{}) context.Context {
	var parentCtx context.Context
	if parent != nil {
		parentCtx = parent
	} else {
		parentCtx = context.Background()
	}

	spCtx, err := ExtractContext(carrier)
	if err != nil {
		return parentCtx
	}
	return context.WithValue(parentCtx, CDTCtxKey, spCtx)
}
