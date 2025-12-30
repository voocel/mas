package audit

import "context"

type Record struct {
	RunID    string
	Tool     string
	Decision string
	Status   string
	Error    string
}

type Logger interface {
	Record(ctx context.Context, record Record)
}
