package memory

import (
	"context"
	"strings"
	"testing"

	domainmemory "github.com/DEEIX-AI/DEEIX-Chat/backend/internal/domain/memory"
	"github.com/DEEIX-AI/DEEIX-Chat/backend/internal/repository"
)

type memoryStubRepo struct {
	nextID uint
	items  map[uint]domainmemory.UserMemory
}

func newMemoryStubRepo() *memoryStubRepo {
	return &memoryStubRepo{nextID: 1, items: map[uint]domainmemory.UserMemory{}}
}

func (r *memoryStubRepo) CreateUserMemory(_ context.Context, item *domainmemory.UserMemory) error {
	item.ID = r.nextID
	r.nextID++
	r.items[item.ID] = *item
	return nil
}

func (r *memoryStubRepo) UpdateUserMemoryByID(_ context.Context, userID uint, memoryID uint, value string, scope string, updatedBy string) (*domainmemory.UserMemory, error) {
	item, ok := r.items[memoryID]
	if !ok || item.UserID != userID {
		return nil, repository.ErrNotFound
	}
	item.Value = value
	item.Scope = scope
	item.UpdatedBy = updatedBy
	r.items[memoryID] = item
	return &item, nil
}

func (r *memoryStubRepo) DeleteUserMemoryByID(_ context.Context, userID uint, memoryID uint) error {
	item, ok := r.items[memoryID]
	if !ok || item.UserID != userID {
		return repository.ErrNotFound
	}
	delete(r.items, memoryID)
	return nil
}

func (r *memoryStubRepo) CountUserMemories(_ context.Context, userID uint) (int64, error) {
	var count int64
	for _, item := range r.items {
		if item.UserID == userID {
			count++
		}
	}
	return count, nil
}

func (r *memoryStubRepo) UpsertUserMemory(_ context.Context, item *domainmemory.UserMemory) error {
	for id, existing := range r.items {
		if existing.UserID == item.UserID && existing.MemoryKey == item.MemoryKey {
			item.ID = id
			r.items[id] = *item
			return nil
		}
	}
	return r.CreateUserMemory(context.Background(), item)
}

func (r *memoryStubRepo) DeleteUserMemory(_ context.Context, userID uint, memoryKey string) error {
	for id, item := range r.items {
		if item.UserID == userID && item.MemoryKey == memoryKey {
			delete(r.items, id)
		}
	}
	return nil
}

func (r *memoryStubRepo) ListUserMemories(_ context.Context, userID uint) ([]domainmemory.UserMemory, error) {
	result := make([]domainmemory.UserMemory, 0)
	for _, item := range r.items {
		if item.UserID == userID {
			result = append(result, item)
		}
	}
	return result, nil
}

func (r *memoryStubRepo) SearchUserMemoriesByEmbedding(_ context.Context, _ uint, _ []float32, _ int, _ float64) ([]domainmemory.UserMemory, error) {
	return nil, nil
}

func (r *memoryStubRepo) UpsertUserMemoryEmbedding(_ context.Context, _ uint, _ string, _ string, _ []float32) error {
	return nil
}

func TestAddUserMemoryCreatesOrdinaryMemoryAndReturnsID(t *testing.T) {
	repo := newMemoryStubRepo()
	svc := NewService(repo)

	item, err := svc.AddUserMemory(context.Background(), 7, "  用户长期使用 Ubuntu  ", "assistant")
	if err != nil {
		t.Fatalf("AddUserMemory() error = %v", err)
	}
	if item.ID == 0 || item.UserID != 7 {
		t.Fatalf("expected persisted user memory, got %#v", item)
	}
	if item.Value != "用户长期使用 Ubuntu" || item.Scope != "memory" || item.UpdatedBy != "assistant" {
		t.Fatalf("unexpected normalized memory: %#v", item)
	}
	if !strings.HasPrefix(item.MemoryKey, "memory:") {
		t.Fatalf("expected generated internal key, got %q", item.MemoryKey)
	}
}

func TestAddUserMemoryRejectsLimit(t *testing.T) {
	repo := newMemoryStubRepo()
	for i := 0; i < MaxUserMemories; i++ {
		item := &domainmemory.UserMemory{UserID: 9, MemoryKey: string(rune('a' + i)), Value: "memory", Scope: "memory"}
		if err := repo.CreateUserMemory(context.Background(), item); err != nil {
			t.Fatal(err)
		}
	}
	svc := NewService(repo)

	if _, err := svc.AddUserMemory(context.Background(), 9, "one more", "assistant"); err != ErrMemoryLimitReached {
		t.Fatalf("expected ErrMemoryLimitReached, got %v", err)
	}
}

func TestUpdateAndDeleteUserMemoryByIDEnforceOwnership(t *testing.T) {
	repo := newMemoryStubRepo()
	item := &domainmemory.UserMemory{UserID: 3, MemoryKey: "legacy-key", Value: "old", Scope: "preference"}
	if err := repo.CreateUserMemory(context.Background(), item); err != nil {
		t.Fatal(err)
	}
	svc := NewService(repo)

	updated, err := svc.UpdateUserMemory(context.Background(), 3, item.ID, " new content ", "assistant")
	if err != nil {
		t.Fatalf("UpdateUserMemory() error = %v", err)
	}
	if updated.Value != "new content" || updated.Scope != "memory" {
		t.Fatalf("unexpected updated memory: %#v", updated)
	}
	if _, err := svc.UpdateUserMemory(context.Background(), 4, item.ID, "stolen", "assistant"); err != repository.ErrNotFound {
		t.Fatalf("expected ownership-safe not found, got %v", err)
	}
	if err := svc.DeleteUserMemoryByID(context.Background(), 4, item.ID); err != repository.ErrNotFound {
		t.Fatalf("expected ownership-safe delete not found, got %v", err)
	}
	if err := svc.DeleteUserMemoryByID(context.Background(), 3, item.ID); err != nil {
		t.Fatalf("DeleteUserMemoryByID() error = %v", err)
	}
}

func TestMemoryContentValidation(t *testing.T) {
	svc := NewService(newMemoryStubRepo())
	if MaxUserMemories != 200 {
		t.Fatalf("expected 200 memories, got %d", MaxUserMemories)
	}
	if MaxMemoryContentRunes != 150 {
		t.Fatalf("expected 150 characters per memory, got %d", MaxMemoryContentRunes)
	}
	if _, err := svc.AddUserMemory(context.Background(), 1, "   ", "assistant"); err != ErrMemoryContentRequired {
		t.Fatalf("expected required error, got %v", err)
	}
	if _, err := svc.AddUserMemory(context.Background(), 1, strings.Repeat("记", 150), "assistant"); err != nil {
		t.Fatalf("expected 150 characters accepted, got %v", err)
	}
	if _, err := svc.AddUserMemory(context.Background(), 1, strings.Repeat("记", 151), "assistant"); err != ErrMemoryContentTooLong {
		t.Fatalf("expected too long error, got %v", err)
	}
}
