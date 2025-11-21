package services

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"regexp"
	"schedule-api/models"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"
)

type ParserService struct{}

func NewParserService() *ParserService {
	return &ParserService{}
}

// ParseXLSXToJSON парсит XLSX в JSON
func (s *ParserService) ParseXLSXToJSON(file io.Reader, scheduleType string) ([]byte, error) {
	f, err := excelize.OpenReader(file)
	if err != nil {
		return nil, fmt.Errorf("failed to open xlsx: %w", err)
	}
	defer f.Close()

	switch scheduleType {
	case "основное", "main":
		return s.parseRegularSchedule(f)
	case "замены", "replacements":
		return s.parseReplacementSchedule(f)
	case "экзамены", "exams":
		return s.parseExamSchedule(f)
	default:
		return nil, fmt.Errorf("unknown schedule type: %s", scheduleType)
	}
}

// parseRegularSchedule парсит основное расписание
func (s *ParserService) parseRegularSchedule(f *excelize.File) ([]byte, error) {
	sheet := f.GetSheetList()[0]
	rows, err := f.GetRows(sheet)
	if err != nil {
		return nil, err
	}

	if len(rows) < 10 {
		return nil, fmt.Errorf("insufficient rows in schedule file")
	}

	schedule := models.RegularSchedule{
		Type:      "regular",
		UpdatedAt: time.Now(),
	}

	// Извлекаем метаданные из заголовка
	schedule.WeekType = s.extractWeekType(rows)
	schedule.Semester = s.extractSemester(rows)
	schedule.AcademicYear = s.extractAcademicYear(rows)

	// Находим строку с номерами групп (строка 9, индекс 8)
	if len(rows) < 9 {
		return nil, fmt.Errorf("insufficient rows: need at least 9 rows, got %d", len(rows))
	}

	groupRow := rows[8]
	groupPositions := s.findGroupPositions(groupRow)

	if len(groupPositions) == 0 {
		return nil, fmt.Errorf("no groups found in schedule (row 9). Row content: %v", groupRow)
	}

	log.Printf("Найдено групп: %d", len(groupPositions))

	// Извлекаем направления (строка 8, индекс 7)
	directionRow := rows[7]

	// Инициализируем группы
	for _, pos := range groupPositions {
		groupSchedule := models.GroupSchedule{
			GroupNumber: s.cleanValue(groupRow[pos.Column]),
			Direction:   s.extractDirection(directionRow, pos.Column, pos.EndColumn),
			Days:        make([]models.DaySchedule, 0),
		}
		schedule.Groups = append(schedule.Groups, groupSchedule)
	}

	// Парсим данные расписания (начиная со строки 10, индекс 9)
	currentDay := ""
	currentDate := ""
	var currentDaySchedules []*models.DaySchedule

	for i := 9; i < len(rows); i++ {
		row := rows[i]
		if len(row) < 2 {
			continue
		}

		dayCell := s.cleanValue(row[0])
		timeCell := s.cleanValue(row[1])

		// Проверяем начало нового дня
		if dayCell != "" {
			// Сохраняем предыдущий день для всех групп
			for idx, daySchedule := range currentDaySchedules {
				if daySchedule != nil && len(daySchedule.Lessons) > 0 {
					schedule.Groups[idx].Days = append(schedule.Groups[idx].Days, *daySchedule)
				}
			}

			// Парсим новый день
			currentDay, currentDate = s.parseDayCell(dayCell)
			log.Printf("Обработка нового дня: %s (%s)", currentDay, currentDate)

			// Инициализируем DaySchedule для каждой группы
			currentDaySchedules = make([]*models.DaySchedule, len(groupPositions))
			for idx := range currentDaySchedules {
				currentDaySchedules[idx] = &models.DaySchedule{
					Date:      currentDate,
					DayOfWeek: currentDay,
					Lessons:   make([]models.Lesson, 0),
				}
			}
		}

		// Пропускаем строки без времени
		if timeCell == "" {
			continue
		}

		// Проверяем, что у нас есть инициализированные расписания дней
		if len(currentDaySchedules) == 0 {
			continue
		}

		// Парсим занятия для каждой группы
		for idx, pos := range groupPositions {
			if idx >= len(currentDaySchedules) {
				continue
			}
			lessons := s.parseLessons(row, pos.Column, pos.EndColumn, timeCell)
			if currentDaySchedules[idx] != nil {
				currentDaySchedules[idx].Lessons = append(currentDaySchedules[idx].Lessons, lessons...)
			}
		}
	}

	// Сохраняем последний день
	for idx, daySchedule := range currentDaySchedules {
		if daySchedule != nil && len(daySchedule.Lessons) > 0 {
			schedule.Groups[idx].Days = append(schedule.Groups[idx].Days, *daySchedule)
		}
	}

	return json.MarshalIndent(schedule, "", "  ")
}

