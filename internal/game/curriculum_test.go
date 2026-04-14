package game

import "testing"

func TestDefaultCurriculumIsValid(t *testing.T) {
	c := DefaultCurriculum()
	if err := ValidateCurriculum(c); err != nil {
		t.Fatalf("default curriculum invalid: %v", err)
	}
}

func TestLessonFourKeepsShortDeclarationAsCorrect(t *testing.T) {
	c := DefaultCurriculum()
	var lesson Lesson
	for _, item := range c.Lessons {
		if item.ID == 4 {
			lesson = item
			break
		}
	}
	if lesson.ID == 0 {
		t.Fatal("lesson 4 not found")
	}
	if got := lesson.Options[lesson.Correct]; got != "name := \"地鼠\"" {
		t.Fatalf("lesson 4 correct option mismatch: got %q", got)
	}
}

func TestLessonThirtyOneUsesStatusOK(t *testing.T) {
	c := DefaultCurriculum()
	var lesson Lesson
	for _, item := range c.Lessons {
		if item.ID == 31 {
			lesson = item
			break
		}
	}
	if lesson.ID == 0 {
		t.Fatal("lesson 31 not found")
	}
	if lesson.FillPrefix != "http.Status" || lesson.FillAnswer != "OK" || lesson.FillSuffix != "" {
		t.Fatalf("lesson 31 fill config unexpected: prefix=%q answer=%q suffix=%q", lesson.FillPrefix, lesson.FillAnswer, lesson.FillSuffix)
	}
}
