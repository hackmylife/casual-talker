package handler

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/naoki-watanabe/casual-talker/backend/internal/domain"
	"github.com/naoki-watanabe/casual-talker/backend/internal/repository"
)

// --- mock SessionRepository (only GetTheme and GetCourse are used) ---

type mockSessionRepo struct {
	themes  map[string]*domain.Theme
	courses map[string]*domain.Course
}

func (m *mockSessionRepo) ListCourses(_ context.Context) ([]domain.Course, error) {
	panic("not implemented")
}

func (m *mockSessionRepo) GetCourse(_ context.Context, id string) (*domain.Course, error) {
	c, ok := m.courses[id]
	if !ok {
		return nil, repository.ErrNotFound
	}
	return c, nil
}

func (m *mockSessionRepo) ListThemesByCourse(_ context.Context, _ string) ([]domain.Theme, error) {
	panic("not implemented")
}

func (m *mockSessionRepo) GetTheme(_ context.Context, id string) (*domain.Theme, error) {
	t, ok := m.themes[id]
	if !ok {
		return nil, repository.ErrNotFound
	}
	return t, nil
}

func (m *mockSessionRepo) CreateSession(_ context.Context, _, _ string, _, _ int) (*domain.Session, error) {
	panic("not implemented")
}

func (m *mockSessionRepo) GetSession(_ context.Context, _ string) (*domain.Session, error) {
	panic("not implemented")
}

func (m *mockSessionRepo) ListSessionsByUser(_ context.Context, _ string, _, _ int) ([]domain.Session, error) {
	panic("not implemented")
}

func (m *mockSessionRepo) CompleteSession(_ context.Context, _ string, _ int) error {
	panic("not implemented")
}

func (m *mockSessionRepo) CreateTurn(_ context.Context, _ *domain.Turn) (*domain.Turn, error) {
	panic("not implemented")
}

func (m *mockSessionRepo) ListTurnsBySession(_ context.Context, _ string) ([]domain.Turn, error) {
	panic("not implemented")
}

func (m *mockSessionRepo) CreateFeedback(_ context.Context, _ *domain.Feedback) (*domain.Feedback, error) {
	panic("not implemented")
}

func (m *mockSessionRepo) GetFeedbackBySession(_ context.Context, _ string) (*domain.Feedback, error) {
	panic("not implemented")
}

func (m *mockSessionRepo) GetPastSessionTopics(_ context.Context, _, _ string, _ int) ([]string, error) {
	panic("not implemented")
}

func (m *mockSessionRepo) GetUserStats(_ context.Context, _ string) (*domain.UserStats, error) {
	panic("not implemented")
}

func (m *mockSessionRepo) GetUserLanguageStats(_ context.Context, _ string) ([]domain.LanguageStat, error) {
	panic("not implemented")
}

func (m *mockSessionRepo) GetUserSessionDates(_ context.Context, _ string) ([]time.Time, error) {
	panic("not implemented")
}

// --- tests ---

func TestResolveTargetLanguage_Success(t *testing.T) {
	courseID := "course-en"
	themeID := "theme-greetings"

	repo := &mockSessionRepo{
		themes: map[string]*domain.Theme{
			themeID: {
				ID:       themeID,
				CourseID: courseID,
				Title:    "Greetings",
			},
		},
		courses: map[string]*domain.Course{
			courseID: {
				ID:             courseID,
				TargetLanguage: "en",
			},
		},
	}

	lang, err := resolveTargetLanguage(context.Background(), repo, themeID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if lang != "en" {
		t.Errorf("expected target language %q, got %q", "en", lang)
	}
}

func TestResolveTargetLanguage_ThemeNotFound(t *testing.T) {
	repo := &mockSessionRepo{
		themes:  map[string]*domain.Theme{},
		courses: map[string]*domain.Course{},
	}

	_, err := resolveTargetLanguage(context.Background(), repo, "nonexistent-theme")
	if err == nil {
		t.Fatal("expected error for unknown theme, got nil")
	}
	if !errors.Is(err, repository.ErrNotFound) {
		t.Errorf("expected repository.ErrNotFound, got %v", err)
	}
}

func TestResolveTargetLanguage_CourseNotFound(t *testing.T) {
	themeID := "theme-orphan"

	repo := &mockSessionRepo{
		themes: map[string]*domain.Theme{
			themeID: {
				ID:       themeID,
				CourseID: "missing-course",
				Title:    "Orphan Theme",
			},
		},
		courses: map[string]*domain.Course{},
	}

	_, err := resolveTargetLanguage(context.Background(), repo, themeID)
	if err == nil {
		t.Fatal("expected error for unknown course, got nil")
	}
	if !errors.Is(err, repository.ErrNotFound) {
		t.Errorf("expected repository.ErrNotFound, got %v", err)
	}
}
