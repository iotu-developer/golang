package trace

import (
	"backend/common/stringutil"
	"encoding/json"
	"fmt"
	"sort"
	"time"
)

var (
	testMode    = false
	testSpanMap = map[uint64][]byte{}
)

const (
	MaxNameLen    = 2 << 10
	MaxCommentLen = 100 * (2 << 10)
)

type Span struct {
	Id        ID    `json:"id"`
	StartTime int64 `json:"start_time"`
	EndTime   int64 `json:"end_time"`
	// recommended name format: scheme host operation
	Name     string            `json:"name"`
	Comment  string            `json:"comment"`
	Tags     map[string]string `json:"tags,omitempty"`
	sampled  bool
	finished bool
	tracer   *cdTracer
}

type SpanContext struct {
	TraceID string
	SpanID  uint64
	Sampled bool
}

type ID struct {
	Trace  string `json:"trace"`
	Span   uint64 `json:"span"`
	Parent uint64 `json:"parent"`
}

func (id ID) String() string {
	return fmt.Sprintf("%d-%d-%d", id.Trace, id.Span, id.Parent)
}

func newSpan(name string, traceId string, parentId uint64, sampled bool) *Span {
	return &Span{
		Id: ID{
			Trace:  traceId,
			Span:   generateID(),
			Parent: parentId,
		},
		StartTime: time.Now().UnixNano(),
		EndTime:   0,
		Name:      stringutil.Cuts(name, MaxNameLen),
		Comment:   "",
		Tags:      map[string]string{},
		sampled:   sampled,
		finished:  false,
	}
}

func (sp *Span) reNew(name string, traceId string, parentId uint64, sampled bool) {
	sp.Id.Trace = traceId
	sp.Id.Span = generateID()
	sp.Id.Parent = parentId
	sp.StartTime = time.Now().UnixNano()
	sp.EndTime = 0
	sp.Name = stringutil.Cuts(name, MaxNameLen)
	sp.Comment = ""
	sp.Tags = map[string]string{}
	sp.sampled = sampled
	sp.finished = false
}

func (sp *Span) Context() SpanContext {
	return SpanContext{
		TraceID: sp.Id.Trace,
		SpanID:  sp.Id.Span,
		Sampled: sp.sampled,
	}
}

func (sp *Span) Finish() {
	defer func() {
		if sp.tracer != nil {
			sp.tracer.putSpan(sp)
		}
	}()
	if sp.finished || !sp.sampled {
		return
	}

	sp.finished = true
	sp.EndTime = time.Now().UnixNano()
	sp.collect()
}

func (sp *Span) collect() {
	data, _ := json.Marshal(sp)
	fmt.Printf("[CDTrace] %v %s\n", sp.Id.Trace, data)

	if testMode {
		testSpanMap[sp.Id.Span] = data
	}
}

func (sp *Span) SetComment(comment string) {
	sp.Comment = stringutil.Cuts(comment, MaxCommentLen)
}

func (sp *Span) SetTag(k, v string) {
	sp.Tags[k] = v
}

type SpanNode struct {
	Span
	Sub []*SpanNode `json:"sub,omitempty"`
}

type SpanSlice []*SpanNode

func (a SpanSlice) Len() int           { return len(a) }
func (a SpanSlice) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a SpanSlice) Less(i, j int) bool { return a[i].StartTime < a[j].StartTime }

func Treeify(traceList []*SpanNode) []*SpanNode {
	treeList := []*SpanNode{}
	lookup := map[uint64]*SpanNode{}
	for _, t := range traceList {
		lookup[t.Id.Span] = t
	}

	for _, t := range traceList {
		if t.Id.Parent != 0 {
			if node, found := lookup[t.Id.Parent]; !found {
				continue
			} else {
				node.Sub = append(node.Sub, t)
				lookup[t.Id.Parent] = node
			}
		} else {
			treeList = append(treeList, t)
		}
	}

	for _, node := range treeList {
		sortSpanNodeSub(node)
	}

	return treeList
}

func sortSpanNodeSub(sn *SpanNode) {
	if len(sn.Sub) == 0 {
		return
	}

	sort.Sort(SpanSlice(sn.Sub))
	for _, subNode := range sn.Sub {
		sortSpanNodeSub(subNode)
	}
}
