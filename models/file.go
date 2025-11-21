package models

import "time"

// Модель расписания с несколькими группами
type RegularSchedule struct {
	Type         string          `json:"type"`
	UpdatedAt    time.Time       `json:"updatedAt"`
	WeekType     string          `json:"weekType"`
	Semester     string          `json:"semester"`
	AcademicYear string          `json:"academicYear"`
	Groups       []GroupSchedule `json:"groups"`
}

type GroupSchedule struct {
	GroupNumber string        `json:"groupNumber"`
	Direction   string        `json:"direction"`
	Days        []DaySchedule `json:"days"`
}

type DaySchedule struct {
	Date      string   `json:"date"`
	DayOfWeek string   `json:"dayOfWeek"`
	Lessons   []Lesson `json:"lessons"`
}

type Lesson struct {
	Time      string `json:"time"`
	Subject   string `json:"subject"`
	Teacher   string `json:"teacher"`
	Type      string `json:"type"`
	Classroom string `json:"classroom"`
	SubGroup  string `json:"subGroup"`
}

type ReplacementSchedule struct {
	Type         string        `json:"type"`
	Date         string        `json:"date"`
	UpdatedAt    time.Time     `json:"updatedAt"`
	Replacements []Replacement `json:"replacements"`
}

type Replacement struct {
	Time            string `json:"time"`
	OriginalSubject string `json:"originalSubject"`
	NewSubject      string `json:"newSubject"`
	OriginalTeacher string `json:"originalTeacher"`
	NewTeacher      string `json:"newTeacher"`
	Classroom       string `json:"classroom"`
}

type ExamSchedule struct {
	Type      string    `json:"type"`
	UpdatedAt time.Time `json:"updatedAt"`
	Exams     []Exam    `json:"exams"`
}

type Exam struct {
	Date      string `json:"date"`
	Time      string `json:"time"`
	Subject   string `json:"subject"`
	Teacher   string `json:"teacher"`
	Classroom string `json:"classroom"`
}
