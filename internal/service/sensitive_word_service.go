package service

import (
	"strings"

	"review-view/internal/model"
	"review-view/internal/store"
)

type SensitiveWordService struct {
	store store.SensitiveWordStore
}

func NewSensitiveWordService(s store.SensitiveWordStore) *SensitiveWordService {
	return &SensitiveWordService{store: s}
}

func (s *SensitiveWordService) List() ([]model.SensitiveWord, error) {
	return s.store.List()
}

func (s *SensitiveWordService) Create(original, replacement string) (*model.SensitiveWord, error) {
	w := &model.SensitiveWord{Original: original, Replacement: replacement}
	return w, s.store.Create(w)
}

func (s *SensitiveWordService) Update(id int64, original, replacement string) (*model.SensitiveWord, error) {
	w := &model.SensitiveWord{ID: id, Original: original, Replacement: replacement}
	return w, s.store.Update(w)
}

func (s *SensitiveWordService) Delete(id int64) error {
	return s.store.Delete(id)
}

// Replace 将文本中所有敏感词替换为对应替换词
func (s *SensitiveWordService) Replace(text string) string {
	words, err := s.store.List()
	if err != nil || len(words) == 0 {
		return text
	}
	for _, w := range words {
		if w.Original != "" {
			text = strings.ReplaceAll(text, w.Original, w.Replacement)
		}
	}
	return text
}

// Restore 将文本中所有替换词还原为原始敏感词
func (s *SensitiveWordService) Restore(text string) string {
	words, err := s.store.List()
	if err != nil || len(words) == 0 {
		return text
	}
	for _, w := range words {
		if w.Replacement != "" {
			text = strings.ReplaceAll(text, w.Replacement, w.Original)
		}
	}
	return text
}
