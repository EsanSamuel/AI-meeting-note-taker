package repository

import (
	"github.com/jackc/pgx/v5"

	"example.com/internal/db/sqlc"
)

type Repository struct {
	meeting    *MeetingRepository
	transcript *TranscriptRepository
}

func New(db sqlc.DBTX) *Repository {
	return newRepository(sqlc.New(db))
}

func (r *Repository) Meetings() *MeetingRepository {
	return r.meeting
}

func (r *Repository) Transcripts() *TranscriptRepository {
	return r.transcript
}

func (r *Repository) WithTx(tx pgx.Tx) *Repository {
	queries, ok := r.meeting.Querier.(*sqlc.Queries)
	if !ok {
		queries = sqlc.New(tx)
	} else {
		queries = queries.WithTx(tx)
	}

	return newRepository(queries)
}

func newRepository(queries sqlc.Querier) *Repository {
	return &Repository{
		meeting:    &MeetingRepository{Querier: queries},
		transcript: &TranscriptRepository{Querier: queries},
	}
}

type queryRepository struct {
	sqlc.Querier
}

type MeetingRepository struct {
	queryRepository
}

type TranscriptRepository struct {
	queryRepository
}