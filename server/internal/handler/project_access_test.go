package handler

import (
	"testing"

	"github.com/jackc/pgx/v5/pgtype"
)

func TestRestrictProjectFilter_AllAccessPassthrough(t *testing.T) {
	a := pgtype.UUID{Bytes: [16]byte{1}, Valid: true}
	ids, includeNull, empty := restrictProjectFilter(true, nil, []pgtype.UUID{a}, true)
	if empty || !includeNull || len(ids) != 1 {
		t.Fatalf("got ids=%v includeNull=%v empty=%v", ids, includeNull, empty)
	}
}

func TestRestrictProjectFilter_RestrictedNoRequestUsesAccessible(t *testing.T) {
	a := pgtype.UUID{Bytes: [16]byte{1}, Valid: true}
	b := pgtype.UUID{Bytes: [16]byte{2}, Valid: true}
	ids, includeNull, empty := restrictProjectFilter(false, []pgtype.UUID{a, b}, nil, true)
	if empty || includeNull || len(ids) != 2 {
		t.Fatalf("got ids=%v includeNull=%v empty=%v", ids, includeNull, empty)
	}
}

func TestRestrictProjectFilter_RestrictedEmptyAccessible(t *testing.T) {
	ids, includeNull, empty := restrictProjectFilter(false, nil, nil, true)
	if !empty || includeNull || ids != nil {
		t.Fatalf("got ids=%v includeNull=%v empty=%v", ids, includeNull, empty)
	}
}

func TestRestrictProjectFilter_Intersection(t *testing.T) {
	a := pgtype.UUID{Bytes: [16]byte{1}, Valid: true}
	b := pgtype.UUID{Bytes: [16]byte{2}, Valid: true}
	c := pgtype.UUID{Bytes: [16]byte{3}, Valid: true}
	ids, includeNull, empty := restrictProjectFilter(false, []pgtype.UUID{a, b}, []pgtype.UUID{b, c}, true)
	if empty || includeNull || len(ids) != 1 || ids[0].Bytes != b.Bytes {
		t.Fatalf("got ids=%v includeNull=%v empty=%v", ids, includeNull, empty)
	}
}

func TestRestrictProjectFilter_NoOverlapEmpty(t *testing.T) {
	a := pgtype.UUID{Bytes: [16]byte{1}, Valid: true}
	c := pgtype.UUID{Bytes: [16]byte{3}, Valid: true}
	ids, includeNull, empty := restrictProjectFilter(false, []pgtype.UUID{a}, []pgtype.UUID{c}, false)
	if !empty || includeNull || ids != nil {
		t.Fatalf("got ids=%v includeNull=%v empty=%v", ids, includeNull, empty)
	}
}
