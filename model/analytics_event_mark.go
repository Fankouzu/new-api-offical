package model

import (
	"errors"

	"github.com/QuantumNous/new-api/common"

	"gorm.io/gorm"
)

const (
	AnalyticsEventStatusPending = "pending"
	AnalyticsEventStatusSending = "sending"
	AnalyticsEventStatusSent    = "sent"
	AnalyticsEventStatusFailed  = "failed"
)

type AnalyticsEventMark struct {
	Id          int    `json:"id"`
	SubjectType string `json:"subject_type" gorm:"size:32;not null;uniqueIndex:idx_analytics_event_mark"`
	SubjectId   int    `json:"subject_id" gorm:"not null;uniqueIndex:idx_analytics_event_mark"`
	EventName   string `json:"event_name" gorm:"size:64;not null;uniqueIndex:idx_analytics_event_mark"`
	Status      string `json:"status" gorm:"size:16;not null;default:sent;index"`
	CreatedAt   int64  `json:"created_at" gorm:"bigint;not null"`
	UpdatedAt   int64  `json:"updated_at" gorm:"bigint;not null;default:0"`
}

func TryMarkAnalyticsEvent(subjectType string, subjectID int, eventName string) bool {
	return BeginAnalyticsEventDelivery(subjectType, subjectID, eventName) > 0
}

func BeginAnalyticsEventDelivery(subjectType string, subjectID int, eventName string) int {
	if subjectType == "" || subjectID <= 0 || eventName == "" {
		return 0
	}
	now := common.GetTimestamp()
	mark := AnalyticsEventMark{
		SubjectType: subjectType,
		SubjectId:   subjectID,
		EventName:   eventName,
		Status:      AnalyticsEventStatusSending,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	err := DB.Create(&mark).Error
	if err == nil {
		return mark.Id
	}

	var existing AnalyticsEventMark
	err = DB.Where("subject_type = ? AND subject_id = ? AND event_name = ?", subjectType, subjectID, eventName).First(&existing).Error
	if err != nil || existing.Status == AnalyticsEventStatusSent || existing.Status == AnalyticsEventStatusSending {
		return 0
	}
	result := DB.Model(&AnalyticsEventMark{}).
		Where("id = ? AND status IN ?", existing.Id, []string{AnalyticsEventStatusPending, AnalyticsEventStatusFailed}).
		Updates(map[string]interface{}{
			"status":     AnalyticsEventStatusSending,
			"updated_at": now,
		})
	if result.Error != nil || result.RowsAffected == 0 {
		return 0
	}
	return existing.Id
}

func MarkAnalyticsEventSent(id int) bool {
	return updateAnalyticsEventStatus(id, AnalyticsEventStatusSent)
}

func MarkAnalyticsEventFailed(id int) bool {
	return updateAnalyticsEventStatus(id, AnalyticsEventStatusFailed)
}

func updateAnalyticsEventStatus(id int, status string) bool {
	if id <= 0 || status == "" {
		return false
	}
	result := DB.Model(&AnalyticsEventMark{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":     status,
			"updated_at": common.GetTimestamp(),
		})
	return result.Error == nil && result.RowsAffected > 0
}

func GetAnalyticsEventMark(subjectType string, subjectID int, eventName string) (*AnalyticsEventMark, error) {
	if subjectType == "" || subjectID <= 0 || eventName == "" {
		return nil, gorm.ErrRecordNotFound
	}
	var mark AnalyticsEventMark
	err := DB.Where("subject_type = ? AND subject_id = ? AND event_name = ?", subjectType, subjectID, eventName).First(&mark).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	return &mark, err
}