// Вспомогательные функции

type GroupPosition struct {
	Column    int // Колонка с номером группы / начало данных группы
	EndColumn int // Колонка окончания данных группы
}

// findGroupPositions находит позиции колонок с группами
func (s *ParserService) findGroupPositions(row []string) []GroupPosition {
	positions := make([]GroupPosition, 0)
	groupPattern := regexp.MustCompile(`^\d{5}$`)

	groupColumns := make([]int, 0)
	for i, cell := range row {
		cleaned := s.cleanValue(cell)
		if groupPattern.MatchString(cleaned) {
			groupColumns = append(groupColumns, i)
		}
	}

	// Определяем диапазоны для каждой группы
	for i, col := range groupColumns {
		endCol := len(row)
		if i+1 < len(groupColumns) {
			endCol = groupColumns[i+1]
		}
		positions = append(positions, GroupPosition{
			Column:    col,
			EndColumn: endCol,
		})
	}

	return positions
}

// extractDirection извлекает направление для группы
func (s *ParserService) extractDirection(row []string, startCol int, endCol int) string {
	if startCol >= len(row) {
		return ""
	}

	parts := make([]string, 0)

	// Собираем все непустые ячейки в диапазоне
	for i := startCol; i < endCol && i < len(row); i++ {
		cell := s.cleanValue(row[i])
		if cell != "" && !regexp.MustCompile(`^\d{5}$`).MatchString(cell) && !strings.Contains(strings.ToLower(cell), "направление") && !strings.Contains(strings.ToLower(cell), "группа") {
			parts = append(parts, cell)
		}
	}

	return strings.TrimSpace(strings.Join(parts, " "))
}

// parseDayCell парсит ячейку дня недели
func (s *ParserService) parseDayCell(cell string) (dayOfWeek, date string) {
	// Формат: "ПОНЕДЕЛЬНИК  17.11.2025"
	parts := strings.Fields(cell)
	if len(parts) >= 1 {
		dayOfWeek = strings.ToUpper(parts[0])
	}
	if len(parts) >= 2 {
		date = parts[len(parts)-1]
	}
	return
}

// parseLessons парсит все занятия для группы в диапазоне колонок
func (s *ParserService) parseLessons(row []string, startCol int, endCol int, time string) []models.Lesson {
	lessons := make([]models.Lesson, 0)

	if startCol >= len(row) {
		return lessons
	}

	// Ищем все дисциплины в диапазоне
	i := startCol
	for i < endCol && i < len(row) {
		cell := s.cleanValue(row[i])

		// Пропускаем пустые ячейки
		if cell == "" {
			i++
			continue
		}

		// Проверяем, это дисциплина (содержит скобки с типом занятия или длинный текст)
		if strings.Contains(cell, "(") || len(cell) > 10 {
			lesson := models.Lesson{
				Time: time,
			}

			// Парсим дисциплину
			lesson.Subject, lesson.Teacher, lesson.Type, lesson.SubGroup = s.parseDiscipline(cell)

			// Ищем аудиторию в следующих 1-3 колонках
			for offset := 1; offset <= 3 && i+offset < endCol && i+offset < len(row); offset++ {
				audCell := s.cleanValue(row[i+offset])
				if audCell != "" {
					// Аудитория обычно короткая и не содержит скобок
					if len(audCell) <= 5 || audCell == "с/з" {
						lesson.Classroom = audCell
						i += offset // Пропускаем обработанные ячейки
						break
					} else if strings.Contains(audCell, "(") {
						// Это следующая дисциплина, не аудитория
						break
					}
				}
			}

			// Добавляем урок только если есть предмет
			if lesson.Subject != "" {
				lessons = append(lessons, lesson)
			}
		}

		i++
	}

	return lessons
}

