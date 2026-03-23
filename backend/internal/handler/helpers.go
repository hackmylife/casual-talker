package handler

import (
	"context"

	"github.com/naoki-watanabe/casual-talker/backend/internal/repository"
)

// resolveTargetLanguage resolves the target language for a theme by traversing
// theme → course. It is used by multiple handlers that need to know the
// language for a session without re-implementing the two-step lookup each time.
func resolveTargetLanguage(ctx context.Context, repo repository.SessionRepository, themeID string) (string, error) {
	theme, err := repo.GetTheme(ctx, themeID)
	if err != nil {
		return "", err
	}
	course, err := repo.GetCourse(ctx, theme.CourseID)
	if err != nil {
		return "", err
	}
	return course.TargetLanguage, nil
}
