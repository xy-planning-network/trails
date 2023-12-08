package trails_test

import (
	"errors"
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/xy-planning-network/trails"
)

type TestModelSource struct {
	ID        uint               `db:"id"`
	CreatedAt time.Time          `db:"created_at" json:"createdAt"`
	UpdatedAt time.Time          `db:"updated_at" json:"updatedAt"`
	DeletedAt trails.DeletedTime `db:"deleted_at" json:"deletedAt"`
	FieldA    string             `db:"field_a"`
	FieldB    int                `db:"field_b"`
}

type TestModelableMatch struct {
	trails.Model
	FieldA string `db:"field_a"`
	FieldB int    `db:"field_b"`
}

type TestNoTagSource struct {
	FieldA string
}

type TestModelableMismatch struct {
	trails.Model
	FieldA string `db:"field_a"`
}

type TestModelableString string

func (TestModelableString) Exists() bool { return true }

func TestCastAll(t *testing.T) {
	// Arrange
	var source []TestModelSource
	customErr := errors.New("oops")

	// Act
	modelMatches, err := trails.CastAll[TestModelableMatch](source, customErr)

	// Assert
	require.Zero(t, modelMatches)
	require.ErrorIs(t, err, customErr)

	// Arrange
	var strSource string

	// Act
	modelMatches, err = trails.CastAll[TestModelableMatch](strSource, trails.ErrNotExist)

	// Assert
	require.Zero(t, modelMatches)
	require.ErrorIs(t, err, trails.ErrNotImplemented)

	// Arrange
	for i := 3; i > 0; i-- {
		m := TestModelSource{
			ID:        uint(i),
			CreatedAt: time.Now(),
			FieldA:    strconv.Itoa(i),
			FieldB:    i,
		}
		source = append(source, m)
	}

	// Act
	modelMatches, err = trails.CastAll[TestModelableMatch](source, nil)

	// Assert
	require.Nil(t, err)
	require.Len(t, modelMatches, len(source))
	for _, m := range modelMatches {
		require.True(t, m.Exists())
	}

	// Arrange + Act
	maps, err := trails.CastAll[map[string]any](source, nil)

	// Assert
	require.Nil(t, err)
	require.Len(t, maps, len(source))
	for _, m := range maps {
		require.NotZero(t, m["id"])
	}
}

func TestCastOne(t *testing.T) {
	// Arrange
	var source TestModelSource
	customErr := errors.New("oops")

	// Act
	modelMatch, err := trails.CastOne[TestModelableMatch](source, customErr)

	// Assert
	require.Zero(t, modelMatch)
	require.ErrorIs(t, err, customErr)

	// Arrange
	var strSource string

	// Act
	modelMatch, err = trails.CastOne[TestModelableMatch](&strSource, trails.ErrNotExist)

	// Assert
	require.Zero(t, modelMatch)
	require.ErrorIs(t, err, trails.ErrNotImplemented)

	// Arrange + Act
	mapp, err := trails.CastOne[map[string]any](source, nil)

	// Assert
	require.Zero(t, mapp)
	require.ErrorIs(t, err, trails.ErrNotExist)

	// Arrange + Act
	str, err := trails.CastOne[TestModelableString](source, nil)

	// Assert
	require.Zero(t, str)
	require.ErrorIs(t, err, trails.ErrNotImplemented)

	// Arrange + Act
	modelMatch, err = trails.CastOne[TestModelableMatch](TestNoTagSource{}, nil)

	// Assert
	require.Zero(t, modelMatch)
	require.ErrorIs(t, err, trails.ErrNotValid)

	// Arrange + Act
	modelMatch, err = trails.CastOne[TestModelableMatch](source, nil)

	// Assert
	require.Zero(t, modelMatch)
	require.ErrorIs(t, err, trails.ErrNotExist)

	// Arrange
	source.ID = 1

	// Act
	mismatch, err := trails.CastOne[TestModelableMismatch](source, nil)

	// Assert
	require.NotZero(t, mismatch)
	require.ErrorIs(t, err, trails.ErrNotValid)

	// Arrange
	source.FieldA = "test"
	source.FieldB = 2

	// Act
	mapp, err = trails.CastOne[map[string]any](source, nil)

	// Assert
	require.Equal(t, source.ID, mapp["id"])
	require.Equal(t, source.CreatedAt, mapp["created_at"])
	require.Equal(t, source.UpdatedAt, mapp["updated_at"])
	require.Equal(t, source.DeletedAt, mapp["deleted_at"])
	require.Equal(t, source.FieldA, mapp["field_a"])
	require.Equal(t, source.FieldB, mapp["field_b"])

	// Arrange + Act
	modelMatch, err = trails.CastOne[TestModelableMatch](source, nil)

	// Assert
	require.Equal(t, source.ID, modelMatch.ID)
	require.Equal(t, source.CreatedAt, modelMatch.CreatedAt)
	require.Equal(t, source.UpdatedAt, modelMatch.UpdatedAt)
	require.Equal(t, source.DeletedAt, modelMatch.DeletedAt)
	require.Equal(t, source.FieldA, modelMatch.FieldA)
	require.Equal(t, source.FieldB, modelMatch.FieldB)
}

type exampleDB struct{}

func (exampleDB) GetExampleByID(_ uint) (ExampleDBModel, error) { return exampleModel, nil }
func (exampleDB) ListExamples() ([]ExampleDBModel, error) {
	ret := make([]ExampleDBModel, 3)
	for i := 0; i < 3; i++ {
		em := exampleModel
		em.ID += uint(i)
		em.FieldB += i
		ret[i] = em
	}

	return ret, nil
}

var (
	db           exampleDB
	exampleModel = ExampleDBModel{
		ID:        1,
		CreatedAt: time.Now(),
		FieldA:    "Example usage",
		FieldB:    42,
	}
)

type ExampleDBModel struct {
	ID        uint               `db:"id"`
	CreatedAt time.Time          `db:"created_at"`
	UpdatedAt time.Time          `db:"updated_at"`
	DeletedAt trails.DeletedTime `db:"deleted_at"`
	FieldA    string             `db:"field_a"`
	FieldB    int                `db:"field_b"`
}

type ExampleDomainModel struct {
	trails.Model
	FieldA string `db:"field_a"`
	FieldB int    `db:"field_b"`
}

func ExampleCastAll() {
	domainStructs, err := trails.CastAll[ExampleDomainModel](db.ListExamples())

	fmt.Println("err is nil:", err == nil)
	fmt.Println("lens match:", len(domainStructs) == 3)
	for _, d := range domainStructs {
		if !d.Exists() {
			fmt.Println("oops, failed copy")
		}
	}
	// Output:
	// err is nil: true
	// lens match: true
}

func ExampleCastOne() {
	domainStruct, err := trails.CastOne[ExampleDomainModel](db.GetExampleByID(1))

	fmt.Println("err is nil:", err == nil)
	fmt.Println("ID copied:", domainStruct.ID == exampleModel.ID)
	fmt.Println("CreatedAt copied:", domainStruct.CreatedAt == exampleModel.CreatedAt)
	fmt.Println("FieldA copied:", domainStruct.FieldA == exampleModel.FieldA)
	fmt.Println("FieldB copied:", domainStruct.FieldB == exampleModel.FieldB)
	// Output:
	// err is nil: true
	// ID copied: true
	// CreatedAt copied: true
	// FieldA copied: true
	// FieldB copied: true
}