// parseDiscipline разбирает строку дисциплины
func (s *ParserService) parseDiscipline(text string) (subject, teacher, lessonType, subGroup string) {
	// Формат: "Математика (пр.) Гареева Г.А."
	// Или: "Иностранный язык (пр.) 1п/г Буланова Л.Н."
	// Или: "Физическая культура и спорт (элективная дисциплина) Телешев С.А."

	text = strings.TrimSpace(text)
	if text == "" {
		return
	}

	original := text

	// Извлекаем подгруппу
	if strings.Contains(text, "п/г") {
		re := regexp.MustCompile(`(\d+\s*п/г)`)
		matches := re.FindStringSubmatch(text)
		if len(matches) > 1 {
			subGroup = matches[1]
			text = strings.Replace(text, subGroup, "", 1)
		}
	}

	// Извлекаем тип занятия (лек., пр., лаб.)
	typePattern := regexp.MustCompile(`\((лек\.|пр\.|лаб\.)\)`)
	matches := typePattern.FindStringSubmatch(text)
	if len(matches) > 1 {
		lessonType = matches[1]
		text = typePattern.ReplaceAllString(text, "")
	}

	// Удаляем дополнительные описания в скобках (например, "элективная дисциплина")
	text = regexp.MustCompile(`\([^)]+\)`).ReplaceAllString(text, "")

	// Оставшееся: "Математика Гареева Г.А."
	text = strings.TrimSpace(text)
	parts := strings.Fields(text)
	if len(parts) == 0 {
		subject = original
		return
	}

	// ФИО обычно последние 2-3 слова с инициалами
	fioPattern := regexp.MustCompile(`[А-ЯЁ][а-яё]+\s+[А-ЯЁ]\.[А-ЯЁ]\.?`)
	fioMatch := fioPattern.FindString(text)

	if fioMatch != "" {
		teacher = strings.TrimSpace(fioMatch)
		subject = strings.TrimSpace(strings.Replace(text, fioMatch, "", 1))
	} else {
		// Если не нашли ФИО, всё считаем предметом
		subject = text
	}

	subject = strings.TrimSpace(subject)
	teacher = strings.TrimSpace(teacher)

	return
}

// extractWeekType извлекает тип недели
func (s *ParserService) extractWeekType(rows [][]string) string {
	if len(rows) < 3 {
		return ""
	}

	for i := 0; i < 5 && i < len(rows); i++ {
		for _, cell := range rows[i] {
			cell = strings.ToLower(s.cleanValue(cell))
			if strings.Contains(cell, "нечетная") {
				return "нечетная"
			}
			if strings.Contains(cell, "четная") {
				return "четная"
			}
		}
	}

	return "нечетная"
}

// extractSemester извлекает семестр
func (s *ParserService) extractSemester(rows [][]string) string {
	if len(rows) < 2 {
		return ""
	}

	for _, cell := range rows[0] {
		if strings.Contains(cell, "семестр") {
			re := regexp.MustCompile(`([IVX]+)\s*семестр`)
			matches := re.FindStringSubmatch(cell)
			if len(matches) > 1 {
				return matches[1]
			}
		}
	}

	return "I"
}

