package game

import (
	"encoding/json"
	"testing"
)

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

func TestLessonZeroIndexCorrectOptionIsKeptInJSON(t *testing.T) {
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
	data, err := json.Marshal(lesson)
	if err != nil {
		t.Fatalf("marshal lesson: %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal lesson json: %v", err)
	}
	got, ok := decoded["correct"]
	if !ok {
		t.Fatalf("lesson json missing correct field: %s", string(data))
	}
	if got.(float64) != 0 {
		t.Fatalf("lesson correct field should be 0 in json, got %#v", got)
	}
}
