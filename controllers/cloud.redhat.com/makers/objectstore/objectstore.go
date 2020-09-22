package objectstore

import "context"

type ObjectStore interface {
	CreateBucket(ctx context.Context, bucket string) error
}
