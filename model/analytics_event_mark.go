package model

import "github.com/QuantumNous/new-api/common"

type AnalyticsEventMark struct {
	Id          int    `json:"id"`
	SubjectType string `json:"subject_type" gorm:"size:32;not null;uniqueIndex:idx_analytics_event_mark"`
	SubjectId   int    `json:"subject_id" gorm:"not null;uniqueIndex:idx_analytics_event_mark"`
	EventName   string `json:"event_name" gorm:"size:64;not null;uniqueIndex:idx_analytics_event_mark"`
	CreatedAt   int64  `json:"created_at" gorm:"bigint;not null"`
}

func TryMarkAnalyticsEvent(subjectType string, subjectID int, eventName string) bool {
	if subjectType == "" || subjectID <= 0 || eventName == "" {
		return false
	}
	mark := AnalyticsEventMark{
		SubjectType: subjectType,
		SubjectId:   subjectID,
		EventName:   eventName,
		CreatedAt:   common.GetTimestamp(),
	}
	err := DB.Create(&mark).Error
	return err == nil
}
