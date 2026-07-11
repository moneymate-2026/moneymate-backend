package repo 
import (
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

// timePtrToPgTimestamptz converts *time.Time to pgtype.Timestamptz
func timePtrToPgTimestamptz(t *time.Time) pgtype.Timestamptz {
	if t == nil {
		return pgtype.Timestamptz{Valid: false}
	}
	return pgtype.Timestamptz{Time: *t, Valid: true}
}

// timeToPgTimestamptz converts time.Time to pgtype.Timestamptz
func timeToPgTimestamptz(t time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: t, Valid: true}
}