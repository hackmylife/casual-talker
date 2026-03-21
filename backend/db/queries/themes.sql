-- name: ListCourses :many
SELECT * FROM courses ORDER BY sort_order ASC;

-- name: GetCourse :one
SELECT * FROM courses WHERE id = $1;

-- name: ListThemesByCourse :many
SELECT * FROM themes WHERE course_id = $1 ORDER BY sort_order ASC;

-- name: GetTheme :one
SELECT * FROM themes WHERE id = $1;
