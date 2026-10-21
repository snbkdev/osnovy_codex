package monitor

import (
	"context"
	"time"
)

type HistoryBucket struct {
	Start   time.Time `json:"start"`
	Success int       `json:"success"`
	Failure int       `json:"failure"`
}

type History struct {
	Monitor       Monitor         `json:"monitor"`
	From          time.Time       `json:"from"`
	To            time.Time       `json:"to"`
	BucketSeconds int             `json:"bucket_seconds"`
	Buckets       []HistoryBucket `json:"buckets"`
}

func (s *Service) History(ctx context.Context, userID, id string) (History, error) {
	if userID == "" {
		return History{}, ErrUnauthorized
	}
	if id == "" || len(id) > 128 {
		return History{}, ErrInvalid
	}
	to := time.Now().UTC()
	return s.Repo.History(ctx, userID, id, to.Add(-24*time.Hour), to)
}