// extractAcademicYear извлекает учебный год
func (s *ParserService) extractAcademicYear(rows [][]string) string {
	if len(rows) < 5 {
		return ""
	}

	for i := 0; i < 5; i++ {
		for _, cell := range rows[i] {
			if strings.Contains(cell, "учебный год") {
				re := regexp.MustCompile(`(\d{4}/\d{4})`)
				matches := re.FindStringSubmatch(cell)
				if len(matches) > 1 {
					return matches[1]
				}
			}
		}
	}

	return ""
}

// cleanValue очищает значение ячейки
func (s *ParserService) cleanValue(value string) string {
	value = strings.TrimSpace(value)

	// Удаляем формулы вида ="текст"
	if strings.HasPrefix(value, "=") {
		value = strings.TrimPrefix(value, "=")
		value = strings.Trim(value, "\"")
	}

	value = strings.TrimSpace(value)

	// Пустые формулы возвращаем как пустую строку
	if value == "" || value == "\"\"" {
		return ""
	}

	return value
}

// parseReplacementSchedule парсит расписание замен
func (s *ParserService) parseReplacementSchedule(f *excelize.File) ([]byte, error) {
	sheet := f.GetSheetList()[0]
	rows, err := f.GetRows(sheet)
	if err != nil {
		return nil, err
	}

	schedule := models.ReplacementSchedule{
		Type:         "replacements",
		UpdatedAt:    time.Now(),
		Date:         time.Now().Format("2006-01-02"),
		Replacements: make([]models.Replacement, 0),
	}

	// Пропускаем заголовок и парсим данные
	for i := 2; i < len(rows); i++ {
		row := rows[i]
		if len(row) < 6 {
			continue
		}

		timeSlot := s.cleanValue(row[0])
		if timeSlot == "" {
			continue
		}

		schedule.Replacements = append(schedule.Replacements, models.Replacement{
			Time:            timeSlot,
			OriginalSubject: s.cleanValue(row[1]),
			NewSubject:      s.cleanValue(row[2]),
			OriginalTeacher: s.cleanValue(row[3]),
			NewTeacher:      s.cleanValue(row[4]),
			Classroom:       s.cleanValue(row[5]),
		})
	}

	return json.MarshalIndent(schedule, "", "  ")
}

// parseExamSchedule парсит экзаменационное расписание
func (s *ParserService) parseExamSchedule(f *excelize.File) ([]byte, error) {
	sheet := f.GetSheetList()[0]
	rows, err := f.GetRows(sheet)
	if err != nil {
		return nil, err
	}

	schedule := models.ExamSchedule{
		Type:      "exams",
		UpdatedAt: time.Now(),
		Exams:     make([]models.Exam, 0),
	}

	for i := 2; i < len(rows); i++ {
		row := rows[i]
		if len(row) < 5 {
			continue
		}

		date := s.cleanValue(row[0])
		if date == "" {
			continue
		}

		schedule.Exams = append(schedule.Exams, models.Exam{
			Date:      date,
			Time:      s.cleanValue(row[1]),
			Subject:   s.cleanValue(row[2]),
			Teacher:   s.cleanValue(row[3]),
			Classroom: s.cleanValue(row[4]),
		})
	}

	return json.MarshalIndent(schedule, "", "  ")
}

// ValidateScheduleFile валидирует структуру
func (s *ParserService) ValidateScheduleFile(file io.Reader, scheduleType string) (bool, error) {
	f, err := excelize.OpenReader(file)
	if err != nil {
		return false, fmt.Errorf("invalid xlsx file: %w", err)
	}
	defer f.Close()

	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		return false, fmt.Errorf("no sheets found")
	}

	rows, err := f.GetRows(sheets[0])
	if err != nil {
		return false, err
	}

	if len(rows) < 5 {
		return false, fmt.Errorf("file must contain at least 5 rows")
	}

	return true, nil
}
