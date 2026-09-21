package repository

import (
	"context"

	"example.com/internal/db/sqlc"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	pgvector_go "github.com/pgvector/pgvector-go"
)

type vectorRepository struct {
	q *sqlc.Queries
}

type CreateTranscriptVectorEmbeddingParams struct {
	MeetingID pgtype.UUID        `json:"meeting_id"`
	Chunk     string             `json:"chunk"`
	Embedding pgvector_go.Vector `json:"embedding"`
}

type VectorResponse struct {
	ID        pgtype.UUID        `json:"id"`
	MeetingID pgtype.UUID        `json:"meeting_id"`
	Chunk     string             `json:"chunk"`
	Embedding pgvector_go.Vector `json:"embedding"`
}

type SearchTranscriptChunkParams struct {
	Embedding pgvector_go.Vector `json:"embedding"`
	Limit     int32              `json:"limit"`
}

type SearchTranscriptChunkResult struct {
	ID        pgtype.UUID `json:"id"`
	MeetingID pgtype.UUID `json:"meeting_id"`
	Chunk     string      `json:"chunk"`
	Distance  float64     `json:"distance"`
}

type TranscriptVector struct {
	ID    pgtype.UUID `json:"id"`
    MeetingID  pgtype.UUID `json:"meeting_id"`
	Chunk string      `json:"chunk"`
	Embedding pgvector_go.Vector `json:"embedding"`
}

type VectorRepository interface {
	CreateVector(ctx context.Context, v CreateTranscriptVectorEmbeddingParams) (VectorResponse, error)
	GetTranscriptVector(ctx context.Context, meetingID uuid.UUID) ([]TranscriptVector, error)
	SearchTranscriptChunk(ctx context.Context, v SearchTranscriptChunkParams) ([]SearchTranscriptChunkResult, error)
}

func NewVectorRepository(pool *pgxpool.Pool) VectorRepository {
	return &vectorRepository{q: sqlc.New(pool)}
}

func (r *vectorRepository) CreateVector(ctx context.Context, v CreateTranscriptVectorEmbeddingParams) (VectorResponse, error) {
	row, err := r.q.CreateTranscriptVectorEmbedding(ctx, sqlc.CreateTranscriptVectorEmbeddingParams{
		MeetingID: v.MeetingID,
		Chunk:     v.Chunk,
		Embedding: v.Embedding,
	})

	if err != nil {
		return VectorResponse{}, err
	}

	return VectorResponse{
		ID:        row.ID,
		MeetingID: row.MeetingID,
		Chunk:     row.Chunk,
		Embedding: row.Embedding,
	}, nil
}

func (r *vectorRepository) GetTranscriptVector(ctx context.Context, meetingID uuid.UUID) ([]TranscriptVector, error) {
	rows, err := r.q.GetTranscriptVector(ctx, pgtype.UUID{Bytes: meetingID, Valid: true})
	if err != nil {
		return nil, err
	}

	vectors := make([]TranscriptVector, 0, len(rows))
	for _, row := range rows {
		vectors = append(vectors, TranscriptVector{
			ID:        row.ID,
			MeetingID: row.MeetingID,
			Chunk:     row.Chunk,
			Embedding: row.Embedding,
		})
	}

	return vectors, nil
}

func (r *vectorRepository) SearchTranscriptChunk(ctx context.Context, v SearchTranscriptChunkParams) ([]SearchTranscriptChunkResult, error) {
	rows, err := r.q.SearchTranscriptChunk(ctx, sqlc.SearchTranscriptChunkParams{
		Embedding: v.Embedding,
		Limit:     v.Limit,
	})

	if err != nil {
		return nil, err
	}

	results := make([]SearchTranscriptChunkResult, 0, len(rows))
	for _, row := range rows {
		distance, ok := row.Distance.(float64)
		if !ok {
			distance = 0
		}

		results = append(results, SearchTranscriptChunkResult{
			ID:        row.ID,
			MeetingID: row.MeetingID,
			Chunk:     row.Chunk,
			Distance:  distance,
		})
	}

	return results, nil
}